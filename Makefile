include .env
export $(shell sed 's/=.*//' .env)

DOCKER_COMPOSE_FILE_PATH=-f docker-compose.yml

ENV_PATH=--env-file .env
USER_FLAG=--user ${USER_NAME}
CMD=go run ./cmd/job

up:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} up -d --build
down:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} down
destroy:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} down --rmi all --volumes --remove-orphans
exec:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} exec ${USER_FLAG} workspace bash
logs:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} logs -f
build:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} build --no-cache --force-rm
mysql:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} exec mysql \
		mysql -h 127.0.0.1 -P ${MYSQL_PORT} -u${MYSQL_USER} -p${MYSQL_PASSWORD} ${MYSQL_DATABASE}
psql:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} exec postgres \
		psql -h 127.0.0.1 -p ${POSTGRES_PORT} -U ${POSTGRES_USER} -d ${POSTGRES_DB}
run:
	@make up
	@echo "Waiting for container to be ready..."
	@sleep 2
	docker-compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c "${CMD}"
	@make down

# テスト・カバレッジ（cmd/job/ は DI 配線のみのため除外）
COVERAGE_DIR=src/test/coverage
COVERAGE_OUT=$(COVERAGE_DIR)/coverage.out
COVERAGE_HTML=$(COVERAGE_DIR)/coverage.html
TEST_PKGS=./internal/...

test:
	@make up
	@echo "Waiting for container to be ready..."
	@sleep 2
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c "go test $(TEST_PKGS)"
	@make down

test-v:
	@make up
	@echo "Waiting for container to be ready..."
	@sleep 2
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c "go test -v $(TEST_PKGS)"
	@make down

coverage:
	@make up
	@echo "Waiting for container to be ready..."
	@sleep 2
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c \
		"mkdir -p $(COVERAGE_DIR) && go test -coverprofile=$(COVERAGE_OUT) -covermode=atomic $(TEST_PKGS) && go tool cover -func=$(COVERAGE_OUT)"
	@make down

coverage-html:
	@make up
	@echo "Waiting for container to be ready..."
	@sleep 2
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c \
		"mkdir -p $(COVERAGE_DIR) && go test -coverprofile=$(COVERAGE_OUT) -covermode=atomic $(TEST_PKGS) && go tool cover -func=$(COVERAGE_OUT) && go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)"
	@make down
	@echo "Report: $(COVERAGE_HTML)"
