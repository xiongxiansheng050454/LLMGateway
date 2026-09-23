FROM golang:1.26-alpine AS build

WORKDIR /src/server

ENV GOPROXY=https://goproxy.cn,direct

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/llmgateway ./cmd/llmgateway

FROM alpine:3.22

RUN addgroup -S llmgateway && adduser -S -G llmgateway llmgateway

WORKDIR /app

COPY --from=build /out/llmgateway /app/llmgateway
COPY --chown=llmgateway:llmgateway server/db/migrations /app/migrations

USER llmgateway

ENV ADDR=:8080 \
    MIGRATIONS_DIR=/app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/llmgateway"]
