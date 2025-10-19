# Cookie Syncer - 开发指南

## 🚀 技术栈

- **Vue 3** - 现代化前端框架
- **TypeScript** - 类型安全的JavaScript
- **Vite** - 快速构建工具
- **Chrome Extension API** - 浏览器扩展API

## 📁 项目结构

```
src/
├── popup/           # Popup弹窗界面
│   ├── Popup.vue    # 主组件
│   ├── index.html   # HTML入口
│   └── index.ts     # TypeScript入口
├── options/         # 设置页面
│   ├── OptionsPage.vue
│   ├── index.html
│   └── index.ts
├── background/      # 后台脚本
│   └── index.ts     # Service Worker
└── core/            # 核心逻辑
    └── CookiePusherApp.ts
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

### 3. 构建生产版本

```bash
npm run build
```

构建优化后的生产版本到 `dist/` 目录。

### 4. 构建浏览器插件包

```bash
npm run build:extension
```

构建完整的浏览器插件，包含zip打包文件。

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
- **Background调试**: 扩展管理页面 → 点击插件的"背景页"链接

## 📋 开发规范

### TypeScript 规范

- 使用严格的TypeScript配置
- 为所有函数和变量添加类型注解
- 使用接口定义数据结构
- 避免使用 `any` 类型

### Vue 3 规范

- 使用 Composition API
- 单文件组件结构：`<template>`, `<script setup>`, `<style scoped>`
- 使用 `ref` 和 `reactive` 管理状态
- 组件命名使用 PascalCase

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

### 单元测试

```bash
npm test
```

### 手动测试清单

- [ ] Popup界面正常显示
- [ ] Cookie获取功能正常
- [ ] 域名管理功能正常
- [ ] 设置页面功能正常
- [ ] 后台脚本运行正常

## 📦 发布

### 打包发布版本

```bash
npm run build:extension
```

生成 `dist/CookiePusher.zip` 文件，可用于发布到浏览器应用商店。

### 版本管理

- 更新 `package.json` 中的版本号
- 更新 `manifest.json` 中的版本号
- 提交 Git 标签

## 🐛 常见问题

### Q: 插件无法加载？
A: 检查 `manifest.json` 配置是否正确，特别是路径配置。

### Q: 热重载不工作？
A: 确保运行 `npm run dev` 开发服务器。

### Q: TypeScript 编译错误？
A: 检查类型注解是否正确，运行 `npm run type-check` 检查类型。

## 📚 相关文档

- [Vue 3 官方文档](https://vuejs.org/)
- [TypeScript 官方文档](https://www.typescriptlang.org/)
- [Chrome Extension API](https://developer.chrome.com/docs/extensions/)
- [Vite 官方文档](https://vitejs.dev/)

---

**Happy Coding! 🎉**
