COMPOSE = docker compose -f deploy/compose/compose.yaml

.PHONY: bootstrap up down logs token test verify

bootstrap:
	./scripts/bootstrap.sh

up: bootstrap
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f --tail=100

token:
	@cat .local/operator.token

test:
	go test ./...

verify: up
	./scripts/verify-e2e.sh
