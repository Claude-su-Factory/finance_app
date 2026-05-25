package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrChatSessionNotFound = errors.New("chat session not found")
var ErrUsageExceeded = errors.New("usage limit exceeded")

// Spec §5 사용량 한도 (MVP).
const (
	LimitChatPerMonth = 30
	LimitInputTokens  = 50_000
	LimitOutputTokens = 10_000
	LimitOpusPerMonth = 1
)

type ChatRepo interface {
	CreateSession(ctx context.Context, userID, title string) (*models.ChatSession, error)
	ListSessions(ctx context.Context, userID string) ([]models.ChatSession, error)
	DeleteSession(ctx context.Context, userID, sessionID string) error
	ListMessages(ctx context.Context, userID, sessionID string) ([]models.ChatMessage, error)
	AppendMessage(ctx context.Context, sessionID string, m models.ChatMessage) (*models.ChatMessage, error)
	MarkFinished(ctx context.Context, messageID string) error
	UnfinishedInSession(ctx context.Context, userID, sessionID string) (*models.ChatMessage, error)

	GetUsage(ctx context.Context, userID string) (*models.ChatUsageMonthly, error)
	IncrementUsage(ctx context.Context, userID string, inTok, outTok int, isOpus bool) error
	CheckLimits(ctx context.Context, userID string, isOpus bool) error
}

type PgChatRepo struct {
	pool *pgxpool.Pool
}

func NewPgChatRepo(pool *pgxpool.Pool) *PgChatRepo { return &PgChatRepo{pool: pool} }

func (r *PgChatRepo) CreateSession(ctx context.Context, userID, title string) (*models.ChatSession, error) {
	row := r.pool.QueryRow(ctx, `
		insert into public.chat_sessions (user_id, title) values ($1, $2)
		returning id::text, user_id::text, title, created_at, updated_at
	`, userID, title)
	var s models.ChatSession
	if err := row.Scan(&s.ID, &s.UserID, &s.Title, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PgChatRepo) ListSessions(ctx context.Context, userID string) ([]models.ChatSession, error) {
	rows, err := r.pool.Query(ctx, `
		select id::text, user_id::text, title, created_at, updated_at
		from public.chat_sessions where user_id = $1 order by updated_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ChatSession
	for rows.Next() {
		var s models.ChatSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.Title, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PgChatRepo) DeleteSession(ctx context.Context, userID, sessionID string) error {
	ct, err := r.pool.Exec(ctx, `delete from public.chat_sessions where id = $1 and user_id = $2`, sessionID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrChatSessionNotFound
	}
	return nil
}

func (r *PgChatRepo) ListMessages(ctx context.Context, userID, sessionID string) ([]models.ChatMessage, error) {
	// 단일 쿼리 — session 소유권을 JOIN으로 한 번에 검증.
	rows, err := r.pool.Query(ctx, `
		select m.id::text, m.session_id::text, m.role, m.content, m.tool_calls,
		       m.input_tokens, m.output_tokens, m.model, m.finished_at, m.created_at
		from public.chat_messages m
		join public.chat_sessions s on s.id = m.session_id
		where s.id = $1 and s.user_id = $2
		order by m.created_at
	`, sessionID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		var tool *json.RawMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &tool, &m.InputTokens, &m.OutputTokens, &m.Model, &m.FinishedAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		if tool != nil {
			m.ToolCalls = *tool
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 결과 0건이면 소유권 검증 (404 vs 빈 리스트 구분)
	if len(out) == 0 {
		var exists bool
		if err := r.pool.QueryRow(ctx, `select exists(select 1 from public.chat_sessions where id = $1 and user_id = $2)`, sessionID, userID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrChatSessionNotFound
		}
	}
	return out, nil
}

func (r *PgChatRepo) AppendMessage(ctx context.Context, sessionID string, m models.ChatMessage) (*models.ChatMessage, error) {
	var tool any
	if len(m.ToolCalls) > 0 {
		tool = []byte(m.ToolCalls)
	}
	row := r.pool.QueryRow(ctx, `
		insert into public.chat_messages
		  (session_id, role, content, tool_calls, input_tokens, output_tokens, model, finished_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		returning id::text, created_at
	`, sessionID, m.Role, m.Content, tool, m.InputTokens, m.OutputTokens, m.Model, m.FinishedAt)
	if err := row.Scan(&m.ID, &m.CreatedAt); err != nil {
		return nil, err
	}
	m.SessionID = sessionID
	// 세션 updated_at 갱신
	_, _ = r.pool.Exec(ctx, `update public.chat_sessions set updated_at = now() where id = $1`, sessionID)
	return &m, nil
}

func (r *PgChatRepo) MarkFinished(ctx context.Context, messageID string) error {
	_, err := r.pool.Exec(ctx, `update public.chat_messages set finished_at = now() where id = $1`, messageID)
	return err
}

func (r *PgChatRepo) UnfinishedInSession(ctx context.Context, userID, sessionID string) (*models.ChatMessage, error) {
	row := r.pool.QueryRow(ctx, `
		select m.id::text, m.session_id::text, m.role, m.content, m.tool_calls,
		       m.input_tokens, m.output_tokens, m.model, m.finished_at, m.created_at
		from public.chat_messages m
		join public.chat_sessions s on s.id = m.session_id
		where s.id = $1 and s.user_id = $2 and m.role = 'assistant' and m.finished_at is null
		order by m.created_at desc limit 1
	`, sessionID, userID)
	var m models.ChatMessage
	var tool *json.RawMessage
	if err := row.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &tool, &m.InputTokens, &m.OutputTokens, &m.Model, &m.FinishedAt, &m.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if tool != nil {
		m.ToolCalls = *tool
	}
	return &m, nil
}

func (r *PgChatRepo) GetUsage(ctx context.Context, userID string) (*models.ChatUsageMonthly, error) {
	ym := time.Now().Format("2006-01")
	row := r.pool.QueryRow(ctx, `
		select user_id::text, year_month, chat_count, input_tokens, output_tokens, opus_count
		from public.chat_usage_monthly where user_id = $1 and year_month = $2
	`, userID, ym)
	var u models.ChatUsageMonthly
	if err := row.Scan(&u.UserID, &u.YearMonth, &u.ChatCount, &u.InputTokens, &u.OutputTokens, &u.OpusCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &models.ChatUsageMonthly{UserID: userID, YearMonth: ym}, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *PgChatRepo) IncrementUsage(ctx context.Context, userID string, inTok, outTok int, isOpus bool) error {
	ym := time.Now().Format("2006-01")
	opusDelta := 0
	if isOpus {
		opusDelta = 1
	}
	_, err := r.pool.Exec(ctx, `
		insert into public.chat_usage_monthly (user_id, year_month, chat_count, input_tokens, output_tokens, opus_count)
		values ($1, $2, 1, $3, $4, $5)
		on conflict (user_id, year_month) do update set
		  chat_count = chat_usage_monthly.chat_count + 1,
		  input_tokens = chat_usage_monthly.input_tokens + excluded.input_tokens,
		  output_tokens = chat_usage_monthly.output_tokens + excluded.output_tokens,
		  opus_count = chat_usage_monthly.opus_count + excluded.opus_count
	`, userID, ym, inTok, outTok, opusDelta)
	return err
}

func (r *PgChatRepo) CheckLimits(ctx context.Context, userID string, isOpus bool) error {
	u, err := r.GetUsage(ctx, userID)
	if err != nil {
		return err
	}
	if u.ChatCount >= LimitChatPerMonth {
		return fmt.Errorf("%w: monthly chat count %d/%d", ErrUsageExceeded, u.ChatCount, LimitChatPerMonth)
	}
	if u.InputTokens >= LimitInputTokens {
		return fmt.Errorf("%w: input tokens %d/%d", ErrUsageExceeded, u.InputTokens, LimitInputTokens)
	}
	if u.OutputTokens >= LimitOutputTokens {
		return fmt.Errorf("%w: output tokens %d/%d", ErrUsageExceeded, u.OutputTokens, LimitOutputTokens)
	}
	if isOpus && u.OpusCount >= LimitOpusPerMonth {
		return fmt.Errorf("%w: opus count %d/%d", ErrUsageExceeded, u.OpusCount, LimitOpusPerMonth)
	}
	return nil
}
