# 功能迭代记录文档

本文档记录了 `CookieSyncer` 浏览器插件从一个有问题的初始版本，通过多次迭代和重构，最终演变为一个功能完善、体验优良的专业工具的全过程。

## V0.1 - 初始状态：功能缺失与架构混乱

项目初始版本使用 Vite + Vue 3 + TypeScript 技术栈，但存在以下核心问题：

1.  **功能不完整**：插件的核心功能——查看当前页面的Cookie——完全无法正常工作。
2.  **架构混乱**：项目结构复杂，存在多个入口点和重复的背景脚本（`background.js` vs `src/background/index.ts`），导致构建和运行逻辑混乱。
3.  **构建问题**：自定义的构建脚本 `scripts/build-extension.js` 存在缺陷，无法正确处理 manifest V3 的文件结构，导致加载扩展时频繁出错。
4.  **数据获取错误**：即使在修复了部分问题后，获取Cookie的逻辑也存在严重缺陷，最初只能获取到当前子域名的Cookie，完全忽略了主域名和其他相关域的Cookie。

## V1.0 - 首次重构：走向单页应用（SPA）

在明确了用户的核心需求——“**首先要能看**”——之后，我们进行了第一次大规模重构，目标是将插件改造为一个基于Popup的单页应用。

**关键迭代：**

1.  **废弃自定义构建脚本**：放弃了 `scripts/build-extension.js`，转而使用社区成熟的 `vite-plugin-copy` 插件来处理静态资源和 `manifest.json` 的复制，解决了长期存在的构建问题。
2.  **统一入口点**：
    *   明确 `src/popup/index.html` 为唯一的 `action.default_popup`。
    *   明确 `src/background/index.ts` 为唯一的 `background.service_worker`。
    *   删除了所有其他冗余的、会产生冲突的HTML文件和JS脚本。
3.  **引入消息通信**：建立了 `popup` (前端) 和 `background` (后台) 之间的通信机制。前端通过 `chrome.runtime.sendMessage` 发送请求，后台监听并处理后返回数据。

**遇到的问题：**

*   **白屏Bug**：重构后Popup页面出现白屏，经排查发现是由于缺少一个关键的 `utils/message.ts` 文件，导致前端无法正确发送消息。
*   **欢迎页Bug**：一个顽固的 "welcome page" Bug 始终存在，最终定位到是由于旧的、未被彻底清除的 `background.js` 脚本仍在`dist`目录中作祟。

## V2.0 - 数据精确性攻坚：从“能看到”到“看正确”

在解决了基础架构问题后，核心矛盾转移到了数据获取的精确性上。这个阶段经历了多次失败的尝试和关键的架构演进。

**迭代路径：**

1.  **V2.1 - 首次尝试 (`cookies.getAll({ domain })`)**：
    *   **实现**：使用 `chrome.cookies.getAll({ domain: tab.hostname })`。
    *   **缺陷**：只能获取到精确子域名（如 `www.bilibili.com`）的Cookie，无法获取到父域名（如 `.bilibili.com`）的Cookie，导致数据大量缺失。

2.  **V2.2 - 改进 (`cookies.getAll({ url })`)**：
    *   **实现**：采纳用户建议，切换到 `chrome.cookies.getAll({ url: tab.url })`。
    *   **进步**：成功获取了包括父域名在内的所有与当前URL直接相关的Cookie。
    *   **新缺陷**：依然无法获取页面动态加载的、来自其他域（如CDN、统计服务）的资源所设置的Cookie。

3.  **V2.3 - 错误的探索 (`debugger.Network.getAllCookies`)**：
    *   **实现**：为了获取所有Cookie，错误地使用了 `chrome.debugger` API 的 `Network.getAllCookies` 命令。
    *   **灾难性后果**：该API获取了**整个浏览器**的所有Cookie，而不是当前页面的，这是一个严重的安全和功能错误。被用户立即指出并纠正。

4.  **V2.4 - 正确的架构 (`debugger` + `Network.requestWillBeSent`)**：
    *   **实现**：在用户的精确指导下，实现了最终的、正确的架构：
        1.  使用 `chrome.debugger.attach` 附加到当前标签页。
        2.  发送 `Network.enable` 命令来开启网络事件监听。
        3.  监听 `Network.requestWillBeSent` 事件，收集页面发出的**所有网络请求**的URL。
        4.  为了确保捕获完整，主动调用 `chrome.tabs.reload()` 刷新页面。
        5.  将收集到的所有URL集合进行去重，然后遍历这个URL列表，对每个URL调用 `chrome.cookies.getAll({ url })`。
        6.  将所有获取到的Cookie结果合并、去重，得到当前页面相关的最完整的Cookie集合。
        7.  在 `finally` 块中确保调用 `chrome.debugger.detach` 来断开连接。
    *   **成果**：这个方案完美解决了数据完整性问题，其获取到的Cookie列表与浏览器开发者工具 "Network" 面板中看到的高度一致。

## V3.0 - UI/UX优化：从“看正确”到“看得爽”

在数据完全准确后，我们进行了最后两项关键的UI/UX优化，使插件达到专业水准。

1.  **智能分组 (`getRegistrableDomain`)**：
    *   **背景**：之前的分组逻辑简单地按 `cookie.domain` 分组，导致 `www.nmc.cn` 和 `typhoon.nmc.cn` 被分为两组，不符合用户直觉。
    *   **实现**：在后台 `background` 脚本中实现了一个 `getRegistrableDomain` 辅助函数。该函数能智能地识别一个域名（如 `typhoon.nmc.cn`）的“可注册域名”（`nmc.cn`），并以此为Key对所有Cookie进行分组。
    *   **效果**：后台直接返回给前端一个已经按主域名分组好的数据结构（如 `{ "nmc.cn": [...] }`），前端逻辑大大简化，用户看到的域名列表也变得极为清晰。

2.  **丰富Cookie详情**：
    *   **背景**：最初的UI只显示Cookie的Name和Value。
    *   **实现**：
        *   改造前端 `App.vue`，在右侧的Cookie列表项中，增加了 **Domain** 和 **Expires** 两列。
        *   增加了一个 `formatExpiry` 辅助函数，将时间戳格式化为可读的本地日期时间字符串（对于Session Cookie则显示 "Session"）。
        *   调整了CSS样式，以适应新的多列布局，保证UI美观。

## 最终状态

经过上述迭代，`CookieSyncer` 插件最终具备了以下特性：

*   **精确的数据**：通过 `debugger` API 实现了与浏览器开发工具同等水平的Cookie捕获能力。
*   **智能的UI**：基于可注册域名的智能分组和丰富的Cookie信息展示，提供了优秀的可用性。
*   **健壮的架构**：清晰的前后台分离、可靠的消息通信和标准的Vite构建流程。

这个过程充分展示了在用户清晰、准确的反馈指导下，一个复杂的技术问题是如何被层层分解、逐步攻克，并最终达成一个完善解决方案的。
