# 更新日志

本文档记录了自项目初始提交以来对 Cloudflare Worker 后端进行的所有重大变更和重构。

## 核心架构与设计重构

### 1. 认证与权限体系简化
- **移除了“管理员”角色**：彻底删除了 `admin` 用户角色，简化了用户模型，所有用户现在都是平等的普通用户。
- **废除了管理员接口**：移除了所有仅供管理员使用的 API 接口，如创建/删除用户、批量刷新密钥等。
- **简化了用户创建脚本**：将 `admin:ensure` 脚本重构为 `user:ensure`，其唯一职责是在数据库为空时创建第一个初始用户，并打印其 API Key。

### 2. Cookie 共享池（Pool）的权限分离
- **实现了独立的访问控制**：将共享 Cookie 池的接口 (`/pool/cookies/{domain}`) 从管理员路由中分离出来，创建了独立的 `/api/v1/pool` 路由组。
- **引入了独立的 Pool Access Key**：访问共享池不再使用高权限的用户 API Key，而是需要一个独立的、权限更小的 `POOL_ACCESS_KEY`，并通过 `x-pool-key` 请求头进行验证。
- **提升了密钥安全性**：将 `POOL_ACCESS_KEY` 的管理方式从 `wrangler.toml` 的明文配置，变更为通过 Cloudflare Secrets (`wrangler secret put`) 进行安全管理，遵循了最佳安全实践。

## 功能增强与缺陷修复

### 1. API 功能对齐
- **补全了缺失接口**：参照 Go 版本的实现，补全了多个缺失的 API 接口，包括：
    - `GET /api/v1/health`：用于健康检查。
    - `GET /api/v1/cookies/{domain}`：获取指定域名的 Cookie。
    - `GET /api/v1/cookies/{domain}/{name}`：获取单个 Cookie 的值。
- **实现了子域匹配**：重构了数据库查询逻辑，现在获取 Cookie 的接口能够正确地返回主域名及其所有子域的 Cookie。
- **支持了 `format` 参数**：为所有获取 Cookie 的接口添加了 `format` 查询参数，使其能够根据需要返回完整的 JSON 对象或与 Go 版本行为一致的 HTTP Header 字符串。
- **统一了默认行为**：将获取 Cookie 接口的默认返回格式统一为 `header`，与 Go 版本保持一致。

### 2. 开发者体验优化
- **修复了 API 路径错误**：彻底解决了因路由嵌套和挂载顺序错误导致的 API 路径重复（如 `/api/v1/api/v1/...` 或 `/admin/admin/...`）的问题。
- **改善了错误响应**：实现了全局错误处理器，特别是针对 Zod 验证错误，现在会返回结构清晰、易于阅读的 JSON 错误信息，极大地改善了 API 的调试体验。
- **完善了 API 文档**：解决了因路由拆分导致 Swagger UI 无法显示认证入口的问题，确保了生成的 OpenAPI 文档的完整性和可用性。

### 3. 数据库结构演进
- 创建了多个数据库迁移文件，以追踪和应用数据库表结构的变更：
    - `0003_remove_role_from_users.sql`：从 `users` 表中移除了 `role` 字段。
    - `0004_add_remark_to_users.sql`：为 `users` 表添加了可为空的 `remark` 字段，用于用户备注。
