include .env
export

.PHONY: db-up db-down db-psql run health test

db-up:
	docker compose up -d

db-down:
	docker compose down

db-psql:
	docker compose exec db psql -U $(DB_USER) -d $(DB_NAME)

run:
	cd api && go run .

health:
	@curl -s -o /dev/null -w "%{http_code}\n" http://localhost:$(PORT)/health

test:
	cd api && go test ./...