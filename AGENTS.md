# AGENTS.md

本文件面向在本仓库中工作的 AI Agent 和协作者，说明项目背景、目录约定、开发原则和提交要求。

## 项目背景

本项目是一个 LLM API Gateway，已有静态前端控制台与 Go 后端（标准库 net/http + PostgreSQL/sqlc）。后端位于 `server/`，提供 `/admin` 管理端接口与 `/v1` OpenAI 兼容下游接口。

前端控制台位于 `dashboard/`，是一个无需打包的静态 HTML/CSS/JS 项目。它默认请求同源 `/admin` 管理端接口，也支持通过 URL 参数覆盖接口地址：

```text
?api_base=http://host:port/admin
```

后端 API 要求集中整理在：

```text
docs/api-requirements.md
```

## 目录说明

```text
server/                    Go 模块根（module LLMGateway/server，go.mod / go.sum / sqlc.yaml）
server/cmd/llmgateway/     进程入口与 HTTP 路由表（router.go）
server/internal/           后端实现（config/crypto/money/domain/store/handler/db）
server/db/                 迁移与 sqlc 查询（migrations/、queries/）
dashboard/                 静态前端控制台（仓库根）
dashboard/index.html       前端入口
dashboard/js/data.js       前端数据接入层，定义 /admin 接口调用
dashboard/js/core.js       前端导航、路由和启动逻辑
dashboard/js/views/        各控制台页面
docs/api-requirements.md   后端 API 要求
docs/backend-structure.md  后端目录与分层约定
deployments/               本地 PostgreSQL docker compose
communication/             本地协作通信文件，不提交
```

## 运行与验证

所有 Go 命令都在 `server/` 目录下执行（或在仓库根使用 `go -C server ...`）：

```powershell
cd server
go build ./...
go vet ./...
go test ./...
gofmt -l cmd internal
sqlc generate
go run ./cmd/llmgateway
```

运行默认使用内存存储，托管仓库根的 `dashboard/`：

```text
DASHBOARD_DIR=../dashboard
```

PostgreSQL 集成测试需要 `TEST_DATABASE_URL`，未设置时自动跳过。

## Git 约定

- `communication/` 不允许提交，已在 `.gitignore` 中忽略。
- 不要提交本地密钥、上游 API Key、数据库密码、Token 或任何敏感配置。
- 修改前先查看当前工作区状态，避免覆盖他人改动。
- 不要无理由重排、格式化大量无关代码。
- 每次提交应聚焦一个明确目标。

## 后端开发原则

- 优先满足 `docs/api-requirements.md` 中 Dashboard 启动必需接口。
- `/admin` 接口统一返回 `{code,message,data}`。
- 列表接口统一返回 `{list,total}`。
- 金额建议使用字符串表示，避免浮点精度问题。
- 时间使用 RFC3339；自然日参数使用 `YYYY-MM-DD`。
- 管理端当前前端假设“不设认证”，如果后端加入认证，需要同步修改前端请求逻辑。
- 下游接口应尽量兼容 OpenAI API，例如 `/v1/models` 和 `/v1/chat/completions`。

## 推荐实现顺序

1. 静态托管 `dashboard/`。
2. 实现 `/healthz`。
3. 实现统一响应封装和错误处理。
4. 实现 Dashboard 启动必需的只读接口。
5. 实现渠道、用户、Key、限流、定价的 CRUD。
6. 实现 OpenAI 兼容下游接口。
7. 实现路由、计费、限流、日志和余额扣减。

## 前端注意事项

- 前端启动时 `loadDashboardData()` 会并发请求多个 `/admin` 接口，任一失败都会进入错误页。
- `dashboard/js/views/pricing.js` 当前使用 `DELETE /admin/pricing` 并携带 JSON 请求体，后端需兼容该行为，或同步调整前端。
- 文档页会加载 `docs/README.md`、`docs/01-下游接口/*`、`docs/02-管理端接口/*` 等文件；如果这些文件不存在，文档页会提示加载失败。
- Dashboard 中成功率在请求量为 0 时可能显示 `NaN%`，后续可修复。

## 验证建议

后端实现后至少验证：

```text
GET /healthz
GET /admin/stats/overview
GET /admin/stats/daily
GET /admin/channels
GET /admin/stats/channels
GET /admin/usage-logs
GET /admin/users
GET /admin/rate-limits
GET /admin/models
```

确认前端 `dashboard/index.html` 可以正常加载并进入仪表盘。
