GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BUMP ?= patch

PROVIDER_NAME := terraform-provider-hetzner-robot
PROVIDER_PATH := registry.terraform.io/sophotechlabs/hetzner-robot/0.0.1/$(GOOS)_$(GOARCH)

ifeq ($(GOOS),darwin)
	INSTALL_PATH := $(HOME)/Library/Application Support/io.terraform/plugins/$(PROVIDER_PATH)
else
	INSTALL_PATH := $(HOME)/.local/share/terraform/plugins/$(PROVIDER_PATH)
endif

.PHONY: default build test testacc lint vet fmt fmtcheck generate dev docs ci release

default: testacc

build:
	go build -v .

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

fmtcheck:
	@sh -c 'files=$$(gofmt -s -l .); if [ -n "$$files" ]; then echo "Files need formatting:"; echo "$$files"; exit 1; fi'

generate:
	go generate ./...

dev: build
	@mkdir -p "$(INSTALL_PATH)"
	@cp $(PROVIDER_NAME) "$(INSTALL_PATH)/$(PROVIDER_NAME)"

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name hrobot

ci: vet fmtcheck test

release:
	@LATEST=$$(gh release list --limit 1 --json tagName --jq '.[0].tagName' 2>/dev/null | sed 's/^v//'); \
	if [ -z "$$LATEST" ]; then LATEST="0.0.0"; fi; \
	IFS='.' read -r MAJOR MINOR PATCH <<< "$$LATEST"; \
	case "$(BUMP)" in \
		patch) PATCH=$$((PATCH + 1));; \
		minor) MINOR=$$((MINOR + 1)); PATCH=0;; \
		major) MAJOR=$$((MAJOR + 1)); MINOR=0; PATCH=0;; \
		*) echo "ERROR: BUMP must be patch, minor, or major"; exit 1;; \
	esac; \
	VERSION="$$MAJOR.$$MINOR.$$PATCH"; \
	echo "Latest: v$$LATEST → Next: v$$VERSION ($(BUMP) bump)"; \
	gh workflow run release.yml -f version=$$VERSION
