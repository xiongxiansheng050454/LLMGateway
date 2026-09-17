# Backend Structure

后端目录按职责分层，目录本身体现 HTTP 入口、协议适配、领域模型和存储实现边界。

```text
server/                             Go 模块根（go.mod / go.sum / sqlc.yaml）
server/cmd/llmgateway/              进程入口与 HTTP 路由表（router.go）：config -> store -> httpapi -> http.Server
server/internal/config/             环境变量配置读取，集中管理默认值
server/internal/httpapi/            HTTP 入口、请求解析、响应封装、Dashboard 静态资源、/admin 与 /v1 分派编排
server/internal/protocol/openai/    OpenAI 兼容 wire DTO 与协议响应结构
server/internal/domain/             API 与业务共享类型，不依赖 HTTP 或数据库
server/internal/money/              定点金额（int64 最小单位）解析与格式化
server/internal/crypto/             渠道 api_key 加解密、网关 Key 生成与哈希
server/internal/store/              存储接口（Store）与通用存储错误
server/internal/store/memory/       默认内存实现，用于 MVP、测试和未接入 PG 前的功能迭代
server/internal/store/postgres/     PostgreSQL Store 实现
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

- HTTP 入口只依赖 `server/internal/store.Store` 接口，不直接访问 PostgreSQL 或 sqlc。
- 业务/API 共享结构放在 `server/internal/domain`，避免 httpapi、memory store、postgres store 互相引用具体实现。
- OpenAI 兼容 JSON wire type 放在 `server/internal/protocol/openai`；`server/internal/domain` 与 `server/internal/store` 不依赖 OpenAI 协议 DTO。
- 默认使用 `server/internal/store/memory`；设置 `DATABASE_URL` 时使用 `server/internal/store/postgres`，两者实现同一个 `store.Store` 接口且行为一致。
- sqlc 查询写在 `server/db/queries/*.sql`，schema 写在 `server/db/migrations/*.sql`，生成代码输出到 `server/internal/db/sqlc`。
- 不要手改 `server/internal/db/sqlc` 生成文件；修改 SQL 后运行 `sqlc generate`。
- 初始 schema 覆盖渠道、模型映射、定价、用户、余额、Key、限流和用量日志，后续 issue 应优先扩展现有表而不是新建重复概念。
- 统计接口（overview/daily/channels）在 `usage_logs` 上实时聚合，按 UTC 自然日分组；`daily_usage_stats` 因未被使用且复合主键无法表达全局日汇总，已在迁移 `000004_drop_daily_usage_stats.sql` 中删除。
- 进程启动时按 `DATABASE_URL` 选择实现：未设置使用 memory，设置则建立 pgxpool 连接并选用 PostgreSQL store。
- PostgreSQL store 已实现渠道/模型/定价、用户/余额/Key、限流规则与用量日志的读写；memory 与 postgres 两个实现的字段、错误码、排序与金额格式必须保持一致。
- `server/cmd/llmgateway` 使用 `http.Server` 并在收到 `SIGINT`/`SIGTERM` 后优雅关闭。
- 金额能力集中在 `server/internal/money`，密钥能力集中在 `server/internal/crypto`；memory 与 postgres 均调用 `internal/money`，禁止各 store 各自实现金额解析。
- 禁止用 `float64` 参与计费；金额在 DB 用 `NUMERIC`，在 Go 用定点整数，对外输出字符串。
- 渠道 `api_key` 落库为密文，网关 Key 只存哈希；任何响应、日志、错误都不得出现明文密钥。
- `server/internal/money` 与 `server/internal/crypto` 为叶子包，不得依赖 `server/internal/store` 或 `server/internal/httpapi`。
- `server/internal/crypto` 的渠道密钥加密密钥来自环境变量 `CHANNEL_KEY_ENCRYPTION_KEY`（原始字节，长度 16/24/32）；缺失或非法时返回错误，禁止明文回退。
- 网关 Key 仅保存 `server/internal/crypto.HashKey` 的哈希，明文 `full_key` 只在创建/重置时返回一次。
- 阶段说明：本阶段 `server/internal/store/memory` 仍以进程内明文 `api_key` 支撑 MVP（不落盘），`server/internal/crypto` 先提供加解密与哈希能力；PostgreSQL store（#12）落库时使用 `api_key_ciphertext`，并复用本包完成加解密。
- 不引入 service 层：HTTP 编排位于 `server/internal/httpapi`，与既有 `admin_*.go` 一致；下游代理按关注点分 `openai*.go`。
- HTTP 路由表（路径到入口的映射）集中在 `server/cmd/llmgateway/router.go`；`server/internal/httpapi` 只提供入口方法，不构造 mux。

## 文件组织约定

目录保持较浅层级：包内按领域拆文件，仅在协议与存储实现处使用子包，使目录能直接呈现模块边界。

- 每个包内按领域命名文件，禁止把多个领域堆进同一个文件：
  - `server/internal/domain/`：`common.go`、`channel.go`、`user.go`、`usage.go`、`ratelimit.go`、`channelhealth.go`、`failurereason.go`；纯规则（熔断状态机、限流规范化 `NormalizeRateLimit`、时间校验 `Validate*`、`FailureReason`）归位此处
  - `server/internal/store/`：`store.go`（错误别名 + 组合接口）、`channel.go`、`user.go`、`usage.go`、`ratelimit.go`、`channelhealth.go`（仅端口接口）；`CanonicalJSON` 为序列化一致性辅助，非业务规则
  - `server/internal/store/memory/`：`memory.go`（结构体/构造函数/共享辅助）、`channel.go`、`user.go`、`usage.go`、`ratelimit.go`、`channelhealth.go`
  - `server/internal/store/postgres/`：`postgres.go`（结构体/构造函数）、`channel.go`、`user.go`、`usage.go`、`ratelimit.go`、`channelhealth.go`
  - `server/cmd/llmgateway/`：`main.go`（装配与优雅关闭）、`router.go`（唯一 HTTP 路由表）
  - `server/internal/httpapi/`：`handler.go`（入口方法/分派/响应/分页/静态托管）、`admin_channel.go`、`admin_pricing.go`、`admin_user.go`、`admin_key.go`、`admin_ratelimit.go`、`admin_usage.go`、`channel_upstream.go`、`openai.go`（/v1 分派与错误映射）、`openai_auth.go`、`openai_route.go`、`openai_billing.go`、`openai_ratelimit.go`、`openai_proxy.go`
  - `server/internal/protocol/openai/`：`types.go`（OpenAI 兼容请求、响应和错误 DTO）
- `store.Store` 由领域子接口组合而成，禁止继续往 `store.go` 堆方法：

```go
type Store interface {
    ChannelStore
    // UserStore / UsageStore / RateLimitStore 由对应 issue 加入
}
```

- 实现必须放在 `server/internal/store/memory` 与 `server/internal/store/postgres`，并以编译期断言固定：

```go
var _ store.ChannelStore = (*memory.Store)(nil)
```

- 领域文件边界与 `server/db/queries/*.sql` 的领域划分保持一致（channels/models/pricing/users/rate_limits/usage_logs）。
- 拆分与移动只做等价搬迁，不得顺手改变路由、响应结构、状态码或错误语义。

## 下游代理（/v1）

- `GET /v1/models` 与 `POST /v1/chat/completions` 由 `server/internal/httpapi` 暴露，按关注点分文件：`openai.go`（分派/错误映射）、`openai_auth.go`、`openai_route.go`、`openai_billing.go`、`openai_ratelimit.go`、`openai_proxy.go`；OpenAI JSON DTO 位于 `server/internal/protocol/openai`。
- 认证使用 `Authorization: Bearer <gateway-key>`；密钥经 `server/internal/crypto.HashKey` 后查询，明文不落日志/响应。
- 路由候选按 `priority` 越大越优先，同级内按 `weight` 加权随机；非正余额渠道被排除。
- 计费：缓存 token 已包含在 `prompt_tokens` 中，仅按 `(prompt_tokens - cached_tokens)` 计输入价，缓存部分计缓存价，避免重复计费。
- 成功结算：非流式 chat completion 成功后通过 store 级 `SettleChatCompletion` 端口统一处理用户扣费、可扣费渠道余额扣减与 success usage log。PostgreSQL 实现在单一事务中提交；内存实现保持相同可观察错误语义。`last_used_at` 仍为成功响应后的 best-effort 更新。
- 限流：`rpm` + `reject` 规则基于 `usage_logs` 统计最近 1 分钟请求次数。`global`/`user`/`api_key` 保持按当前用户/Key 计数；`model` 规则额外按 public model 精确过滤；`channel` 规则在路由选中最终渠道后、调用上游前评估，超限直接返回 429 且写入 `error_code=rate_limited` 的 error usage log，不自动改选其他渠道。
- 熔断：每个渠道有 `channel_health` 状态（closed/open/half-open）。连续失败达阈值（默认 5）或确定性失败（上游 401/403/402）立即 open；冷却（默认 30s）后惰性转为 half-open 允许探测，探测成功回 closed、失败回 open。`ListRouteCandidates` 排除 open 渠道；当无可用渠道（无映射或全部 open）时返回 `503 no_healthy_channel`（错误码由 `no_available_channel` 变更而来，同时覆盖这两种情况）。失败分类仅计入传输错误、上游 429/401/403/402 与 5xx，其余 4xx 透传且不计渠道失败。健康记录为 best-effort。
- 已知限制（后续 issue 处理）：
  - 未配置 `model_pricing` 的渠道×模型按 cost=0 放行（建议为所有可路由模型配置定价）。
  - `queue` 动作未实现；`stream=true` 返回 400（SSE 未实现）。

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
MIGRATIONS_DIR=db/migrations
```

路径均相对于运行目录 `server/`：`DASHBOARD_DIR` 默认 `../dashboard`，`MIGRATIONS_DIR` 默认 `db/migrations`。

未设置 `DATABASE_URL` 时使用内存 store，可直接启动：

```powershell
cd server
go run ./cmd/llmgateway
```

## 迁移

本仓库选择**最小自建 runner**（`server/internal/db/migrate`），基于 pgx，不引入额外迁移依赖。

- 迁移文件为 `server/db/migrations/*.sql`，按文件名（版本前缀）字典序执行。
- 每个文件在独立事务中执行，并在 `schema_migrations(version)` 中记录，已执行版本会跳过。
- 设置 `DATABASE_URL` 时，进程启动会自动执行 `MIGRATIONS_DIR`（默认 `db/migrations`）下的待执行迁移；失败时启动报错退出。

## sqlc

在 `server/` 目录下执行：

```powershell
cd server
sqlc generate
```

配置文件：`server/sqlc.yaml`；`schema` 指向 `server/db/migrations`，`queries` 指向 `server/db/queries`，生成代码输出到 `server/internal/db/sqlc`。
