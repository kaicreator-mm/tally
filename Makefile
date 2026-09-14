.PHONY: help build export-src docscheck repohealth test test-runtime test-integration check lint fmt migrate reindex sqlc dev-db proto

# 默认用本地 Postgres;并行开发请为每个实例覆盖成独立端口/库(见 tasks/README.md 并行规约)。
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/tally?sslmode=disable
export DATABASE_URL

help:
	@grep -E '^[a-z-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-18s %s\n",$$1,$$2}'

build: ## 编译三个二进制
	go build -o bin/server ./cmd/server
	go build -o bin/worker ./cmd/worker
	go build -o bin/tally  ./cmd/tally

export-src: ## 生成确定性源码导出包(P6-06:钉定 commit + denylist 扫描,输出到 bin/export/)
	go run ./scripts/export

docscheck: ## 检查 CURRENT 文档是否回流旧实现状态
	go run ./scripts/docscheck

repohealth: ## 拒绝被 Git 跟踪的运行态/构建产物/凭据形文件
	go run ./scripts/repohealth

fmt: ## 格式化
	gofmt -w .

lint: ## 静态检查
	gofmt -l . | tee /dev/stderr | (! read)
	go vet ./...
	golangci-lint run

test: ## 单元测试 + 架构测试(不需要 PostgreSQL/真实外部 runtime)
	go test ./test/arch/...
	go test ./internal/...
	go test ./cmd/...
	go test ./scripts/...


test-runtime: ## Coding Runtime / conformance 测试；显式 live gate 另行配置
	go test ./runtimes/...
	go test ./test/runtimeconformance/...

test-integration: migrate ## 集成测试(需要可达的 Postgres:本地或 make dev-db)
	go test -tags=integration ./...

check: docscheck repohealth lint ## 提交前跑这个；外部环境集成测试单独运行
	go test ./test/arch/...
	go test ./internal/...
	go test ./cmd/...
	go test ./runtimes/...
	go test ./test/runtimeconformance/...
	go test ./scripts/...

sqlc: ## 从 SQL 生成 Go
	sqlc generate

migrate: ## 跑迁移(幂等;经 cmd/tally 用 goose 库,不依赖 goose CLI)
	go run ./cmd/tally migrate

reindex: ## drop 索引面并从 git + 事实面重算
	go run ./cmd/tally reindex

dev-db: ## (可选)用 docker 起一个 Postgres;已装本地 Postgres 可跳过
	docker compose -f docker-compose.dev.yml up -d

proto: ## 在浏览器里看界面原型
	@echo "打开 web/prototype/index.html"
	@python3 -m http.server 8081 --directory web/prototype
