# CookiePusher - Go 后端服务

这是 CookiePusher 项目的 Go 语言后端实现。它提供了一个稳定、高效的 API 服务，用于数据推送和管理，推荐在拥有自己服务器的用户部署。

## ✨ 特性

- **高性能**: 基于 Go 语言和 `chi` 路由，性能出色。
- **多数据库支持**: 支持 PostgreSQL, MySQL 和 SQLite，开箱即用。
- **容器化**: 提供 Docker 和 Docker Compose 配置，实现一键部署。
- **热重载**: 集成 `air` 实现开发环境下的热重载，提升开发效率。
- **API 文档**: 内置 Swagger，提供交互式 API 文档。
- **配置灵活**: 支持通过 `.env` 文件和命令行参数进行配置。

## 🚀 快速开始

### 1. 依赖

- [Go](https://go.dev/) (v1.25+)
- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/) (推荐)

### 2. 配置

项目通过 `.env` 文件或命令行参数进行配置，优先级：**命令行参数 > 环境变量 > .env 文件 > 默认值**。

1.  **复制模板**:
    ```bash
    cp .env.example .env
    ```

2.  **编辑配置**: 打开 `.env` 文件并根据您的需求进行修改。
    - **必须**设置 `ADMIN_KEY`。
    - 如果您想使用外部数据库，请修改 `DB_TYPE` 和 `DSN`。

    **数据库 DSN 示例:**
    - **PostgreSQL**: `DSN="host=localhost user=user password=pass dbname=db port=5432 sslmode=disable"`
    - **MySQL**: `DSN="user:pass@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local"`
    - **SQLite (默认)**: `DSN="CookiePusher.db"`

### 3. 启动服务

#### 使用 Docker Compose (推荐)

这是最简单、最推荐的启动方式。它会自动为您启动一个 PostgreSQL 数据库实例和后端 API 服务。

```bash
# 启动服务 (后台运行)
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

#### 本地开发

如果您想在本地直接运行 Go 代码：

```bash
# 进入后端目录
cd api_service/backend

# 确保您已经配置好了 .env 文件或相关的环境变量
# (推荐使用 air 进行热重载开发)
go run github.com/air-verse/air
```

服务将根据您的配置启动，默认监听在 `http://0.0.0.0:8080`。

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


## 🐳 Docker 部署

我们推荐使用 Docker 进行部署。

### 场景一: 连接到您自己的数据库 (主要方式)

这是最常见的部署方式。您需要一个正在运行的 PostgreSQL 或 MySQL 数据库，并通过 `.env` 文件将连接信息提供给 API 服务。

```bash
# 1. 创建并配置 .env 文件
# 确保填写了 ADMIN_KEY, DB_TYPE, 和 DSN
cp .env.example .env
nano .env

# 2. 启动 API 服务
docker-compose up -d api

# 3. 查看日志
docker-compose logs -f api
```

### 场景二: 使用 Docker Compose 启动本地测试环境

如果您只是想快速在本地启动一个**包含数据库**的完整测试环境，可以使用 `with-db` profile。

```bash
# 1. 创建并配置 .env 文件
cp .env.example .env
nano .env # 必须设置 ADMIN_KEY

# 在 .env 文件中，确保 DSN 指向 Docker Compose 内部的数据库
# DB_TYPE=postgres
# DSN="host=db user=user password=password dbname=cookiepusher port=5432 sslmode=disable"
# 注意：上面的 DSN 是一个示例，您应该使用 docker-compose.yml 中 db 服务的配置

# 2. 启动所有服务 (API + DB)
docker-compose --profile with-db up -d

# 3. 查看日志
docker-compose logs -f
```

### 手动运行 Docker 镜像

您也可以不使用 `docker-compose`，直接运行我们构建好的 Docker 镜像：

```bash
# 1. 构建镜像
docker build -t cookiepusher-api .

# 2. 运行容器 (连接到外部数据库)
# 方法 A: 使用 .env 文件 (推荐)
docker run -d \
  --name cookiepusher-api \
  -p 8080:8080 \
  --env-file ./.env \
  cookiepusher-api

# 方法 B: 直接设置环境变量
docker run -d \
  --name cookiepusher-api \
  -p 8080:8080 \
  -e ADMIN_KEY="your-super-secret-key" \
  -e DB_TYPE="postgres" \
  -e DSN="host=your_db_host user=user password=pass dbname=db port=5432" \
  cookiepusher-api
```

## 🔄 自动化发布

本项目使用 GitHub Actions 进行自动化构建和发布：

### 发布工作流

**发布构建** (`.github/workflows/release.yml`)
- 多平台二进制文件构建
- Docker 镜像构建和推送
- 自动创建 GitHub Release

### 发布流程

1. 创建版本标签：
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions 自动触发：
   - 构建多平台二进制文件
   - 构建并推送 Docker 镜像
   - 创建 GitHub Release

3. 下载发布版本：
   - 从 [GitHub Releases](../../releases) 下载二进制文件
   - 或使用 `docker pull ghcr.io/drean21/cookiepusher:v1.0.0`

### 支持的平台

**二进制文件**：
- Linux (amd64, arm64)
- Windows (amd64)
- macOS (amd64, arm64)

**Docker 镜像**：
- Linux (amd64, arm64)

## 📦 发布版本

### 二进制文件

每个发布版本包含以下文件：

| 文件名 | 平台 | 架构 |
|--------|------|------|
| `cookiepusher-linux-amd64.tar.gz` | Linux | AMD64 |
| `cookiepusher-linux-arm64.tar.gz` | Linux | ARM64 |
| `cookiepusher-windows-amd64.zip` | Windows | AMD64 |
| `cookiepusher-darwin-amd64.tar.gz` | macOS | Intel |
| `cookiepusher-darwin-arm64.tar.gz` | macOS | Apple Silicon |


### Docker 镜像标签

- `ghcr.io/drean21/cookiepusher:latest` - 最新稳定版
- `ghcr.io/drean21/cookiepusher:v1.0.0` - 特定版本
- `ghcr.io/drean21/cookiepusher:v1.0` - 主版本

## 🔧 配置选项

完整的环境变量列表请参考 [`.env.example`](.env.example)。



## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](../../LICENSE) 文件了解详情。
