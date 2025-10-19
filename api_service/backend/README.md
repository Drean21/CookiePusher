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

## 🐳 Docker 部署

### 使用 Docker Compose（推荐）

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑环境变量（必须修改 ADMIN_KEY）
nano .env

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 使用 Docker CLI

```bash
# 构建镜像
docker build -t cookiepusher .

# 运行容器
docker run -d \
  --name cookiepusher \
  -p 8080:8080 \
  -e ADMIN_KEY=your-super-secret-admin-key \
  -v $(pwd)/data:/root/data \
  cookiepusher
```

### 使用预构建镜像

```bash
# 拉取最新镜像
docker pull ghcr.io/Drean21/CookiePusher:latest

# 运行容器
docker run -d \
  --name cookiepusher \
  -p 8080:8080 \
  -e ADMIN_KEY=your-super-secret-admin-key \
  -v cookiepusher_data:/root/data \
  ghcr.io/Drean21/CookiePusher:latest
```

详细的 Docker 部署指南请参考 [docker/README.md](docker/README.md)。

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
   - 或使用 `docker pull ghcr.io/your-username/cookiepusher:v1.0.0`

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

### 使用预编译二进制文件

```bash
# 下载并解压
wget https://github.com/your-username/cookiepusher/releases/download/v1.0.0/cookiepusher-linux-amd64.tar.gz
tar -xzf cookiepusher-linux-amd64.tar.gz

# 运行
./cookiepusher -admin-key=your-admin-key
```

### Docker 镜像标签

- `ghcr.io/your-username/cookiepusher:latest` - 最新稳定版
- `ghcr.io/your-username/cookiepusher:v1.0.0` - 特定版本
- `ghcr.io/your-username/cookiepusher:v1.0` - 主版本

## 🔧 配置选项

### 环境变量

完整的环境变量列表请参考 [`.env.example`](.env.example)。

### 配置文件

支持通过配置文件进行配置：

```yaml
# config.yml
server:
  port: 8080
  host: "0.0.0.0"

database:
  path: "/root/data/CookiePusher.db"
  max_open_connections: 25
  max_idle_connections: 5

security:
  admin_key: "your-admin-key"
  pool_access_key: "your-pool-key"

logging:
  level: "info"
  format: "json"
```

使用配置文件启动：
```bash
./cookiepusher -config config.yml
```

## 🛠️ 开发

### 本地开发

```bash
# 安装依赖
go mod download

# 安装开发工具
go install github.com/air-verse/air@latest
go install github.com/swaggo/swag/cmd/swag@latest

# 生成 Swagger 文档
swag init -g cmd/api/main.go -o docs

# 启动热重载开发服务器
air
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并生成覆盖率报告
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# 查看覆盖率
go tool cover -html=coverage.out
```

### 代码质量

```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...

# 安全扫描
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
gosec ./...
```

## 🚨 故障排除

### 常见问题

**Q: 数据库连接失败**
A: 检查数据库文件路径和权限，确保目录存在且可写。

**Q: API 认证失败**
A: 确认 `ADMIN_KEY` 环境变量设置正确，且请求头格式正确。

**Q: Docker 容器启动失败**
A: 检查环境变量设置，查看容器日志：
```bash
docker logs cookiepusher
```

**Q: 跨域问题**
A: 检查 CORS 配置，确保前端地址在允许列表中。

### 日志调试

启用调试模式：
```bash
export LOG_LEVEL=debug
./cookiepusher
```

### 性能监控

启用内置指标：
```bash
export METRICS_ENABLED=true
./cookiepusher
```

访问 `http://localhost:8080/metrics` 查看指标。

## 🤝 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](../../LICENSE) 文件了解详情。
