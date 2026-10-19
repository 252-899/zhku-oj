# ZK Code Arena Server Makefile

.PHONY: help build run dev stop clean test lint docker-build docker-run

# 默认目标
help:
	@echo "ZK Code Arena Server 可用命令:"
	@echo ""
	@echo "  make build        - 构建项目"
	@echo "  make run          - 运行项目"
	@echo "  make dev          - 开发模式运行（热更新）"
	@echo "  make stop         - 停止服务"
	@echo "  make clean        - 清理构建文件"
	@echo "  make test         - 运行测试"
	@echo "  make lint         - 代码检查"
	@echo "  make swagger      - 生成 Swagger 文档"
	@echo "  make docker-build - 构建 Docker 镜像"
	@echo "  make docker-run   - 使用 Docker 运行"
	@echo "  make docker-dev   - 使用 Docker 开发环境"
	@echo "  make docker-stop  - 停止 Docker 服务"
	@echo "  make setup        - 初始化开发环境"

# 构建项目
build:
	@echo "🔨 构建项目..."
	go build -o bin/zk-code-arena-server cmd/*.go

# 运行项目
run: build
	@echo "🚀 运行项目..."
	./bin/zk-code-arena-server run

# 开发模式运行
dev:
	@echo "🔧 开发模式运行..."
	@chmod +x scripts/dev.sh
	./scripts/dev.sh

# 停止服务
stop:
	@echo "🛑 停止服务..."
	@chmod +x scripts/stop.sh
	./scripts/stop.sh

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	rm -rf bin/
	rm -rf tmp/
	go clean

# 运行测试
test:
	@echo "🧪 运行测试..."
	go test -v ./...

# 代码检查
lint:
	@echo "🔍 代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "请安装 golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 构建 Docker 镜像
docker-build:
	@echo "🐳 构建 Docker 镜像..."
	docker build -t zk-code-arena-server .

# 使用 Docker 运行
docker-run:
	@echo "🐳 使用 Docker 运行..."
	@chmod +x scripts/start.sh
	./scripts/start.sh

# 使用 Docker 开发环境
docker-dev:
	@echo "🐳 使用 Docker 开发环境..."
	@chmod +x scripts/start.sh
	./scripts/start.sh dev

# 停止 Docker 服务
docker-stop:
	@echo "🛑 停止 Docker 服务..."
	@chmod +x scripts/stop.sh
	./scripts/stop.sh

# 初始化开发环境
setup:
	@echo "⚙️ 初始化开发环境..."
	@if ! command -v go >/dev/null 2>&1; then \
		echo "❌ 请先安装 Go 1.23+"; \
		exit 1; \
	fi
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "❌ 请先安装 Docker"; \
		exit 1; \
	fi
	@echo "📦 下载依赖..."
	go mod download
	@echo "🔧 安装 Air..."
	# 使用兼容 Go 1.23 的版本
	go install github.com/air-verse/air@v1.49.0
	@echo "✅ 开发环境初始化完成"

# 安装依赖
deps:
	@echo "📦 安装依赖..."
	go mod download
	go mod tidy

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	go fmt ./...

# 生成 Swagger 文档
swagger:
	@echo "📚 生成 Swagger 文档..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/main.go -o docs --parseDependency --parseInternal; \
		echo "✅ Swagger 文档已生成"; \
	else \
		echo "请先安装 swag: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi
swagger-dev:
	@echo "📚 生成 Swagger 文档..."
	@if command -v swag >/dev/null 2>&1; then \
        cp cmd/main.go tmp/main.go; \
  		sed -i.bak 's|// @host .*|// @host      localhost:8081|' cmd/main.go; \
  		swag init -g cmd/main.go -o docs --parseDependency --parseInternal; \
  		mv cmd/main.go.bak cmd/main.go; \
    	echo "✅ Swagger 文档已生成"; \
    else \
    	echo "请先安装 swag: go install github.com/swaggo/swag/cmd/swag@latest"; \
    fi

# 生成 godoc 文档
docs:
	@echo "📚 生成文档..."
	@if command -v godoc >/dev/null 2>&1; then \
		godoc -http=:6060; \
	else \
		echo "请安装 godoc: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# 检查代码质量
check: fmt lint test
	@echo "✅ 代码质量检查完成"
