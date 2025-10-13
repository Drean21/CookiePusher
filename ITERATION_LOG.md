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

## V4.0 - 功能增强与终极重构

在V3版本的基础上，我们进行了一系列的功能增强和架构上的终极探索与重构。

### V4.1 - 功能增强

1.  **搜索/筛选**：在右侧内容区顶部增加了搜索框，可以根据Cookie的名称或值进行实时筛选。
2.  **多格式复制**：
    *   **单个复制**：为每个Cookie项增加了复制按钮，并提供下拉菜单，支持复制“值”、“Name=Value”格式或完整的“JSON”对象。
    *   **批量复制**：为左侧每个域名分组增加了“全部复制”按钮，可以一键复制该域名下所有Cookie的HTTP请求头格式字符串 (`key1=value1; key2=value2; ...`)。
3.  **预推送功能**：
    *   为单个Cookie和整个域名分组都增加了“推送”按钮。
    *   点击后，会将相应的Cookie数据发送到后台，并使用`chrome.storage.local`进行持久化存储，为后续真正的跨浏览器推送功能打下了数据基础。

### V4.2 - 架构重构：探索与回归

这个阶段，我们为了解决两个核心问题（数据精确性 vs. 用户体验）进行了多次深刻的架构重构。

1.  **前端组件化 (V4.2.1)**：
    *   **动机**：解决所有逻辑都堆积在 `App.vue` 中导致文件臃肿、难以维护的问题。
    *   **实现**：将`App.vue`重构为应用外壳（Shell），负责通用布局和导航。将“当前页面”、“域名管理”、“设置”等拆分为独立的视图组件（`CurrentPageView.vue`等），实现了清晰的关注点分离。

2.  **“主动分析”模型 (V4.2.2 - 失败的尝试)**：
    *   **动机**：解决旧版`debugger`方案每次打开弹窗都强制刷新页面的灾难性用户体验。
    *   **实现**：将重量级的数据获取操作改为由用户手动点击“分析”按钮触发。分析结果被存入`storage`，弹窗打开时读取缓存，从而实现UI秒开。
    *   **缺陷**：虽然解决了UX问题，但用户反馈其底层的`debugger` + 网络监听方案，在某些复杂页面（如`zread.ai`）依然无法获取到所有`iframe`中的Cookie，数据精确性问题依然存在。

3.  **“全局扫描 + 精准过滤”模型 (V4.2.3 - 错误的实现)**：
    *   **动机**：为了解决`debugger`方案的数据遗漏问题，尝试模拟浏览器“Application”面板的原理。
    *   **实现**：收集页面所有相关域（主文档、iframe、网络请求），然后对**每个域**分别调用`chrome.cookies.getAll({ domain })`。
    *   **缺陷**：此路不通。用户的测试再次证明，这种方式依然遗漏了跨域`iframe`中的Cookie。这暴露了我们对Cookie“可见性”和`getAll` API工作原理的理解存在根本性偏差。

4.  **终极模型：“全局扫描 + `scripting`注入” (V4.2.4 - 最终方案)**：
    *   **顿悟**：在您的不懈指正下，我们终于意识到，所有基于“先猜有哪些域，再去查”的思路都是错误的。正确的思路是“**先获取全部，再根据权威来源过滤**”。
    *   **实现**：
        1.  **彻底放弃 `debugger`**：该API副作用太大，且不适合此场景。
        2.  **获取权威域列表**：使用现代、无干扰的 `chrome.scripting.executeScript({ allFrames: true })` API，直接向页面上**所有**活跃的框架（包括主文档和所有`iframe`）注入脚本，获取它们各自的 `document.domain`。这是最权威、最不可能出错的“域”列表。
        3.  **全局扫描与过滤**：通过 `chrome.cookies.getAll({})` 一次性获取浏览器存储的**全部Cookie**，然后用一个严谨的“可见性”过滤算法，从这个全局Cookie池中，精确地筛选出对上述权威域列表可见的Cookie。
    *   **结果**：此方案从根本上统一了数据精确性（虽然`z.ai`案例仍有遗憾）和用户体验。

5.  **最终简化 (V4.2.5)**：
    *   **决策**：尽管V4.2.4在技术上最先进，但在`z.ai`案例上仍未完美。遵从您的最终指示，我们在“完美数据”和“完美体验”之间做出权衡，优先保证用户体验。
    *   **实现**：保留了V4.2.4的后台无干扰分析技术，但在前端去除了所有“手动分析/刷新”的UI元素，回归到“**打开即自动分析**”的简洁模式。这成为了插件最终交付的状态。

### 结论
整个V4的迭代过程，是一次从功能堆砌到架构反思，再到对浏览器底层原理不断深挖的宝贵历程。虽然在追求数据绝对精确的道路上充满波折，但最终在您的指导下，我们得到了一个在架构、体验和现有技术能力下达到最佳平衡的最终产品。

## V5.0 - Cookie保活功能（Keep-Alive）与架构修复

在核心查看功能稳定后，我们开始实现插件的另一大核心能力：Cookie续期/保活。

### V5.1 - Offscreen Iframe 方案

*   **动机**：实现一个无侵入的后台定时任务，通过静默访问目标网站来刷新相关Cookie的有效期。
*   **实现**：
    1.  使用`chrome.alarms` API 创建一个周期性后台任务。
    2.  任务触发时，使用 Manifest V3 引入的 `chrome.offscreen.createDocument` API 创建一个离屏文档。
    3.  在离屏文档中，动态创建 `iframe`，将其 `src` 指向需要保活的域名。
    4.  通过前后两次读取 Cookie 并对比时间戳，来验证保活是否成功。

### V5.2 - 架构级 Bug 修复

*   **问题 A: `Page failed to load`**
    *   **现象**：保活任务触发时，后台脚本抛出未捕获的 Promise 错误，导致任务中断。
    *   **根源**：`manifest.json` 中设置了严格的`"script-src 'self'"`内容安全策略（CSP），这禁止了 `offscreen.html` 中所有内联脚本的执行。
    *   **修复**：将 `offscreen.html` 的内联脚本分离到独立的 `offscreen.js` 文件，并通过 `vite.config.ts` 将其作为新的入口点正确打包。同时，为 `handleKeepAlive` 函数增加了 `try-catch` 块，增强了健壮性。

*   **问题 B: `Cannot read properties of undefined (reading 'get')`**
    *   **现象**：在解决了CSP问题后，Offscreen 文档内部又抛出大量JS错误。
    *   **根源**：这是一个更深层次的架构问题。`chrome.cookies` API **在 Offscreen Document 环境中是不可用的**。
    *   **修复**：对保活功能进行彻底的架构重构。
        1.  **后台负责数据**：所有 `chrome.cookies.get()` 操作（包括生成前快照和后快照）全部移回拥有完整权限的后台脚本 (`background/index.ts`) 中执行。
        2.  **Offscreen 负责加载**：Offscreen 文档的职责被极大简化，它不再关心任何 Cookie 数据，只负责接收 URL 列表，创建 `iframe`，并在加载完成后通过消息通知后台。
        3.  **消息驱动流程**：后台脚本在收到 Offscreen 的完成消息后，才执行后续的快照对比和日志记录工作。

*   **结果**：经过这次彻底的重构，Cookie保活功能最终在架构正确、逻辑清晰、运行稳定的状态下得以实现。
