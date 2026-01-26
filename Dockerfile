# 多阶段构建 Dockerfile
FROM golang:1.25.3-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make

WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件（只包含 PostgreSQL 支持）
RUN go build -tags=rdb_psql -o combinator cmd/main.go

# 运行时镜像
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/combinator .

# 暴露端口
EXPOSE 8899

# 启动命令
ENTRYPOINT ["/app/combinator"]
CMD ["-c", "/config/config.json", "-l", "0.0.0.0:8899"]
