# 构建阶段
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 安装 swag 工具并生成 Swagger 文档
RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/main.go -o docs --parseDependency --parseInternal

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd

# 运行阶段
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone

# 创建非root用户
RUN adduser -D -s /bin/sh appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 复制配置文件
COPY --from=builder /app/conf ./conf

# 复制 Swagger 文档
COPY --from=builder /app/docs ./docs

# 创建日志目录
RUN mkdir -p logs && chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
CMD ["./main", "run"]
