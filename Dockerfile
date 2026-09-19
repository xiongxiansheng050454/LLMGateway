FROM golang:1.26-alpine AS build

WORKDIR /src/server

ENV GOPROXY=https://goproxy.cn,direct

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/llmgateway ./cmd/llmgateway

FROM node:22-alpine AS dashboard-build

WORKDIR /src/dashboard-react
COPY dashboard-react/package.json dashboard-react/package-lock.json ./
RUN npm ci
COPY dashboard-react/ ./
# DocsPage imports these Markdown files as Vite raw assets during the image build.
COPY docs/ /src/docs/
RUN npm run build

FROM alpine:3.22

RUN addgroup -S llmgateway && adduser -S -G llmgateway llmgateway

WORKDIR /app

COPY --from=build /out/llmgateway /app/llmgateway
COPY --chown=llmgateway:llmgateway server/db/migrations /app/migrations
COPY --from=dashboard-build --chown=llmgateway:llmgateway /src/dashboard-react/dist /app/dashboard
COPY --chown=llmgateway:llmgateway docs /app/dashboard/docs

USER llmgateway

ENV ADDR=:8080 \
    DASHBOARD_DIR=/app/dashboard \
    MIGRATIONS_DIR=/app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/llmgateway"]
