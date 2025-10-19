# Cookie Syncer - Cloudflare Worker 后端

> **⚠️ 重要警告：关于成本和使用策略**
> 
> 此 Cloudflare Worker 后端为用户提供了“零服务器”的便捷部署选项。然而，**请务必注意 Cloudflare 的免费额度限制**。对于个人自用，这些额度通常是充足的。但如果您**分发给多用户使用**，或者在插件中监控了**高频变化的 Cookie**，将有**超出免费额度并产生费用的风险**。
> 
> **强烈建议**:
> 1.  **按需同步**: 只将绝对必要的 Cookie 加入同步列表。
> 2.  **避免高频目标**: 不要同步那些频繁自动刷新且对您的使用场景无用的 Cookie。

此目录包含 Cookie Syncer 后端 API 的 Cloudflare Worker 实现。它被设计为 Go 后端的一个功能完备、无需服务器的替代方案，利用 Cloudflare 生态系统为个人用户提供简单、免费且稳健的部署选项。

## 1. 部署和管理指南

### 第 1 步: 前提条件

-   一个 [Cloudflare 账户](https://dash.cloudflare.com/sign-up)。
-   已安装 [Node.js](https://nodejs.org/) 和 `npm`。
-   登录到 Wrangler: `npx wrangler login`。

### 第 2 步: 安装依赖

进入此目录 (`api_service/cf`) 并运行：
```sh
npm install
```

### 第 3 步: 创建并初始化数据库

此步骤将一次性完成数据库的创建和表结构的初始化。

1.  **创建数据库容器**:
    ```sh
    npm run d1:create
    ```
    此命令会在您的 Cloudflare 账户下创建一个名为 `cookie-syncer-db` 的**空数据库容器**。

2.  **更新配置**: 命令执行成功后，Wrangler 将输出一个 `database_id`。请复制此 ID 并将其粘贴到 `wrangler.toml` 文件中的 `database_id` 字段。

3.  **执行数据库迁移**:
    ```sh
    npm run d1:migrate:prod
    ```
    此命令会找到 `migrations/` 目录下的 `0000_init_schema.sql` 文件，并在您上一步创建的数据库中执行它，从而一次性创建好所有需要的表和索引。

### 第 4 步: 配置密钥

您必须通过 Wrangler CLI 设置 `ADMIN_KEY` 作为生产环境的 **Secret**。`POOL_ACCESS_KEY` 是可选的。

```sh
# 请将 YOUR_SECRET_ADMIN_KEY 替换为您自己生成的强密码
npx wrangler secret put ADMIN_KEY
```
在提示符后输入您的密钥并回车。

### 第 5 步: 部署 Worker

此命令会打包您的代码并将其上传到 Cloudflare 网络。

```sh
npm run deploy
```
部署后，Wrangler 将显示您的 Worker 的公共 URL（例如 `https://cookie-syncer-api.<your-subdomain>.workers.dev`）。

### 第 6 步: 创建用户

您需要使用配置好的 `ADMIN_KEY` 调用管理员接口来为插件创建用户和对应的 `x-api-key`。

**示例 (使用 cURL):**
```bash
curl -X POST 'https://<YOUR_WORKER_URL>/api/v1/admin/users' \
--header 'x-admin-key: <YOUR_SECRET_ADMIN_KEY>' \
--header 'Content-Type: application/json' \
--data-raw '[
    {
        "username": "my-user",
        "remark": "My primary account"
    }
]'
```
在响应中可以找到为 `my-user` 生成的 `api_key`，这个值就是插件设置中需要的 "Auth Token"。

### 第 7 步: 配置浏览器插件

-   打开 Cookie Syncer 插件的设置。
-   将“API 端点”设置为您的 Worker 的公共 URL。
-   将“Auth Token”设置为您刚刚通过 API 创建的 `api_key`。
-   测试连接，此时应该会成功。

## 2. API 文档

本项目使用 `Hono` 的 OpenAPI 模块自动生成 API 文档。部署成功后，您可以访问以下路径查看：

-   **Swagger UI 界面**: `https://<YOUR_WORKER_URL>/swagger`
-   **OpenAPI 规范 (JSON)**: `https://<YOUR_WORKER_URL>/doc`

通过 Swagger UI，您可以直观地浏览所有 API 接口、请求参数和响应格式，并进行在线测试。

现在，您的 Serverless 后端已完全投入使用。
