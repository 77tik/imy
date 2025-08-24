override PROJECT_NAME := imy
override APP_NAME := $(PROJECT_NAME)-backend
override SERVER_NAME := imy

.DEFAULT_GOAL: help
.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":[^:]*?## "}; {printf "\033[38;5;69m%-30s\033[38;5;38m %s\033[0m\n", $$1, $$2}'

.PHONY: sql
sql:  ## Generate sql related codes
	@echo "Generating SQL-related code locally..."
	@mkdir -p bin
	@go build -o ./bin/dbgen ./cmd/dbgen
	@./bin/dbgen || true

.PHONY: api
api:  ## Generate api related codes
	@echo "Generating API-related code locally..."
	@cd api && make || true

# ----------------------------
# Redis Utilities
# ----------------------------
override REDIS_CONTAINER ?= imy-redis
override REDIS_AUTH ?= 123456
override REDIS_CLI := docker exec -i $(REDIS_CONTAINER) redis-cli -a $(REDIS_AUTH) --no-auth-warning

.PHONY: redis-cli
redis-cli: ## 进入 Redis CLI（容器内）
	@docker exec -it $(REDIS_CONTAINER) redis-cli -a $(REDIS_AUTH)

.PHONY: redis-keys
redis-keys: ## 列出 Redis 中的键，支持 PATTERN='prefix:*'（默认 *）
	@PATTERN=$${PATTERN:-*}; \
	$(REDIS_CLI) --scan --pattern "$$PATTERN" | head -n 200

.PHONY: redis-get
redis-get: ## 查看指定键的值（自动识别类型），用法：make redis-get KEY=your_key
	@test -n "$(KEY)" || (echo "用法: make redis-get KEY=your_key" && exit 1)
	@echo "TYPE: " && $(REDIS_CLI) TYPE "$(KEY)"
	@echo "TTL:  " && $(REDIS_CLI) TTL  "$(KEY)"
	@echo "----- VALUE -----"
	@t=$$($(REDIS_CLI) TYPE "$(KEY)"); \
	if [ "$$t" = "hash" ]; then $(REDIS_CLI) HGETALL "$(KEY)"; \
	elif [ "$$t" = "list" ]; then $(REDIS_CLI) LRANGE "$(KEY)" 0 -1; \
	elif [ "$$t" = "set" ]; then $(REDIS_CLI) SMEMBERS "$(KEY)"; \
	elif [ "$$t" = "zset" ]; then $(REDIS_CLI) ZRANGE "$(KEY)" 0 -1 WITHSCORES; \
	else $(REDIS_CLI) GET "$(KEY)"; fi
