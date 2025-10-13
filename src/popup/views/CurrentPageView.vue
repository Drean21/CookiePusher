<template>
  <div class="current-view-container">
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>正在获取Cookie...</p>
    </div>

    <div v-else-if="error" class="error-state">
      <p>获取Cookie失败</p>
      <p class="error-message">{{ error }}</p>
      <button @click="fetchCookies" class="action-button">重试</button>
    </div>

    <div v-else-if="totalCookieCount === 0" class="empty-state">
      <p>
        在 <strong>{{ currentTabDomain }}</strong> 没有发现任何Cookie。
      </p>
    </div>

    <div v-else class="two-column-layout">
      <!-- Left Sidebar -->
      <aside class="sidebar">
        <div class="sidebar-header">
          <p>域名分组</p>
        </div>
        <ul>
          <li
            v-for="domain in Object.keys(groupedCookies)"
            :key="domain"
            :class="{ active: domain === selectedDomain }"
            class="domain-item"
          >
            <div class="domain-info" @click="selectedDomain = domain">
              <span class="domain-name">{{ domain }}</span>
              <span class="cookie-count">({{ groupedCookies[domain].length }})</span>
            </div>
            <div class="domain-actions">
              <button
                @click.stop="copyAllCookies(domain)"
                class="action-btn"
                title="复制该域下的所有Cookie (HTTP Header格式)"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                  <path
                    d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                  ></path>
                </svg>
              </button>
              <button
                @click.stop="syncAllCookies(domain)"
                class="action-btn"
                title="将该域下的所有Cookie项全都加入推送序列"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M21 12a9 9 0 0 1-9 9H3" />
                  <path d="M3 12a9 9 0 0 1 9-9h9" />
                  <path d="m16 3 5 5-5 5" />
                  <path d="m8 21-5-5 5-5" />
                </svg>
              </button>
            </div>
          </li>
        </ul>
      </aside>
      <!-- Right Main Content -->
      <section class="main-content">
        <div v-if="selectedDomain" class="cookie-group">
          <div class="search-bar">
            <input v-model="searchQuery" placeholder="筛选Cookie的名称或值..." />
          </div>
          <ul class="cookie-list">
            <li
              v-for="cookie in filteredAndSortedCookies"
              :key="getCookieKey(cookie)"
              class="cookie-item"
            >
              <div class="cookie-info">
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
              </div>
              <div class="cookie-actions">
                <button
                  @click="syncCookie(cookie)"
                  class="action-btn"
                  title="加入推送列表"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="M21 12a9 9 0 0 1-9 9H3" />
                    <path d="M3 12a9 9 0 0 1 9-9h9" />
                    <path d="m16 3 5 5-5 5" />
                    <path d="m8 21-5-5 5-5" />
                  </svg>
                </button>
                <div class="copy-action">
                  <button
                    @click="toggleCopyMenu(getCookieKey(cookie))"
                    class="action-btn"
                    title="复制"
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    >
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                      <path
                        d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                      ></path>
                    </svg>
                  </button>
                  <div v-if="activeCopyMenu === getCookieKey(cookie)" class="copy-menu">
                    <a @click.prevent="copyCookie(cookie, 'value')">复制值</a>
                    <a @click.prevent="copyCookie(cookie, 'name-value')"
                      >复制 Name=Value</a
                    >
                    <a @click.prevent="copyCookie(cookie, 'json')">复制 JSON</a>
                  </div>
                </div>
              </div>
            </li>
          </ul>
          <div
            v-if="filteredAndSortedCookies.length === 0 && selectedDomain"
            class="empty-state-inner"
          >
            <p>没有匹配的Cookie。</p>
          </div>
        </div>
        <div v-else class="empty-state">
          <p>请在左侧选择一个域名以查看Cookies。</p>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted, ref } from "vue";
import type { Cookie } from "../../../types/extension.d";
import { sendMessage } from "../../utils/message";

type ShowNotification = (
  message: string,
  type?: "success" | "error",
  duration?: number
) => void;
const showNotification = inject<ShowNotification>("showNotification", () => {});

interface GroupedCookies {
  [domain: string]: Cookie[];
}

const loading = ref(true);
const error = ref<string | null>(null);
const groupedCookies = ref<GroupedCookies>({});
const selectedDomain = ref<string | null>(null);
const currentTabDomain = ref<string>("当前标签页");
const searchQuery = ref("");
const activeCopyMenu = ref<string | null>(null);

const totalCookieCount = computed(() =>
  Object.values(groupedCookies.value).reduce((sum, cookies) => sum + cookies.length, 0)
);

const filteredAndSortedCookies = computed(() => {
  if (!selectedDomain.value || !groupedCookies.value[selectedDomain.value]) {
    return [];
  }
  let cookies = groupedCookies.value[selectedDomain.value];
  if (searchQuery.value) {
    const lowerCaseQuery = searchQuery.value.toLowerCase();
    cookies = cookies.filter(
      (cookie) =>
        cookie.name.toLowerCase().includes(lowerCaseQuery) ||
        cookie.value.toLowerCase().includes(lowerCaseQuery)
    );
  }
  return [...cookies].sort((a, b) => a.name.localeCompare(b.name));
});

function getCookieKey(cookie: Cookie): string {
  return cookie.name + cookie.domain + cookie.path;
}

function toggleCopyMenu(cookieKey: string) {
  activeCopyMenu.value = activeCopyMenu.value === cookieKey ? null : cookieKey;
}

async function copyCookie(cookie: Cookie, format: "value" | "name-value" | "json") {
  let textToCopy = "";
  switch (format) {
    case "value":
      textToCopy = cookie.value;
      break;
    case "name-value":
      textToCopy = `${cookie.name}=${cookie.value}`;
      break;
    case "json":
      textToCopy = JSON.stringify(cookie, null, 2);
      break;
  }
  await navigator.clipboard.writeText(textToCopy);
  activeCopyMenu.value = null;
}

async function syncCookie(cookie: Cookie) {
  try {
    const response = await sendMessage("syncSingleCookie", { cookie });
    if (response.success) {
      showNotification(`已添加 ${cookie.name}`, "success");
    }
  } catch (e: any) {
    showNotification(`添加失败: ${e.message}`, "error");
  }
}

async function syncAllCookies(domain: string) {
  const cookiesToSync = groupedCookies.value[domain];
  if (cookiesToSync?.length > 0) {
    try {
      const response = await sendMessage("syncAllCookiesForDomain", {
        cookies: cookiesToSync,
      });
      if (response.success) {
        showNotification(
          `已添加 ${domain} 下的 ${cookiesToSync.length} 个Cookie`,
          "success"
        );
      }
    } catch (e: any) {
      showNotification(`添加失败: ${e.message}`, "error");
    }
  }
}

async function copyAllCookies(domain: string) {
  const cookiesToCopy = groupedCookies.value[domain];
  if (cookiesToCopy?.length > 0) {
    const textToCopy = cookiesToCopy.map((c) => `${c.name}=${c.value}`).join("; ");
    await navigator.clipboard.writeText(textToCopy);
  }
}

function formatExpiry(expirationDate?: number): string {
  if (expirationDate === undefined) return "Session";
  return new Date(expirationDate * 1000).toLocaleString();
}

async function fetchCookies() {
  loading.value = true;
  error.value = null;
  searchQuery.value = "";
  selectedDomain.value = null;

  try {
    const response = await sendMessage("getCookiesForCurrentTab");
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
    console.error("获取Cookie失败:", e);
    error.value = e.message || "发生未知错误。";
  } finally {
    loading.value = false;
  }
}

onMounted(fetchCookies);
</script>

<style scoped>
.current-view-container {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 12px;
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
  display: flex;
  flex-direction: column;
}
.sidebar-header {
  padding: 12px;
  border-bottom: 1px solid #e0e0e0;
  text-align: center;
  font-weight: 600;
  color: #424242;
}
.sidebar ul {
  list-style: none;
  padding: 8px;
  margin: 0;
  overflow-y: auto;
  flex-grow: 1;
}
.sidebar li {
  padding: 10px 12px;
  cursor: pointer;
  border-radius: 4px;
  margin-bottom: 4px;
  font-size: 14px;
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
  margin-left: 4px;
}
.main-content {
  flex: 1;
  height: 100%;
  overflow-y: auto;
  padding: 0 12px;
}
.cookie-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.cookie-item {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  padding: 10px;
  border-radius: 6px;
  background-color: #fff;
  margin-bottom: 8px;
  border: 1px solid #e0e0e0;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}
.cookie-info {
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
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
.loading-state,
.error-state,
.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: #757575;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}
.empty-state-inner {
  padding: 20px;
  text-align: center;
  color: #757575;
}
.error-message {
  background-color: #fff0f0;
  color: #d8000c;
  border: 1px solid #ffd2d2;
  padding: 10px;
  border-radius: 4px;
  font-family: monospace;
}
.action-button {
  background-color: #667eea;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 6px;
  font-size: 16px;
  cursor: pointer;
  transition: background-color 0.2s;
}
.action-button:hover {
  background-color: #5a6ed0;
}
.domain-info {
  flex-grow: 1;
  overflow: hidden;
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.domain-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 500;
}
.domain-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.search-bar {
  padding: 0 0 12px 0;
}
.search-bar input {
  width: 100%;
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid #ccc;
  font-size: 14px;
}
.cookie-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 4px;
  flex-shrink: 0;
}
.action-btn {
  background: none;
  border: 1px solid transparent;
  padding: 4px;
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #555;
}
.action-btn:hover {
  background-color: #f0f2f5;
  color: #333;
}
.copy-action {
  position: relative;
}
.copy-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 4px;
  background-color: white;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  z-index: 10;
  width: 150px;
  padding: 6px;
  display: flex;
  flex-direction: column;
}
.copy-menu a {
  padding: 8px 12px;
  font-size: 13px;
  color: #333;
  border-radius: 4px;
  text-decoration: none;
  white-space: nowrap;
}
.copy-menu a:hover {
  background-color: #f0f2f5;
  cursor: pointer;
}
.spinner {
  border: 4px solid #f3f3f3;
  border-top: 4px solid #667eea;
  border-radius: 50%;
  width: 30px;
  height: 30px;
  animation: spin 1s linear infinite;
}
@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}
</style>
