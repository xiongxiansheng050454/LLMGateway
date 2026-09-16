# Backend Structure

后端目录按职责分层，避免后续功能继续堆在 `internal/handler`。

```text
cmd/llmgateway/              进程入口，只做装配：config -> store -> handler -> http.Server
internal/config/             环境变量配置读取，集中管理默认值
internal/handler/            HTTP 路由、请求解析、响应封装、Dashboard 静态托管
internal/domain/             API 与业务共享类型，不依赖 HTTP 或数据库
internal/store/              存储接口（Store）与通用存储错误
internal/store/memory/       默认内存实现，用于 MVP、测试和未接入 PG 前的功能迭代
internal/store/postgres/     PostgreSQL Store 实现（查询接线见子 issue）
internal/db/migrate/         最小迁移 runner，按文件名顺序应用 db/migrations/*.sql
internal/db/sqlc/            sqlc 生成代码输出目录，不手写业务逻辑
db/migrations/               PostgreSQL schema 迁移 SQL
db/queries/                  sqlc 查询 SQL
deployments/                 本地开发部署配置，如 PostgreSQL docker compose
dashboard/                   静态前端控制台
```

## 约定

- HTTP handler 只依赖 `internal/store.Store` 接口，不直接访问 PostgreSQL 或 sqlc。
- 业务/API 共享结构放在 `internal/domain`，避免 handler、memory store、postgres store 互相引用具体实现。
- 当前默认仍使用 `internal/store/memory`，后续接入 PG 时在 `internal/store/postgres` 实现同一个 `store.Store` 接口。
- sqlc 查询写在 `db/queries/*.sql`，schema 写在 `db/migrations/*.sql`，生成代码输出到 `internal/db/sqlc`。
- 不要手改 `internal/db/sqlc` 生成文件；修改 SQL 后运行 `sqlc generate`。
- 初始 schema 覆盖渠道、模型映射、定价、用户、余额、Key、限流、用量日志和日汇总，后续 issue 应优先扩展现有表而不是新建重复概念。
- 进程启动时按 `DATABASE_URL` 选择实现：未设置使用 memory，设置则建立 pgxpool 连接并选用 PostgreSQL store。
- PostgreSQL store 当前对未接线方法返回 `store.ErrNotImplemented`（HTTP 501），具体查询由 PostgreSQL store 子 issue 实现，避免静默返回空数据。
- `cmd/llmgateway` 使用 `http.Server` 并在收到 `SIGINT`/`SIGTERM` 后优雅关闭。

## 本地 PostgreSQL

```powershell
docker compose -f deployments/docker-compose.postgres.yml up -d
```

默认开发库：

```text
postgres://llmgateway:llmgateway_dev@localhost:5432/llmgateway?sslmode=disable
```

后端环境变量：

```text
ADDR=:8080
DASHBOARD_DIR=dashboard
DATABASE_URL=postgres://llmgateway:llmgateway_dev@localhost:5432/llmgateway?sslmode=disable
MIGRATIONS_DIR=db/migrations
```

未设置 `DATABASE_URL` 时使用内存 store，可直接启动：

```powershell
go run ./cmd/llmgateway
```

## 迁移

本仓库选择**最小自建 runner**（`internal/db/migrate`），基于 pgx，不引入额外迁移依赖。

- 迁移文件为 `db/migrations/*.sql`，按文件名（版本前缀）字典序执行。
- 每个文件在独立事务中执行，并在 `schema_migrations(version)` 中记录，已执行版本会跳过。
- 设置 `DATABASE_URL` 时，进程启动会自动执行 `MIGRATIONS_DIR`（默认 `db/migrations`）下的待执行迁移；失败时启动报错退出。

## sqlc

```powershell
sqlc generate
```

配置文件：`sqlc.yaml`
