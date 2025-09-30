SHELL := /bin/bash

compose := docker compose -f deployments/docker-compose.yml --env-file deployments/.env

.PHONY: up down logs migrate curl

up:
	$(compose) up -d --build

down:
	$(compose) down -v

logs:
	$(compose) logs -f

migrate:
	docker exec -i $$(docker ps -qf name=postgres) \
	  psql -U $$(grep POSTGRES_USER deployments/.env | cut -d= -f2) \
	      -d $$(grep POSTGRES_DB deployments/.env | cut -d= -f2) \
	      -f /migrations/0001_init.sql
