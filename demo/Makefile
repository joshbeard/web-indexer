# Demo Makefile for web-indexer
# Run 'make help' for a list of targets.

.DEFAULT_GOAL := help

.PHONY: help
help: ## Shows this help for demo targets
	@echo "Web-Indexer Demo Targets:"
	@egrep -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: demo
demo: ## Generate and serve a live demo (local only)
	@echo "🚀 Generating and serving comprehensive demo..."
	@if [ -n '$(CUSTOM_DEMOS)' ]; then \
		echo "  Including custom demos: $(CUSTOM_DEMOS)"; \
		go run demo.go -serve -custom-demos '$(CUSTOM_DEMOS)'; \
	else \
		go run demo.go -serve; \
	fi

.PHONY: demo-local
demo-local: ## Generate and serve a local-only demo
	@echo "🚀 Generating local demo..."
	@if [ -n '$(CUSTOM_DEMOS)' ]; then \
		echo "  Including custom demos: $(CUSTOM_DEMOS)"; \
		go run demo.go -type local -serve -custom-demos '$(CUSTOM_DEMOS)'; \
	else \
		go run demo.go -type local -serve; \
	fi

.PHONY: demo-s3
demo-s3: ## Generate S3 demo
	@echo "🚀 Generating S3 demo..."
	@if [ -n '$(CUSTOM_DEMOS)' ]; then \
		echo "  Including custom demos: $(CUSTOM_DEMOS)"; \
		go run demo.go -type s3 -custom-demos '$(CUSTOM_DEMOS)'; \
	else \
		go run demo.go -type s3; \
	fi

.PHONY: demo-s3-release
demo-s3-release: ## Generate S3 demo for release preview
	@echo "🚀 Generating S3 release preview demo..."
	@if [ -z "$(DEMO_S3_BUCKET)" ]; then \
		echo "❌ DEMO_S3_BUCKET environment variable is required"; \
		exit 1; \
	fi
	@echo "  Target bucket: $(DEMO_S3_BUCKET)"
	@echo "  Release version: $(RELEASE_VERSION)"
	go run demo.go -type s3-release

.PHONY: demo-custom
demo-custom: ## Generate custom demo with config file (e.g., make demo-custom CONFIG=my-config.yml)
	@echo "🚀 Generating demo with custom config: $(CONFIG)"
	@if [ -z "$(CONFIG)" ]; then \
		echo "❌ Please specify CONFIG parameter (e.g., make demo-custom CONFIG=my-config.yml)"; \
		echo "💡 Tip: Copy config.yml to create your own configuration"; \
		exit 1; \
	fi
	go run demo.go -config "$(CONFIG)" -serve

.PHONY: demo-with-custom
demo-with-custom: ## Generate demos with custom args (e.g., make demo-with-custom CUSTOM="--theme nord --title 'Test'")
	@echo "🚀 Generating demos with custom args: $(CUSTOM)"
	@if [ -z "$(CUSTOM)" ]; then \
		echo "❌ Please specify CUSTOM parameter"; \
		echo "💡 Examples:"; \
		echo "  make demo-with-custom CUSTOM=\"--theme nord --title 'My Demo'\""; \
		echo "  make demo-with-custom CUSTOM=\"nord-demo:--theme nord;dracula-demo:--theme dracula\""; \
		exit 1; \
	fi
	go run demo.go -custom-demos "$(CUSTOM)" -serve

.PHONY: clean
clean: ## Clean up demo files and tracked S3 buckets
	@echo "🧹 Cleaning up demo files..."
	go run demo.go -cleanup

.PHONY: clean-s3-all
clean-s3-all: ## Delete all s3 buckets prefixed with configured bucket name
	@echo "🧹 Cleaning up S3 buckets..."
	@BUCKET_PREFIX=$$(grep "bucket_prefix:" config.yml | sed 's/.*bucket_prefix: *"\([^"]*\)".*/\1/' | tr -d '"'); \
	if [ -z "$$BUCKET_PREFIX" ]; then \
		echo "❌ Could not find bucket_prefix in config.yml"; \
		exit 1; \
	fi; \
	echo "🔍 Searching for buckets with prefix: $$BUCKET_PREFIX"; \
	BUCKETS=$$(aws s3api list-buckets --query "Buckets[?starts_with(Name, '$$BUCKET_PREFIX')].Name" --output text 2>/dev/null || echo ""); \
	if [ -z "$$BUCKETS" ]; then \
		echo "✅ No buckets found with prefix $$BUCKET_PREFIX"; \
	else \
		echo "Found buckets:"; \
		echo "$$BUCKETS" | tr '\t' '\n' | sed 's/^/  - /'; \
		echo ""; \
		echo "⚠️  WARNING: This will delete ALL buckets with prefix '$$BUCKET_PREFIX'"; \
		read -p "Are you sure? Type 'yes' to continue: " confirm; \
		if [ "$$confirm" = "yes" ]; then \
			echo "$$BUCKETS" | tr '\t' '\n' | while read bucket; do \
				if [ -n "$$bucket" ]; then \
					echo "🗑️  Deleting bucket: $$bucket"; \
					aws s3 rm s3://$$bucket --recursive 2>/dev/null || true; \
					aws s3 rb s3://$$bucket 2>/dev/null || true; \
				fi; \
			done; \
			echo "✅ Cleanup completed"; \
		else \
			echo "❌ Cleanup cancelled"; \
		fi; \
	fi