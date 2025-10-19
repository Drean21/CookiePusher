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

## 3. 迁移到 VPS/Docker 指南

> **📋 为什么考虑迁移？**
>
> 虽然 Cloudflare Workers 提供了便捷的无服务器部署，但在某些场景下您可能需要自托管：
> - 需要更高的请求限制或更长的执行时间
> - 希望完全控制数据和基础设施
> - 需要更复杂的数据库操作或集成
> - 组织策略要求使用特定的云服务商

### 3.1 迁移可行性分析

**好消息：此 CF 实现迁移到 VPS/Docker 极其简单！**
（当然你更应该选择隔壁Go实现，此处只做技术可行性探讨）

原因：
- ✅ 使用 **Hono 框架** - 支持多运行时的通用 Web 框架
- ✅ 标准 **SQL 查询** - 与 SQLite 完全兼容
- ✅ 纯 **TypeScript 业务逻辑** - 无 CF 特定依赖
- ✅ **95% 代码无需修改** - 只需调整数据库连接和启动方式

### 3.2 技术栈对比

| 组件 | Cloudflare Workers | VPS/Docker |
|------|-------------------|------------|
| **Web 框架** | Hono (运行时: Workers) | Hono (运行时: Node.js) |
| **数据库** | D1 (SQLite 兼容) | 标准 SQLite |
| **环境变量** | `c.env.DB` | `process.env.DB` |
| **部署方式** | `wrangler deploy` | Docker/传统部署 |
| **启动方式** | `export default app` | `app.listen(port)` |

### 3.3 迁移步骤概览

#### 第 1 步：准备环境

```bash
# 创建新的项目目录
mkdir cookiepusher-vps
cd cookiepusher-vps

# 初始化 package.json
npm init -y
```

#### 第 2 步：安装依赖

```bash
npm install hono @hono/zod-openapi @hono/swagger-ui zod uuid sqlite3
npm install -D @types/node typescript tsx nodemon
```

#### 第 3 步：复制核心代码

从 `api_service/cf/src/` 复制以下文件到新项目：
- `index.ts` (主应用文件)
- `store.ts` (数据库操作)
- `models.ts` (数据模型)
- `schema.ts` (API 模式)
- `presenter.ts` (数据转换)
- `response.ts` (响应处理)

#### 第 4 步：修改数据库连接

**创建 `src/database.ts`**：
```typescript
import sqlite3 from 'sqlite3';
import { open, Database } from 'sqlite';

export async function createDatabase(): Promise<Database> {
  return open({
    filename: process.env.DB_PATH || './data/cookiepusher.db',
    driver: sqlite3.Database
  });
}
```

**修改 `store.ts` 构造函数**：
```typescript
// 原来
constructor(db: D1Database, adminKey: string, poolKey: string)

// 修改为
constructor(db: Database, adminKey: string, poolKey: string)
```

#### 第 5 步：创建服务器入口

**创建 `src/server.ts`**：
```typescript
import { Hono } from 'hono';
import { createDatabase } from './database';
import app from './index'; // 导入原有的 Hono 应用

const port = process.env.PORT || 3000;

async function startServer() {
  const db = await createDatabase();
  
  // 设置全局数据库实例
  (globalThis as any).db = db;
  
  // 添加数据库中间件
  app.use('*', async (c, next) => {
    c.set('store', new D1Store(
      db,
      process.env.ADMIN_KEY!,
      process.env.POOL_ACCESS_KEY!
    ));
    await next();
  });
  
  console.log(`Server running on port ${port}`);
  return {
    fetch: app.fetch,
    port
  };
}

// 启动服务器
startServer().then(() => {
  console.log('🚀 CookiePusher API Server started successfully!');
});
```

#### 第 6 步：数据库初始化

**创建 `scripts/init-db.js`**：
```javascript
const sqlite3 = require('sqlite3').verbose();
const fs = require('fs');

const db = new sqlite3.Database('./data/cookiepusher.db');

// 读取迁移文件
const migration = fs.readFileSync('./migrations/0000_init_schema.sql', 'utf8');

// 执行迁移
db.exec(migration, (err) => {
  if (err) {
    console.error('Migration failed:', err);
  } else {
    console.log('✅ Database initialized successfully');
  }
  
  db.close();
});
```

#### 第 7 步：Docker 配置

**创建 `Dockerfile`**：
```dockerfile
FROM node:18-alpine

WORKDIR /app

# 安装依赖
COPY package*.json ./
RUN npm ci --only=production

# 复制源代码
COPY src/ ./src/
COPY migrations/ ./migrations/

# 创建数据目录
RUN mkdir -p /app/data

# 初始化数据库
RUN npx tsx scripts/init-db.js

# 构建应用
RUN npm run build

EXPOSE 3000

# 设置环境变量
ENV NODE_ENV=production
ENV PORT=3000

CMD ["node", "dist/server.js"]
```

**创建 `docker-compose.yml`**：
```yaml
version: '3.8'

services:
  cookiepusher-api:
    build: .
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - ADMIN_KEY=${ADMIN_KEY}
      - POOL_ACCESS_KEY=${POOL_ACCESS_KEY}
      - DB_PATH=/app/data/cookiepusher.db
    volumes:
      - ./data:/app/data
    restart: unless-stopped
    
  # 可选：添加 nginx 反向代理
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - cookiepusher-api
    restart: unless-stopped
```

#### 第 8 步：环境变量配置

**创建 `.env` 文件**：
```bash
# 数据库路径
DB_PATH=./data/cookiepusher.db

# 服务端口
PORT=3000

# 密钥（请使用强密码）
ADMIN_KEY=your-super-secret-admin-key-here
POOL_ACCESS_KEY=your-pool-access-key-here

# 运行环境
NODE_ENV=production
```

### 3.4 部署命令

```bash
# 1. 构建并启动 Docker 容器
docker-compose up -d

# 2. 查看日志
docker-compose logs -f cookiepusher-api

# 3. 创建用户（与 CF 版本相同的 API）
curl -X POST 'http://localhost:3000/api/v1/admin/users' \
--header 'x-admin-key: your-super-secret-admin-key-here' \
--header 'Content-Type: application/json' \
--data-raw '[{"remark": "My VPS User"}]'
```

### 3.5 迁移验证清单

- [ ] API 接口正常响应
- [ ] 用户创建和管理功能正常
- [ ] Cookie 同步功能正常
- [ ] 数据持久化正常
- [ ] Swagger 文档可访问：`http://localhost:3000/swagger`
- [ ] 错误处理和日志记录正常

### 3.6 性能对比

| 指标 | Cloudflare Workers | VPS/Docker |
|------|-------------------|------------|
| **冷启动** | ~100ms | 无冷启动 |
| **请求限制** | 100,000/天 (免费) | 无限制 |
| **CPU 时间** | 50ms/请求 | 无限制 |
| **内存** | 128MB | 可配置 |
| **存储** | 5GB D1 | 磁盘空间限制 |
| **成本** | 免费额度 + 超额费用 | 服务器固定成本 |

### 3.7 常见问题

**Q: 迁移后 API 接口会变化吗？**
A: 不会。所有 API 路径、请求格式、响应格式完全保持一致。

**Q: 数据如何从 CF D1 迁移到自托管 SQLite？**
A: 可以使用 D1 导出功能，或者通过 API 调用重新同步数据。

**Q: 需要修改浏览器插件配置吗？**
A: 只需将 API 端点从 CF Workers URL 改为您的 VPS 地址即可。

**Q: 如何备份自托管的数据？**
A: 直接复制 SQLite 文件，或使用 `sqlite3 .backup` 命令。

---

**💡 提示**：欢迎提交Pull Request。社区贡献将帮助完善这个迁移指南。
