BIN = $(GOPATH)/bin
BASE = $(GOPATH)/src/$(PACKAGE)
PKGS = go list ./... | grep -v "^vendor/"

# Tools
## Testing library
GINKGO = $(BIN)/ginkgo
$(BIN)/ginkgo:
	go get -u github.com/onsi/ginkgo/ginkgo

## Migration tool
GOOSE = $(BIN)/goose
$(BIN)/goose:
	go get -u -d github.com/pressly/goose/cmd/goose
	go build -tags='no_mysql no_sqlite' -o $(BIN)/goose github.com/pressly/goose/cmd/goose

## Source linter
LINT = $(BIN)/golint
$(BIN)/golint:
	go get -u golang.org/x/lint/golint

## Combination linter
METALINT = $(BIN)/gometalinter.v2
$(BIN)/gometalinter.v2:
	go get -u gopkg.in/alecthomas/gometalinter.v2
	$(METALINT) --install


.PHONY: installtools
installtools: | $(LINT) $(GOOSE) $(GINKGO)
	echo "Installing tools"

.PHONY: metalint
metalint: | $(METALINT)
	$(METALINT) ./... --vendor \
	--fast \
	--exclude="exported (function)|(var)|(method)|(type).*should have comment or be unexported" \
	--format="{{.Path.Abs}}:{{.Line}}:{{if .Col}}{{.Col}}{{end}}:{{.Severity}}: {{.Message}} ({{.Linter}})"

.PHONY: lint
lint:
	$(LINT) $$($(PKGS)) | grep -v -E "exported (function)|(var)|(method)|(type).*should have comment or be unexported"

#Database
HOST_NAME = localhost
PORT = 5432
NAME =
USER = postgres
CONNECT_STRING=postgresql://$(USER)@$(HOST_NAME):$(PORT)/$(NAME)?sslmode=disable

#Test
TEST_DB = vulcanize_testing
TEST_CONNECT_STRING = postgresql://$(USER)@$(HOST_NAME):$(PORT)/$(TEST_DB)?sslmode=disable

.PHONY: test
test: | $(GINKGO) $(LINT)
	go vet ./...
	go fmt ./...
	dropdb --if-exists $(TEST_DB)
	createdb $(TEST_DB)
	$(GOOSE) -dir db/migrations postgres "$(TEST_CONNECT_STRING)" up
	$(GOOSE) -dir db/migrations postgres "$(TEST_CONNECT_STRING)" reset
	make migrate NAME=$(TEST_DB)
	$(GINKGO) -r --skipPackage=integration_tests,integration

.PHONY: integrationtest
integrationtest: | $(GINKGO) $(LINT)
	go vet ./...
	go fmt ./...
	dropdb --if-exists $(TEST_DB)
	createdb $(TEST_DB)
	$(GOOSE) -dir db/migrations "$(TEST_CONNECT_STRING)" up
	$(GOOSE) -dir db/migrations "$(TEST_CONNECT_STRING)" reset
	make migrate NAME=$(TEST_DB)
	$(GINKGO) -r integration_test/

build:
	go fmt ./...
	GO111MODULE=on go build

## Build docker image
.PHONY: docker-build
docker-build:
	docker build -t vulcanize/ipld-eth-indexer -f dockerfiles/Dockerfile .

# Parameter checks
## Check that DB variables are provided
.PHONY: checkdbvars
checkdbvars:
	test -n "$(HOST_NAME)" # $$HOST_NAME
	test -n "$(PORT)" # $$PORT
	test -n "$(NAME)" # $$NAME
	@echo $(CONNECT_STRING)

## Check that the migration variable (id/timestamp) is provided
.PHONY: checkmigration
checkmigration:
	test -n "$(MIGRATION)" # $$MIGRATION

# Check that the migration name is provided
.PHONY: checkmigname
checkmigname:
	test -n "$(NAME)" # $$NAME

# Migration operations
## Rollback the last migration
.PHONY: rollback
rollback: $(GOOSE) checkdbvars
	$(GOOSE) -dir db/migrations postgres "$(CONNECT_STRING)" down
	pg_dump -O -s $(CONNECT_STRING) > db/schema.sql


## Rollback to a select migration (id/timestamp)
.PHONY: rollback_to
rollback_to: $(GOOSE) checkmigration checkdbvars
	$(GOOSE) -dir db/migrations postgres "$(CONNECT_STRING)" down-to "$(MIGRATION)"

## Apply all migrations not already run
.PHONY: migrate
migrate: $(GOOSE) checkdbvars
	$(GOOSE) -dir db/migrations postgres "$(CONNECT_STRING)" up
	pg_dump -O -s $(CONNECT_STRING) > db/schema.sql

## Create a new migration file
.PHONY: new_migration
new_migration: $(GOOSE) checkmigname
	$(GOOSE) -dir db/migrations create $(NAME) sql

## Check which migrations are applied at the moment
.PHONY: migration_status
migration_status: $(GOOSE) checkdbvars
	$(GOOSE) -dir db/migrations postgres "$(CONNECT_STRING)" status

# Convert timestamped migrations to versioned (to be run in CI);
# merge timestamped files to prevent conflict
.PHONY: version_migrations
version_migrations:
	$(GOOSE) -dir db/migrations fix

# Import a psql schema to the database
.PHONY: import
import:
	test -n "$(NAME)" # $$NAME
	psql $(NAME) < db/schema.sql
