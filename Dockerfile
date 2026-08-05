ARG GO_VERSION=1.26.5
FROM golang:${GO_VERSION}-alpine AS builder

ARG MAIN_PATH=main.go
WORKDIR /app

# 提前复制 go.mod 和 go.sum，以利用缓存
COPY go.mod go.sum ./

# ✅ 缓存依赖目录，加速 go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# 再复制全部代码
COPY . .

# ✅ 构建可复用缓存的构建命令
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o alertmanager-agent -ldflags="-s -w" ${MAIN_PATH}

FROM alpine:3.23.5
WORKDIR /app
COPY --from=builder /app/alertmanager-agent .
ENTRYPOINT ["./alertmanager-agent"]
EXPOSE 8080
