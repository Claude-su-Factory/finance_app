package portfolio

import (
	"context"
	"sort"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/models"
)

// EquityComputer는 paper_transactions 시계열 + 시점별 가격/환율로
// 일자별 KRW 평가액을 계산한다.
//
// 알고리즘:
//
//	t시점까지의 transactions를 누적해 cash + holdings 상태 재구성 (replay)
//	각 holding의 t일 가격 × 환율 × 수량 → KRW 환산
//	equity_t = cash_t + sum(market_value_krw_t)
type EquityComputer struct {
	deps EquityDeps
}

type EquityDeps interface {
	TradingDays(ctx context.Context, pool db.Executor, since, until time.Time) ([]string, error)
	InstrumentClosesOnDates(ctx context.Context, pool db.Executor, instrumentID string, dates []string) (map[string]float64, error)
	FxRatesOnDates(ctx context.Context, pool db.Executor, currency string, dates []string) (map[string]float64, error)
}

// NewEquityComputer는 paper_equity 전용. deps는 alpha의 PgDeps 재사용 가능.
func NewEquityComputer(deps EquityDeps) *EquityComputer {
	return &EquityComputer{deps: deps}
}

// Compute는 사용자 paper account의 평가액 시계열을 계산.
// account.CreatedAt 이전 시점은 데이터 없음 — since = max(period start, account.CreatedAt).
func (c *EquityComputer) Compute(ctx context.Context, pool db.Executor,
	account *models.PaperAccount, transactions []models.PaperTransaction,
	currentHoldings []models.PaperHolding, periodDays int) ([]models.EquityPoint, error) {

	if periodDays <= 0 {
		periodDays = 90
	}
	// KST 자정 기준 — 한국 사용자 시점 기준으로 trading day가 일관됨.
	kst, _ := time.LoadLocation("Asia/Seoul")
	now := time.Now().In(kst)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, kst)
	since := today.AddDate(0, 0, -periodDays)
	createdDay := account.CreatedAt.In(kst).Truncate(24 * time.Hour)
	if since.Before(createdDay) {
		since = createdDay
	}

	tradingDays, err := c.deps.TradingDays(ctx, pool, since, today)
	if err != nil {
		return nil, err
	}
	if len(tradingDays) == 0 {
		return []models.EquityPoint{}, nil
	}

	// 현재 보유 + transactions에 등장한 모든 instrument 종합
	instrumentSet := map[string]string{} // id → currency
	for _, h := range currentHoldings {
		instrumentSet[h.InstrumentID] = h.Currency
	}
	for _, t := range transactions {
		instrumentSet[t.InstrumentID] = t.Currency
	}

	priceByInst := map[string]map[string]float64{}
	for iid := range instrumentSet {
		m, err := c.deps.InstrumentClosesOnDates(ctx, pool, iid, tradingDays)
		if err != nil {
			return nil, err
		}
		priceByInst[iid] = m
	}

	fxByCur := map[string]map[string]float64{"KRW": {}}
	for _, cur := range instrumentSet {
		if cur == "KRW" || fxByCur[cur] != nil {
			continue
		}
		m, err := c.deps.FxRatesOnDates(ctx, pool, cur, tradingDays)
		if err != nil {
			return nil, err
		}
		fxByCur[cur] = m
	}

	// transactions를 created_at 순으로 정렬 (replay 위해)
	txSorted := make([]models.PaperTransaction, len(transactions))
	copy(txSorted, transactions)
	sort.Slice(txSorted, func(i, j int) bool {
		if !txSorted[i].CreatedAt.Equal(txSorted[j].CreatedAt) {
			return txSorted[i].CreatedAt.Before(txSorted[j].CreatedAt)
		}
		return txSorted[i].ID < txSorted[j].ID
	})

	// 시계열 replay: 각 trading day d에서
	// since 시점부터 d시점까지의 transactions 적용 → cash + holdings 상태 → equity_d 계산
	out := make([]models.EquityPoint, 0, len(tradingDays))
	txIdx := 0
	cash := account.InitialCash
	holdingsByInst := map[string]float64{} // instrument_id → quantity

	for _, d := range tradingDays {
		// dayEnd = d일 KST 자정의 다음날 0시 (즉 d일 KST 24:00)
		dayDate, _ := time.ParseInLocation("2006-01-02", d, kst)
		dayEnd := dayDate.Add(24 * time.Hour)
		// d일(KST) 자정 이전까지 발생한 모든 active transaction 적용
		for txIdx < len(txSorted) && txSorted[txIdx].CreatedAt.Before(dayEnd) {
			t := txSorted[txIdx]
			delta := t.Quantity
			if t.Action == "sell" {
				delta = -delta
			}
			holdingsByInst[t.InstrumentID] += delta
			// cash 변화
			if t.Action == "buy" {
				cash -= t.TotalKRW
			} else {
				cash += t.TotalKRW
			}
			txIdx++
		}

		// d시점 holdings 평가
		equity := cash
		for iid, qty := range holdingsByInst {
			if qty <= 0 {
				continue
			}
			price, ok := priceByInst[iid][d]
			if !ok {
				continue // 가격 없으면 평가에서 제외
			}
			cur := instrumentSet[iid]
			fx := 1.0
			if cur != "KRW" {
				fx = lookupFxForward(fxByCur[cur], tradingDays, indexOf(tradingDays, d))
			}
			equity += qty * price * fx
		}
		out = append(out, models.EquityPoint{Date: d, EquityKRW: equity})
	}

	return out, nil
}

func indexOf(dates []string, target string) int {
	for i, d := range dates {
		if d == target {
			return i
		}
	}
	return 0
}
