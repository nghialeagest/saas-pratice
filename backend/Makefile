.PHONY: proto build run test db-up db-down tidy

proto:
	buf generate

build:
	go build ./...

run:
	go run .

test:
	TEST_DATABASE_DSN="postgres://postgres:postgres@localhost:55432/saas?sslmode=disable" go test ./... -p 1 -count=1

db-up:
	docker run -d --rm --name saasbase-pg \
		-e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=saas \
		-p 55432:5432 postgres:18-bookworm

db-down:
	docker stop saasbase-pg

tidy:
	go mod tidy
