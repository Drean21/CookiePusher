<template>
  <div class="popup-container">
    <header class="popup-header">
      <h1>CookieSyncer</h1>
    </header>
    <main class="popup-content" :class="{ 'view-current': activeView === 'current' }">
      <!-- 当前页面 Cookies -->
      <div v-if="activeView === 'current'" class="current-view-container">
        <div v-if="loading" class="loading-state">正在捕获网络活动并智能分组...</div>
        <div v-else-if="error" class="error-state">{{ error }}</div>
        <div v-else-if="totalCookieCount === 0" class="empty-state">
          <p>
            在 <strong>{{ currentTabDomain }}</strong> 没有发现任何Cookie。
          </p>
        </div>
        <div v-else class="two-column-layout">
          <!-- Left Sidebar -->
          <aside class="sidebar">
            <ul>
              <li
                v-for="domain in Object.keys(groupedCookies)"
                :key="domain"
                :class="{ active: domain === selectedDomain }"
                @click="selectedDomain = domain"
              >
                {{ domain }}
                <span class="cookie-count">({{ groupedCookies[domain].length }})</span>
              </li>
            </ul>
          </aside>
          <!-- Right Main Content -->
          <section class="main-content">
            <div v-if="selectedDomainData" class="cookie-group">
              <ul class="cookie-list">
                <li
                  v-for="cookie in selectedDomainData"
                  :key="cookie.name + cookie.domain + cookie.path"
                  class="cookie-item"
                >
                  <div class="cookie-main">
                    <strong class="cookie-name">{{ cookie.name }}</strong>
                    <span class="cookie-value">{{ cookie.value }}</span>
                  </div>
                  <div class="cookie-details">
                    <div class="detail-item">
                      <strong>Domain:</strong> <span>{{ cookie.domain }}</span>
                    </div>
                    <div class="detail-item">
                      <strong>Expires:</strong>
                      <span>{{ formatExpiry(cookie.expirationDate) }}</span>
                    </div>
                  </div>
                </li>
              </ul>
            </div>
            <div v-else class="empty-state">
              <p>请在左侧选择一个域名以查看Cookies。</p>
            </div>
          </section>
        </div>
      </div>
      <!-- 已管理域名 -->
      <div v-if="activeView === 'managed'">
        <p>已管理域名视图（待实现）</p>
      </div>
      <!-- 设置 -->
      <div v-if="activeView === 'settings'">
        <p>设置视图（待实现）</p>
      </div>
    </main>
    <footer class="popup-footer">
      <nav class="tab-nav">
        <a @click="activeView = 'current'" :class="{ active: activeView === 'current' }"
          >当前页面</a
        >
        <a @click="activeView = 'managed'" :class="{ active: activeView === 'managed' }"
          >域名管理</a
        >
        <a @click="activeView = 'settings'" :class="{ active: activeView === 'settings' }"
          >设置</a
        >
      </nav>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { Cookie } from "../../types/extension.d";
import { sendMessage } from "../utils/message";

type View = "current" | "managed" | "settings";
interface GroupedCookies {
  [domain: string]: Cookie[];
}

const activeView = ref<View>("current");
const loading = ref(true);
const error = ref<string | null>(null);
const groupedCookies = ref<GroupedCookies>({});
const selectedDomain = ref<string | null>(null);
const currentTabDomain = ref<string>("当前标签页");

const totalCookieCount = computed(() =>
  Object.values(groupedCookies.value).reduce((sum, cookies) => sum + cookies.length, 0)
);

const selectedDomainData = computed(() => {
  if (!selectedDomain.value || !groupedCookies.value[selectedDomain.value]) {
    return [];
  }
  return [...groupedCookies.value[selectedDomain.value]].sort((a, b) =>
    a.name.localeCompare(b.name)
  );
});

function formatExpiry(expirationDate?: number): string {
  if (expirationDate === undefined) {
    return "Session";
  }
  const date = new Date(expirationDate * 1000);
  return date.toLocaleString();
}

onMounted(async () => {
  try {
    loading.value = true;
    error.value = null;

    const response = await sendMessage("getCurrentTabCookies");

    if (response.success) {
      groupedCookies.value = response.groupedCookies || {};
      currentTabDomain.value = response.domain || "未知域名";

      const domains = Object.keys(groupedCookies.value);
      if (domains.length > 0) {
        selectedDomain.value = domains[0];
      }
    } else {
      throw new Error(response.error || "获取Cookie失败。");
    }
  } catch (e: any) {
    console.error("初始化失败:", e);
    error.value = e.message || "发生未知错误。";
  } finally {
    loading.value = false;
  }
});
</script>

<style>
html,
body {
  margin: 0;
  padding: 0;
  width: 600px;
  min-height: 500px;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue",
    Arial, sans-serif;
  color: #333;
}
.popup-container {
  display: flex;
  flex-direction: column;
  width: 600px;
  height: 500px;
  background-color: #f9f9f9;
}
.popup-header {
  padding: 12px;
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}
.popup-header h1 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #424242;
}
.popup-content {
  flex-grow: 1;
  overflow: hidden;
  display: flex;
}
.popup-content.view-current {
  padding: 0;
}
.current-view-container {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
}
.two-column-layout {
  display: flex;
  width: 100%;
  height: 100%;
  align-items: flex-start;
}
.sidebar {
  width: 200px;
  height: 100%;
  border-right: 1px solid #e0e0e0;
  background-color: #fafafa;
  overflow-y: auto;
}
.sidebar ul {
  list-style: none;
  padding: 8px;
  margin: 0;
}
.sidebar li {
  padding: 10px 12px;
  cursor: pointer;
  border-radius: 4px;
  margin-bottom: 4px;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.sidebar li:hover {
  background-color: #f0f0f0;
}
.sidebar li.active {
  background-color: #e0e7ff;
  color: #4f46e5;
  font-weight: 600;
}
.sidebar li span.cookie-count {
  font-size: 12px;
  color: #6b7280;
  font-weight: normal;
}
.main-content {
  flex: 1;
  height: 100%;
  overflow-y: auto;
  padding: 12px;
}
.cookie-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.cookie-item {
  display: flex;
  flex-direction: column;
  padding: 10px;
  border-radius: 6px;
  background-color: #fff;
  margin-bottom: 8px;
  border: 1px solid #e0e0e0;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  gap: 8px;
}
.cookie-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.cookie-name {
  font-weight: 600;
  font-size: 14px;
  color: #333;
}
.cookie-value {
  font-size: 13px;
  color: #555;
  word-break: break-all;
  background-color: #f5f5f5;
  padding: 6px 8px;
  border-radius: 4px;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, Courier, monospace;
}
.cookie-details {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 4px 8px;
  font-size: 12px;
  color: #666;
  border-top: 1px solid #f0f0f0;
  padding-top: 8px;
}
.cookie-details .detail-item strong {
  font-weight: 500;
  color: #333;
}
.cookie-details .detail-item span {
  word-break: break-all;
}
.loading-state,
.error-state,
.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: #757575;
  width: 100%;
}
.empty-state .tip {
  font-size: 12px;
  color: #9e9e9e;
}
.popup-footer {
  border-top: 1px solid #e0e0e0;
  background-color: #fff;
  box-shadow: 0 -1px 3px rgba(0, 0, 0, 0.05);
}
.tab-nav {
  display: flex;
  justify-content: space-around;
  padding: 4px 0;
}
.tab-nav a {
  flex: 1;
  padding: 10px 12px;
  cursor: pointer;
  transition: all 0.2s ease-in-out;
  color: #555;
  text-decoration: none;
  text-align: center;
  font-size: 14px;
  font-weight: 500;
}
.tab-nav a:hover {
  background-color: #f0f2f5;
}
.tab-nav a.active {
  color: #667eea;
  border-bottom: 3px solid #667eea;
  background-color: #f0f2f5;
}
</style>
