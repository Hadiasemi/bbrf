# Makefile for BBRF project

.PHONY: help build test release release-patch release-minor release-major

# Get current version
CURRENT_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
VERSION_PARTS := $(subst ., ,$(subst v,,$(CURRENT_VERSION)))
MAJOR := $(word 1,$(VERSION_PARTS))
MINOR := $(word 2,$(VERSION_PARTS))
PATCH := $(word 3,$(VERSION_PARTS))

# Calculate new versions
NEW_PATCH := v$(MAJOR).$(MINOR).$(shell echo $$(($(PATCH) + 1)))
NEW_MINOR := v$(MAJOR).$(shell echo $$(($(MINOR) + 1))).0
NEW_MAJOR := v$(shell echo $$(($(MAJOR) + 1))).0.0

# Repository info
REPO := $(shell git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building bbrf..."
	@go build -o bbrf .

test: ## Run tests
	@echo "Running tests..."
	@go test ./...

check-clean: ## Check if working directory is clean
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Working directory is not clean:"; \
		git status --short; \
		exit 1; \
	fi

release-patch: check-clean ## Release a new patch version (0.0.X)
	@echo "Current version: $(CURRENT_VERSION)"
	@echo "New version: $(NEW_PATCH)"
	@$(MAKE) do-release VERSION=$(NEW_PATCH)

release-minor: check-clean ## Release a new minor version (0.X.0)
	@echo "Current version: $(CURRENT_VERSION)"
	@echo "New version: $(NEW_MINOR)"
	@$(MAKE) do-release VERSION=$(NEW_MINOR)

release-major: check-clean ## Release a new major version (X.0.0)
	@echo "Current version: $(CURRENT_VERSION)"
	@echo "New version: $(NEW_MAJOR)"
	@$(MAKE) do-release VERSION=$(NEW_MAJOR)

release: ## Release with custom version (use VERSION=v1.2.3)
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Please specify VERSION (e.g., make release VERSION=v1.2.3)"; \
		exit 1; \
	fi
	@$(MAKE) check-clean
	@$(MAKE) do-release VERSION=$(VERSION)

do-release: ## Internal: perform the actual release
	@echo "🚀 Starting release $(VERSION)..."
	
	# Confirm release
	@read -p "Continue with release $(VERSION)? (y/N): " confirm && [ "$$confirm" = "y" ]
	
	# Pull latest changes
	@echo "📥 Pulling latest changes..."
	@git pull origin main
	
	# Run tests
	@echo "🧪 Running tests..."
	@go test ./...
	
	# Build to ensure everything works
	@echo "🔨 Building..."
	@go build .
	
	# Create and push tag
	@echo "🏷️  Creating tag $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	
	@echo "📤 Pushing tag to origin..."
	@git push origin $(VERSION)
	
	# Trigger Go proxy update
	@echo "🔄 Triggering Go proxy update..."
	@curl -s "https://proxy.golang.org/$(REPO)/@latest" > /dev/null || true
	@sleep 2
	@curl -s "https://proxy.golang.org/$(REPO)/@v/$(VERSION).info" > /dev/null || true
	

status: ## Show current version and repository info
	@echo "Current version: $(CURRENT_VERSION)"
	@echo "Repository: $(REPO)"
	@echo "Next patch: $(NEW_PATCH)"
	@echo "Next minor: $(NEW_MINOR)"
	@echo "Next major: $(NEW_MAJOR)"
