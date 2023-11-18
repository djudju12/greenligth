include .env

## help: show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N]' && read ans && [ $${ans:-N} = y  ]

.PHONY: run/api
## run/api: run the cmd/api application
run/api:
	go run ./cmd/api -db-dsn=${GREENLIGHT_DB_DSN} -smtp-username=${SMTP_USERNAME} -smtp-password=${SMTP_PASSWORD}

.PHONY: db/psql
## db/psql: connect to the database using psql
db/psql:
	psql ${GREENLIGHT_DB_DSN}

.PHONY: db/local/build
## db/build: create a new database
db/local/build: confirm
	./set_db.sh

.PHONY: db/migration/new
## db/migration/new name=$1: create a new database migration
db/migration/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

.PHONY: db/migrations/up
## db/migration/up: apply all up database migrations
db/migrations/up: confirm
	@echo 'Running up migrations...'
	migrate -path ./migrations -database ${GREENLIGHT_DB_DSN} up

git_description = $(shell git describe --always --dirty --tags --long)
current_time = $(shell date --iso-8601=seconds)
linker_flags = '-s -X main.buildTime=$(current_time) -X main.version=${git_description}'

.PHONY: build/api
## build/api: build the binaries for the application
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags=${linker_flags} -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./bin/linux_amd64/api ./cmd/api

.PHONY: run/tests/integration
## Run all tests, including units
run/tests/integration:
	go test ./... -tags=integration -db-dsn=$(shell ./set_test_db.sh | tail -n 1)

.PHONY: audit
## audit: tidy and vendor dependencies and format, vet and test all code
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

.PHONY: vendor
## vendor: tidy and vendor dependencies
vendor:
	@echo 'Tidying and verifying module depencecies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor
