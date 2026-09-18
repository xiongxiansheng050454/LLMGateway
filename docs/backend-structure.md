# Backend Structure

后端目录按业务能力纵向组织，目录本身体现 catalog、accounts、usage、ratelimit、proxy 等高内聚模块边界。

```text
server/                             Go 模块根（go.mod / go.sum / sqlc.yaml）
server/cmd/llmgateway/              进程入口与 HTTP 路由表（router.go）：config -> store -> httpapi -> http.Server
server/internal/config/             环境变量配置读取，集中管理默认值
server/internal/catalog/            目录、渠道、模型映射、定价、渠道连通性测试及其管理能力
server/internal/accounts/           用户、余额、网关 Key、认证上下文及权限能力
server/internal/usage/              用量日志、审计查询和统计能力
server/internal/ratelimit/          限流规则管理和运行时限流能力
server/internal/httpapi/            顶层 HTTP 装配、响应 envelope、Dashboard 和业务入口委托
server/internal/httpcommon/         共享 HTTP 请求解析、路径、分页、存储错误映射和删除响应 helper 的唯一归属
server/internal/proxy/              下游代理业务：OpenAI 适配、路由、计费、限流、熔断、结算和上游调用
server/internal/proxy/openai/       OpenAI 兼容 wire DTO 与协议适配；归属 proxy 业务模块
server/internal/domain/             API 与业务共享类型，不依赖 HTTP 或数据库
server/internal/money/              定点金额（int64 最小单位）解析与格式化
server/internal/crypto/             渠道 api_key 加解密、网关 Key 生成与哈希
server/internal/store/              存储接口（Store）与通用存储错误
server/internal/store/postgres/     唯一生产 Store 实现
server/internal/testutil/storefake/ 不需要数据库的测试专用 fake，生产代码不得导入
server/internal/db/migrate/         最小迁移 runner，按文件名顺序应用 server/db/migrations/*.sql
server/internal/db/sqlc/            sqlc 生成代码输出目录，不手写业务逻辑
server/db/migrations/               PostgreSQL schema 迁移 SQL（SQL 资产）
server/db/queries/                  sqlc 查询 SQL（SQL 资产）
deployments/                        本地开发部署配置，如 PostgreSQL docker compose
dashboard/                          静态前端控制台（仓库根，由 server 通过 ../dashboard 托管）
```

Go 模块路径为 `LLMGateway/server`；Go 命令需在 `server/` 目录下执行（或在仓库根使用 `go -C server ...`）。

两个 `db` 目录职责不同，不要混淆：

- `server/db/`：SQL 资产（`migrations/` 迁移、`queries/` sqlc 查询），由 `sqlc.yaml` 读取。
- `server/internal/db/`：Go 包（`migrate/` 迁移 runner、`sqlc/` 生成代码），由 Go 代码导入。

`CHANNEL_KEY_ENCRYPTION_KEY` 环境变量名常量位于 `server/internal/config`（env 解析职责）；`server/internal/crypto` 只负责密钥长度/算法校验，不再定义 env 常量。

## 约定

- 业务模块按能力纵向组织：`catalog`、`accounts`、`usage`、`ratelimit`、`proxy` 各自聚合规则、端口使用和 HTTP 入口契约；这不是按 HTTP/store/protocol 的横向分层。
- `server/internal/httpapi` 只负责顶层 HTTP 装配、通用响应和委托，不作为跨业务 admin 文件集中地。
- 业务模块只依赖 `server/internal/store.Store` 接口，不直接访问 PostgreSQL 或 sqlc。
- 代理编排位于 `server/internal/proxy`，依赖 `store.Store` 端口、`domain`、`money`、`crypto` 和协议中立的代理 contract，不直接访问 PostgreSQL 或 sqlc；OpenAI wire 转换只在 `server/internal/proxy/openai` 完成。
- 业务/API 共享结构放在 `server/internal/domain`，避免 httpapi、proxy 与 postgres store 互相引用具体实现。
- OpenAI 兼容 JSON wire type 放在 `server/internal/proxy/openai`；`catalog`、`accounts`、`usage`、`ratelimit`、`domain` 与 `store` 不依赖 OpenAI 协议 DTO。
- PostgreSQL 是唯一运行时存储；`DATABASE_URL` 与 `CHANNEL_KEY_ENCRYPTION_KEY` 均为必填配置，缺失或非法时进程启动失败。
- sqlc 查询写在 `server/db/queries/*.sql`，schema 写在 `server/db/migrations/*.sql`，生成代码输出到 `server/internal/db/sqlc`。
- 不要手改 `server/internal/db/sqlc` 生成文件；修改 SQL 后运行 `sqlc generate`。
- 初始 schema 覆盖渠道、模型映射、定价、用户、余额、Key、限流和用量日志，后续 issue 应优先扩展现有表而不是新建重复概念。
- 统计接口（overview/daily/channels）在 `usage_logs` 上实时聚合，按 UTC 自然日分组；`daily_usage_stats` 因未被使用且复合主键无法表达全局日汇总，已在迁移 `000004_drop_daily_usage_stats.sql` 中删除。
- 进程启动时建立 pgxpool 连接、执行迁移并装配 PostgreSQL store，不提供无数据库运行模式。
- PostgreSQL store 已实现渠道/模型/定价、用户/余额/Key、限流规则、用量日志与代理结算的持久化行为。
- `server/cmd/llmgateway` 使用 `http.Server` 并在收到 `SIGINT`/`SIGTERM` 后优雅关闭。
- 金额能力集中在 `server/internal/money`，密钥能力集中在 `server/internal/crypto`；store 与测试 fake 均复用这些能力，禁止重复实现金额解析。
- 禁止用 `float64` 参与计费；金额在 DB 用 `NUMERIC`，在 Go 用定点整数，对外输出字符串。
- 渠道 `api_key` 落库为密文，网关 Key 只存哈希；任何响应、日志、错误都不得出现明文密钥。
- `server/internal/money` 与 `server/internal/crypto` 为叶子包，不得依赖 `server/internal/store` 或 `server/internal/httpapi`。
- `server/internal/crypto` 的渠道密钥加密密钥来自环境变量 `CHANNEL_KEY_ENCRYPTION_KEY`（原始字节，长度 16/24/32）；缺失或非法时返回错误，禁止明文回退。
- 网关 Key 仅保存 `server/internal/crypto.HashKey` 的哈希，明文 `full_key` 只在创建/重置时返回一次。
- PostgreSQL 以 `api_key_ciphertext` 保存渠道密钥；测试 fake 仅存在于 `internal/testutil`，不得作为生产持久化实现。
- 共享 HTTP parsing/response glue 由 `server/internal/httpcommon` 统一持有；业务模块只保留领域相关的请求分派，顶层 HTTP 路由仍由 `server/cmd/llmgateway/router.go` 统一装配。
- HTTP 路由表（路径到入口的映射）集中在 `server/cmd/llmgateway/router.go`；`server/internal/httpapi` 只提供入口方法，不构造 mux。

## 文件组织约定

目录保持较浅层级：包内按领域拆文件，仅在协议与存储实现处使用子包，使目录能直接呈现模块边界。

- 每个包内按领域命名文件，禁止把多个领域堆进同一个文件：
  - `server/internal/domain/`：`common.go`、`channel.go`、`user.go`、`usage.go`、`ratelimit.go`、`channelhealth.go`、`failurereason.go`；纯规则（熔断状态机、限流规范化 `NormalizeRateLimit`、时间校验 `Validate*`、`FailureReason`）归位此处
  - `server/internal/store/`：`store.go`（错误别名 + 组合接口）、`channel.go`、`user.go`、`usage.go`、`ratelimit.go`、`channelhealth.go`（仅端口接口）；`CanonicalJSON` 为序列化一致性辅助，非业务规则
  - `server/internal/store/postgres/`：`postgres.go`（结构体/构造函数）、`channel.go`、`user.go`、`usage.go`、`ratelimit.go`、`channelhealth.go`
  - `server/internal/testutil/storefake/`：测试专用 Store fake，仅供测试夹具使用
  - `server/cmd/llmgateway/`：`main.go`（装配与优雅关闭）、`router.go`（唯一 HTTP 路由表）
  - `server/internal/catalog/`：渠道、模型映射、定价和渠道连通性测试业务模块（HTTP 入口由顶层装配）
  - `server/internal/accounts/`：用户、余额、网关 Key 和身份业务模块（HTTP 入口由顶层装配）
  - `server/internal/usage/`：用量日志、审计和统计业务模块（HTTP 入口由顶层装配）
  - `server/internal/ratelimit/`：限流规则和运行时限流业务模块（HTTP 入口由顶层装配）
  - `server/internal/httpapi/`：`handler.go`（顶层入口/分派/响应/静态托管）、`openai.go`（/v1 分派与错误映射）
  - `server/internal/httpcommon/`：共享 HTTP 请求解析、路径解析、分页、存储错误映射和删除响应 helper；这些通用行为只在此处实现
- `server/internal/proxy/`：`proxy.go`（编排依赖装配与代理错误）、`contracts.go`（协议中立请求/响应/usage contract）、`auth.go`（认证）、`routing.go`（选路）、`billing.go`（计费）、`ratelimit.go`（限流）、`orchestration.go`（代理编排）
    - `server/internal/proxy/openai/`：`types.go`、`adapter.go`（OpenAI 兼容 wire DTO、请求解析和响应适配；proxy 业务模块的协议边界）
- `store.Store` 由领域子接口组合而成，禁止继续往 `store.go` 堆方法：

```go
type Store interface {
    ChannelStore
    // UserStore / UsageStore / RateLimitStore 由对应 issue 加入
}
```

- PostgreSQL 实现以编译期断言固定其端口契约：

```go
var _ store.ChannelStore = (*postgres.Store)(nil)
```

- 领域文件边界与 `server/db/queries/*.sql` 的领域划分保持一致（channels/models/pricing/users/rate_limits/usage_logs）。
- 拆分与移动只做等价搬迁，不得顺手改变路由、响应结构、状态码或错误语义。

## 下游代理（/v1）

- `GET /v1/models` 与 `POST /v1/chat/completions` 由 `server/internal/httpapi/openai.go` 暴露 HTTP 入口、方法校验、body 读取、客户端 IP 提取、SSE write/flush 和 OpenAI 错误响应映射；代理业务编排及其 `openai` 协议适配位于 `server/internal/proxy`。
- 认证使用 `Authorization: Bearer <gateway-key>`；密钥经 `server/internal/crypto.HashKey` 后查询，明文不落日志/响应。
- 路由候选按 `priority` 越大越优先，同级内按 `weight` 加权随机；非正余额渠道被排除。
- 计费：缓存 token 已包含在 `prompt_tokens` 中，仅按 `(prompt_tokens - cached_tokens)` 计输入价，缓存部分计缓存价，避免重复计费。
- 成功结算：非流式 chat completion 成功后通过 store 级 `SettleChatCompletion` 端口统一处理用户扣费、可扣费渠道余额扣减与 success usage log。PostgreSQL 实现在单一事务中提交；`last_used_at` 仍为成功响应后的 best-effort 更新。
- 流式结算：`stream=true` 时网关强制向上游请求 `stream_options.include_usage=true`，逐事件重写 public model 并 flush；首个合法 JSON data 帧记录 TTFT。只有同时收到 usage 与 `[DONE]` 后才执行一次原子结算并下发终止帧。缺 usage、缺 `[DONE]`、畸形帧或中途断流均不扣费，写明确 error usage log，并通过流内 OpenAI error 帧结束；客户端取消会传播到上游且不计渠道失败。
- 限流：`rpm` + `reject` 规则基于 `usage_logs` 统计最近 1 分钟请求次数。`global`/`user`/`api_key` 保持按当前用户/Key 计数；`model` 规则额外按 public model 精确过滤；`channel` 规则在路由选中最终渠道后、调用上游前评估，超限直接返回 429 且写入 `error_code=rate_limited` 的 error usage log，不自动改选其他渠道。
- 熔断：每个渠道有 `channel_health` 状态（closed/open/half-open）。连续失败达阈值（默认 5）或确定性失败（上游 401/403/402）立即 open；冷却（默认 30s）后惰性转为 half-open 允许探测，探测成功回 closed、失败回 open。`ListRouteCandidates` 排除 open 渠道；当无可用渠道（无映射或全部 open）时返回 `503 no_healthy_channel`（错误码由 `no_available_channel` 变更而来，同时覆盖这两种情况）。失败分类仅计入传输错误、上游 429/401/403/402 与 5xx，其余 4xx 透传且不计渠道失败。健康记录为 best-effort。
- 已知限制（后续 issue 处理）：
  - 未配置 `model_pricing` 的渠道×模型按 cost=0 放行（建议为所有可路由模型配置定价）。
  - `queue` 动作未实现。

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
DASHBOARD_DIR=../dashboard
DATABASE_URL=postgres://llmgateway:llmgateway_dev@localhost:5432/llmgateway?sslmode=disable
CHANNEL_KEY_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef
MIGRATIONS_DIR=db/migrations
```

路径均相对于运行目录 `server/`：`DASHBOARD_DIR` 默认 `../dashboard`，`MIGRATIONS_DIR` 默认 `db/migrations`。

启动前必须设置 PostgreSQL URL 和 16/24/32 字节的渠道密钥加密密钥：

```powershell
cd server
$env:DATABASE_URL="postgres://llmgateway:llmgateway_dev@localhost:5432/llmgateway?sslmode=disable"
$env:CHANNEL_KEY_ENCRYPTION_KEY="0123456789abcdef0123456789abcdef"
go run ./cmd/llmgateway
```

普通 `go test ./...` 使用 `internal/testutil/storefake` 运行单元与 HTTP 契约测试；PostgreSQL 集成测试在设置 `TEST_DATABASE_URL` 后启用，未设置时会明确跳过。

## 迁移

本仓库选择**最小自建 runner**（`server/internal/db/migrate`），基于 pgx，不引入额外迁移依赖。

- 迁移文件为 `server/db/migrations/*.sql`，按文件名（版本前缀）字典序执行。
- 每个文件在独立事务中执行，并在 `schema_migrations(version)` 中记录，已执行版本会跳过。
- 进程启动会使用必填的 `DATABASE_URL` 自动执行 `MIGRATIONS_DIR`（默认 `db/migrations`）下的待执行迁移；失败时启动报错退出。

## sqlc

在 `server/` 目录下执行：

```powershell
cd server
sqlc generate
```

配置文件：`server/sqlc.yaml`；`schema` 指向 `server/db/migrations`，`queries` 指向 `server/db/queries`，生成代码输出到 `server/internal/db/sqlc`。
