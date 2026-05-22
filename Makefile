.PHONY: dev api web db-up db-down db-reset migrate test lint

dev:
	@echo "Run 'make api' and 'make web' in separate terminals"

api:
	cd apps/api && air

web:
	cd apps/web && npm run dev

db-up:
	cd supabase && supabase start

db-down:
	cd supabase && supabase stop

db-reset:
	cd supabase && supabase db reset

migrate:
	cd supabase && supabase db push

test:
	cd apps/api && go test ./...
	cd apps/web && npm test

lint:
	cd apps/api && golangci-lint run
	cd apps/web && npm run lint
