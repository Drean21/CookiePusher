<template>
  <div class="stats-view">
    <div class="sidebar">
      <ul class="domain-list">
        <li v-if="domains.length === 0" class="no-domains">没有已推送的域名</li>
        <li
          v-for="domain in domains"
          :key="domain"
          :class="{ active: selectedDomain === domain }"
          @click="selectedDomain = domain"
        >
          {{ domain }}
        </li>
      </ul>
    </div>
    <div class="content-area">
      <div v-if="!selectedDomain" class="placeholder">请从左侧选择一个域名查看统计</div>
      <div v-else>
        <ul class="cookie-stats-list">
          <li v-for="cookie in sortedCookies" :key="cookie.key" class="cookie-item">
            <div class="cookie-details">
              <div class="cookie-header">
                <strong class="cookie-name">{{ cookie.name }}</strong>
                <span class="cookie-value">{{ cookie.value || "N/A" }}</span>
              </div>
              <div class="detail-row">
                <span>总续期次数:</span>
                <span class="total-count">{{
                  cookie.stats.successCount + cookie.stats.failureCount
                }}</span>
              </div>
              <div class="detail-row">
                <span>当前过期时间:</span>
                <span>{{
                  cookie.expirationDate ? formatTime(cookie.expirationDate) : "Session"
                }}</span>
              </div>
               <div class="history-toggle" v-if="cookie.stats.history && cookie.stats.history.length > 0">
                   <button @click="toggleHistory(cookie.key)">
                   {{ cookie.isHistoryExpanded ? '收起' : '查看' }}活动历史
                   </button>
               </div>
              <div class="history-section" v-if="cookie.isHistoryExpanded">
                <ul class="history-list">
                  <li v-for="(item, index) in cookie.stats.history.slice(0, 5)" :key="index" class="history-item">
                    <div class="history-item-header">
                       <span :class="['history-status', item.status]">{{ formatStatus(item.status) }}</span>
                       <span class="history-timestamp">{{ formatTime(item.timestamp) }}</span>
                    </div>
                    <div class="history-item-body">
                      <span class="history-source">来源: {{ formatChangeSource(item.changeSource) }}</span>
                      <span class="history-interval" v-if="item.intervalSeconds !== undefined">间隔: {{ formatInterval(item.intervalSeconds) }}</span>
                    </div>
                    <div class="history-item-error" v-if="item.status === 'failure' && item.error">
                      原因: {{ item.error }}
                    </div>
                  </li>
                </ul>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { Cookie } from "../../../types/extension";
import { sendMessage } from "../../utils/message";

interface StatHistory {
  status: "success" | "failure" | "no-change";
  timestamp: string;
  changeSource: 'keep-alive' | 'on-change';
  intervalSeconds?: number;
  error?: string;
}

interface CookieStat {
  successCount: number;
  failureCount: number;
  history: StatHistory[];
  expirationDate?: number;
  value?: string;
}

const loading = ref(true);
const syncList = ref<Cookie[]>([]);
const rawStats = ref<{ [key: string]: CookieStat }>({});
const selectedDomain = ref<string | null>(null);
const expandedHistories = ref<Record<string, boolean>>({});

const domains = computed(() => {
  const domainSet = new Set<string>();
  for (const cookie of syncList.value) {
    domainSet.add(getRegistrableDomain(cookie.domain));
  }
  return Array.from(domainSet).sort();
});

const sortedCookies = computed(() => {
  if (!selectedDomain.value) return [];

  const cookiesInDomain = syncList.value.filter(
    (c) => getRegistrableDomain(c.domain) === selectedDomain.value
  );

  const enrichedCookies = cookiesInDomain.map((cookie) => {
    const key = `${cookie.name}|${cookie.domain}|${cookie.path}`;
    const stats = rawStats.value[key] || {
      successCount: 0,
      failureCount: 0,
      history: [],
    };
    return {
      key,
      name: cookie.name,
      value: rawStats.value[key]?.value || cookie.value,
      expirationDate: rawStats.value[key]?.expirationDate || cookie.expirationDate,
      stats,
      isHistoryExpanded: expandedHistories.value[key] || false,
    };
  });

  return enrichedCookies.sort((a, b) => {
    const totalA = a.stats.successCount + a.stats.failureCount;
    const totalB = b.stats.successCount + b.stats.failureCount;
    return totalB - totalA;
  });
});

const formatTime = (timestamp: number | string) => {
  if (!timestamp) return "N/A";
  if (typeof timestamp === "string") {
    return new Date(timestamp).toLocaleString();
  }
  return new Date(timestamp * 1000).toLocaleString();
};

const formatStatus = (status: "success" | "failure" | "no-change") => {
  switch (status) {
    case "success":
      return "成功";
    case "failure":
      return "失败";
    default:
      return "无变化";
  }
};

const formatChangeSource = (source: 'keep-alive' | 'on-change') => {
  switch (source) {
    case 'keep-alive':
      return '后台保活';
    case 'on-change':
      return '监听变更';
    default:
      return '未知';
  }
};

const formatInterval = (seconds: number | undefined) => {
  if (seconds === undefined) return 'N/A';
  if (seconds < 60) return `${seconds}秒`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}分${remainingSeconds}秒`;
};

const toggleHistory = (cookieKey: string) => {
  expandedHistories.value[cookieKey] = !expandedHistories.value[cookieKey];
};

const getRegistrableDomain = (domain: string): string => {
  if (domain.startsWith(".")) domain = domain.substring(1);
  const parts = domain.split(".");
  if (parts.length <= 2) return domain;
  const twoLevelTlds = new Set([
    "com.cn",
    "org.cn",
    "net.cn",
    "gov.cn",
    "co.uk",
    "co.jp",
  ]);
  const lastTwo = parts.slice(-2).join(".");
  if (twoLevelTlds.has(lastTwo) && parts.length > 2) {
    return parts.slice(-3).join(".");
  }
  return lastTwo;
};

onMounted(async () => {
  loading.value = true;
  try {
    const [{ syncList: storedSyncList = [] }, statsResponse] = await Promise.all([
      chrome.storage.local.get("syncList"),
      sendMessage("getKeepAliveStats"),
    ]);

    syncList.value = storedSyncList;
    if (statsResponse.success && statsResponse.stats) {
      rawStats.value = statsResponse.stats;
    }

    if (domains.value.length > 0) {
      selectedDomain.value = domains.value[0];
    }
  } catch (e) {
    console.error("Failed to fetch stats page data:", e);
  } finally {
    loading.value = false;
  }
});
</script>

<style scoped>
.stats-view {
  display: flex;
  width: 100%;
  height: 100%;
}
.sidebar {
  width: 180px;
  flex-shrink: 0;
  border-right: 1px solid #e0e0e0;
  overflow-y: auto;
  background-color: #fafafa;
}
.domain-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.domain-list li {
  padding: 12px 16px;
  cursor: pointer;
  font-size: 14px;
  border-bottom: 1px solid #e0e0e0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.domain-list li:hover {
  background-color: #f0f0f0;
}
.domain-list li.active {
  background-color: #667eea;
  color: white;
  font-weight: 600;
}
.content-area {
  flex-grow: 1;
  overflow-y: auto;
  padding: 16px;
}
.placeholder {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: #999;
  font-size: 16px;
}
.cookie-stats-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.cookie-item {
  background: white;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}
.cookie-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.history-toggle {
  margin-top: 8px;
}

.history-toggle button {
  background: none;
  border: none;
  color: #667eea;
  cursor: pointer;
  font-size: 12px;
  padding: 4px 0;
}

.history-section {
  margin-top: 8px;
}

.history-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.history-item {
  font-size: 12px;
  background-color: #f9f9f9;
  border-radius: 4px;
  padding: 8px;
}
.history-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}
.history-status {
  font-weight: bold;
}
.history-status.success { color: #2e7d32; }
.history-status.failure { color: #c62828; }
.history-status.no-change { color: #757575; }

.history-timestamp {
  color: #777;
}
.history-item-body {
  display: flex;
  justify-content: space-between;
  color: #555;
}
.history-item-error {
  margin-top: 4px;
  color: #c62828;
  word-break: break-all;
}

.cookie-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 4px;
}
.cookie-name {
  font-weight: bold;
  font-size: 16px;
  color: #333;
}
.cookie-value {
  font-family: monospace;
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  max-width: 250px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}
.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-size: 13px;
  color: #555;
}
.error-reason span:last-child {
  text-overflow: ellipsis;
  overflow: hidden;
  white-space: nowrap;
  max-width: 200px;
  cursor: help;
}
.total-count {
  font-weight: bold;
}
.last-status {
  font-style: italic;
}
.last-status.success {
  color: #2e7d32;
}
.last-status.failure {
  color: #c62828;
}
.last-status.no-change {
  color: #757575;
}
.no-domains {
  padding: 16px;
  text-align: center;
  color: #999;
  font-style: italic;
  cursor: default;
}
.no-domains:hover {
  background-color: transparent;
}
</style>
