# Docker 部署

1. 根据 `deployments/.env.example` 创建 `deployments/.env`，并替换其中的两个占位密钥。
   `CHANNEL_KEY_ENCRYPTION_KEY` 必须恰好为 16、24 或 32 个 ASCII 字节；渠道密钥已保存后，不能再修改该值，否则网关无法解密已有渠道密钥。
2. 在仓库根目录执行：

   ```powershell
   docker compose -f deployments/docker-compose.yml up -d --build
   ```

3. 打开 `http://localhost:8080/dashboard/` 网关会在接收请求前自动执行数据库迁移。

查看网关日志：

```powershell
docker compose -f deployments/docker-compose.yml logs -f llmgateway
```

停止服务但保留 PostgreSQL 数据卷：

```powershell
docker compose -f deployments/docker-compose.yml down
```
