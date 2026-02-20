default: build

BINARY     = terraform-provider-openclaw
HOSTNAME   = registry.terraform.io
NAMESPACE  = openclaw
TYPE       = openclaw
VERSION   ?= 0.1.0
OS_ARCH   ?= linux_amd64

build:
	go build -o $(BINARY)

install: build
	mkdir -p ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(TYPE)/$(VERSION)/$(OS_ARCH)
	cp $(BINARY) ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(TYPE)/$(VERSION)/$(OS_ARCH)/

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./... -v -timeout 120m

lint:
	golangci-lint run

fmt:
	gofmt -s -w .

generate:
	go generate ./...

clean:
	rm -f $(BINARY)

# ── Docker-based integration testing ─────────────────────────
# Spins up an OpenClaw gateway in Docker and runs tests against it.

docker-test: ## Run file-mode acceptance tests in Docker (no gateway needed)
	./docker/test.sh test

docker-test-ws: ## Run WS-mode tests against a Dockerized gateway
	./docker/test.sh test-ws

docker-test-all: ## Run all acceptance tests in Docker (file + WS)
	./docker/test.sh test-all

docker-apply: ## Terraform apply the test-stack against a Dockerized gateway
	./docker/test.sh apply

docker-plan: ## Terraform plan the test-stack against a Dockerized gateway
	./docker/test.sh plan

docker-shell: ## Interactive shell with provider + terraform (gateway running)
	./docker/test.sh shell

docker-down: ## Tear down Docker test environment
	./docker/test.sh down

docker-logs: ## Tail gateway container logs
	./docker/test.sh logs

.PHONY: build install test testacc lint fmt generate clean \
        docker-test docker-test-ws docker-test-all docker-apply docker-plan \
        docker-shell docker-down docker-logs
