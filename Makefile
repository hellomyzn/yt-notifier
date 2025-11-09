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
