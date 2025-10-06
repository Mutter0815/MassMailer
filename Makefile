.PHONY: up down logs migrate resetdb

compose := deployments/docker-compose.yml
envfile := deployments/.env

# загружаем значения из .env (без лишних строк и комментов)
include $(envfile)
export $(shell sed 's/=.*//' $(envfile))

up:
	docker compose -f $(compose) up -d --build

down:
	docker compose -f $(compose) down -v

logs:
	docker compose -f $(compose) logs -f

migrate:
	docker compose -f $(compose) exec -T postgres \
		psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -v ON_ERROR_STOP=1 \
		-f /migrations/0001_init.sql

resetdb:
	docker compose -f $(compose) exec -T postgres \
		psql -U $(POSTGRES_USER) -d postgres -c "DROP DATABASE IF EXISTS $(POSTGRES_DB);"
	docker compose -f $(compose) exec -T postgres \
		psql -U $(POSTGRES_USER) -d postgres -c "CREATE DATABASE $(POSTGRES_DB);"
	$(MAKE) migrate
