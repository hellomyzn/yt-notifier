include .env
export $(shell sed 's/=.*//' .env)

DOCKER_COMPOSE_FILE_PATH=-f docker-compose.yml

ENV_PATH=--env-file .env
PROJECT_FLAG=--project-name ${PROJECT_NAME}_devcontainer
USER_FLAG=--user ${USER_NAME}
COVERAGE_DIR=${WORKDIR}/src/test/coverage
TEST_WITH_COVERAGE=go test -v -cover -coverprofile=${COVERAGE_DIR}/coverage.out . 
CREATE_COVERAGE_HTML=go tool cover -html=${COVERAGE_DIR}/coverage.out -o ${COVERAGE_DIR}/coverage.html

up:
	@echo "VSCode の Dev Container から起動してください"
down:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} down
destroy:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} down --rmi all --volumes --remove-orphans
exec:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} exec ${USER_FLAG} workspace bash
logs:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} logs -f
dev:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c 'cd ${WORKDIR}/src && go run ./cmd/server'
tidy:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c 'cd ${WORKDIR}/src && go mod tidy'
test:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c 'cd ${WORKDIR}/src && go test -v'
test-cover:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c 'cd ${WORKDIR}/src && ${TEST_WITH_COVERAGE} && ${CREATE_COVERAGE_HTML}' \
		&& open ../src/test/coverage/coverage.html
build:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} exec ${USER_FLAG} workspace bash -c 'cd ${WORKDIR}/src && go build ./...'
mysql:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} exec mysql \
		mysql -h 127.0.0.1 -P ${MYSQL_PORT} -u${MYSQL_USER} -p${MYSQL_PASSWORD} ${MYSQL_DATABASE}
psql:
	docker compose ${DOCKER_COMPOSE_FILE_PATH} ${PROJECT_FLAG} ${ENV_PATH} exec postgres \
		psql -h 127.0.0.1 -p ${POSTGRES_PORT} -U ${POSTGRES_USER} -d ${POSTGRES_DB}



