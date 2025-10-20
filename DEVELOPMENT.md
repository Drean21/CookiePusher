# CookiePusher - 开发指南

## 🚀 技术栈

- **Vue 3** - 现代化前端框架
- **TypeScript** - 类型安全的JavaScript
- **Vite** - 快速构建工具
- **Chrome Extension Manifest V3** - 最新浏览器扩展API
- **CryptoJS** - 加密库
- **ECharts** - 数据可视化图表

## 📁 项目结构

```
src/
├── popup/                    # Popup弹窗界面
│   ├── App.vue              # 主组件
│   ├── index.html           # HTML入口
│   ├── index.ts             # TypeScript入口
│   └── views/               # 页面组件
│       ├── CurrentPageView.vue
│       ├── LogView.vue
│       ├── ManagedDomainsView.vue
│       ├── SettingsView.vue
│       └── StatsView.vue
├── options/                 # 设置页面
│   ├── OptionsPage.vue
│   ├── index.html
│   └── index.ts
├── background/              # 后台脚本
│   └── index.ts             # Service Worker
├── offscreen/               # 离屏文档
│   ├── index.html
│   └── index.js
├── cookies/                 # Cookie管理页面
│   ├── CookiesPage.vue
│   └── components/
│       └── CookieDetails.vue
├── welcome/                 # 欢迎页面
│   ├── index.html
│   ├── navigation.js
│   └── redirect.js
├── utils/                   # 工具函数
│   └── message.ts
└── types/                   # TypeScript类型定义
    └── extension.d.ts

api_service/                 # 后端API服务
├── backend/                 # Go后端服务
└── cf/                      # Cloudflare Workers服务
```

## 🛠️ 开发环境设置

### 1. 安装依赖

```bash
npm install
```

### 2. 开发模式

```bash
npm run dev
```

启动开发服务器，支持热重载。修改代码后会自动重新构建。

### 3. 后端开发模式

```bash
npm run dev:backend
```

启动Go后端服务的热重载开发模式。

### 4. 构建生产版本

```bash
npm run build
```

构建优化后的生产版本到 `dist/` 目录。

### 5. 类型检查

```bash
npm run type-check
```

运行TypeScript类型检查，不生成输出文件。

## 🔧 插件加载

### Chrome/Edge 浏览器

1. 打开扩展管理页面：`chrome://extensions/` 或 `edge://extensions/`
2. 开启"开发者模式"
3. 点击"加载已解压的扩展程序"
4. 选择项目根目录下的 `dist/` 文件夹
5. 插件即可加载并运行

### 开发调试

- **Popup调试**: 右键点击插件图标 → "检查弹出内容"
- **Options页面调试**: 右键点击插件图标 → "选项" → 打开开发者工具
- **Background调试**: 扩展管理页面 → 点击插件的"Service Worker"链接
- **Offscreen调试**: 扩展管理页面 → 查看背景页控制台中的offscreen相关日志

## 📋 开发规范

### TypeScript 规范

- 使用严格的TypeScript配置
- 为所有函数和变量添加类型注解
- 使用接口定义数据结构
- 避免使用 `any` 类型
- 类型定义文件放在 `types/` 目录下

### Vue 3 规范

- 使用 Composition API
- 单文件组件结构：`<template>`, `<script setup>`, `<style scoped>`
- 使用 `ref` 和 `reactive` 管理状态
- 组件命名使用 PascalCase
- 页面组件放在 `views/` 目录下，通用组件放在 `components/` 目录下

### Chrome Extension 规范

- 使用 Manifest V3 规范
- Service Worker 处理后台任务
- 使用 Offscreen Document 处理需要 DOM 的操作
- 遵循最小权限原则配置权限

### 代码风格

- 使用 Prettier 格式化代码
- 遵循 ESLint 规则
- 组件样式使用 CSS Scoped
- 使用语义化的类名

## 🔄 更新流程

1. 修改代码
2. 运行 `npm run build` 构建
3. 在扩展管理页面点击刷新按钮
4. 测试功能

## 🧪 测试

### 类型检查

```bash
npm run type-check
```

### 手动测试清单

- [ ] Popup界面正常显示
- [ ] Cookie获取功能正常
- [ ] Cookie管理页面功能正常
- [ ] 域名管理功能正常
- [ ] 设置页面功能正常
- [ ] 统计页面显示正常
- [ ] Service Worker运行正常
- [ ] Offscreen Document功能正常
- [ ] 定时保活功能正常
- [ ] Cookie变更监听正常
- [ ] API推送功能正常

## 📦 发布

### 构建发布版本

```bash
npm run build
```

构建后的文件在 `dist/` 目录，可直接用于发布。

### 版本管理

- 更新 `package.json` 中的版本号
- 更新 `manifest.json` 中的版本号
- 提交 Git 标签

## 🐛 常见问题

### Q: 插件无法加载？
A: 检查 `manifest.json` 配置是否正确，特别是路径配置和权限设置。

### Q: 热重载不工作？
A: 确保运行 `npm run dev` 开发服务器，并检查浏览器扩展是否已重新加载。

### Q: TypeScript 编译错误？
A: 检查类型注解是否正确，运行 `npm run type-check` 检查类型。

### Q: Service Worker 不工作？
A: 检查 Manifest V3 的 Service Worker 配置，确保没有使用已弃用的 API。

### Q: Offscreen Document 创建失败？
A: 检查是否正确申请了 `offscreen` 权限，以及创建原因和说明是否合理。

## 🔧 核心功能调试

### Service Worker 调试
1. 打开扩展管理页面
2. 点击"Service Worker"链接
3. 查看控制台日志和网络请求

### Cookie 变更监听调试
1. 在 Service Worker 控制台中查看变更日志
2. 检查 `chrome.cookies.onChanged` 事件是否正确触发
3. 验证去抖机制是否正常工作

### 定时保活调试
1. 检查 `chrome.alarms` API 是否正确设置
2. 查看 Offscreen Document 的创建和销毁日志
3. 验证 Cookie 快照对比逻辑

## 📚 相关文档

- [Vue 3 官方文档](https://vuejs.org/)
- [TypeScript 官方文档](https://www.typescriptlang.org/)
- [Chrome Extension Manifest V3](https://developer.chrome.com/docs/extensions/mv3/)
- [Vite 官方文档](https://vitejs.dev/)
- [CryptoJS 文档](https://cryptojs.gitbook.io/docs/)
- [ECharts 文档](https://echarts.apache.org/)

---

**Happy Coding! 🎉**
