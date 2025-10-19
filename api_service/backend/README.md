# CookiePusher - Go 后端服务

这是 CookiePusher 项目的 Go 语言后端实现。它提供了一个稳定、高效的 API 服务，用于数据推送和管理，推荐在拥有自己服务器的用户部署。

## ✨ 特性

- **高性能**: 基于 Go 语言和 `chi` 路由，性能出色。
- **轻量级**: 使用 SQLite 作为数据库，无需额外配置，开箱即用。
- **热重载**: 集成 `air` 实现开发环境下的热重载，提升开发效率。
- **API 文档**: 内置 Swagger，提供交互式 API 文档。
- **三层认证**:
  - `x-api-key`: 普通用户认证。
  - `x-pool-key`: 用于共享池的特殊认证。
  - `x-admin-key`: 用于管理操作的管理员认证。

## 🚀 快速开始

### 1. 依赖

- [Go](https://go.dev/) (v1.25+)

### 2. 配置密钥

在启动服务前，您必须配置 `ADMIN_KEY`。`POOL_ACCESS_KEY` 是可选的。

您可以通过以下**三种方式**（优先级从高到低）进行配置：

1.  **命令行参数 (最高优先级)**:

    ```bash
    go run ./cmd/api -admin-key="YOUR_SECRET_ADMIN_KEY"
    ```

2.  **环境变量**:

    ```bash
    export ADMIN_KEY="YOUR_SECRET_ADMIN_KEY"
    go run ./cmd/api
    ```

3.  **.env 文件 (最低优先级)**:
    在 `api_service/backend` 目录下创建一个 `.env` 文件，内容如下：
    ```
    ADMIN_KEY="YOUR_SECRET_ADMIN_KEY"
    ```
    **注意**: 请自行生成并保管好您的密钥。服务本身**不会**自动生成。

### 3. 启动服务

```bash
# 进入后端目录
cd api_service/backend

# 运行服务 (推荐使用 air 进行热重载开发)
# air 会自动加载 .env 文件
go run github.com/air-verse/air
```

服务首次启动时，会在当前目录创建 `CookiePusher.db` 数据库文件。服务将默认监听在 `http://localhost:8080`。

### 4. API 文档

启动服务后，可访问 [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) 查看和测试所有 API 接口。

### 5. 创建用户

您需要使用配置好的 `ADMIN_KEY` 来为插件创建用户和对应的 `x-api-key`。

**示例 (使用 cURL):**

```bash
curl -X POST 'http://localhost:8080/api/v1/admin/users' \
--header 'x-admin-key: YOUR_SECRET_ADMIN_KEY' \
--header 'Content-Type: application/json' \
--data-raw '[
    {
        "username": "my-user",
        "remark": "My primary account"
    }
]'
```

在响应中可以找到为 `my-user` 生成的 `api_key`，这个值就是插件设置中需要的 "Auth Token"。

## 6. API 文档

本项目使用 `swaggo` 自动生成 Swagger UI 文档。

启动服务后，可访问 [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) 查看和测试所有 API 接口。
