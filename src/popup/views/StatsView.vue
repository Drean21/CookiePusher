<template>
  <div class="stats-view">
    <div class="sidebar">
      <div class="sidebar-header">
        <h3>已推送域名</h3>
      </div>
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
      <div v-if="!selectedDomain" class="placeholder">
        请从左侧选择一个域名查看统计数据
      </div>
      <div v-else>
        <div class="controls">
          <div class="sort-control">
            <label for="sort-order">排序方式:</label>
            <select id="sort-order" v-model="sortOrder">
              <option value="health">按健康度</option>
              <option value="lastActivity">按最后活动时间</option>
              <option value="failureCount">按失败次数</option>
              <option value="name">按名称</option>
            </select>
          </div>
          <button @click="clearDomainStats" class="clear-btn" title="清除当前域名的所有统计数据">清除本域统计</button>
        </div>
        <ul class="cookie-stats-list">
          <li
            v-for="cookie in sortedCookies"
            :key="cookie.key"
            class="cookie-card"
            @click="toggleDetails(cookie.key)"
          >
            <div class="card-header">
              <strong class="cookie-name">{{ cookie.name }}</strong>
              <div class="health-status">
                <span :class="['indicator', getHealthStatus(cookie).status]"></span>
                <span>{{ getHealthStatus(cookie).text }}</span>
              </div>
            </div>
            <div class="last-activity">
              <span>最后活动:</span>
              <span>{{
                cookie.stats.history.length > 0
                  ? formatTime(cookie.stats.history[0].timestamp)
                  : "无记录"
              }}</span>
            </div>
            <transition name="slide-fade">
              <div v-if="cookie.isDetailsExpanded" class="card-details">
                <div class="details-summary">
                  <span
                    ><strong>总计:</strong>
                    {{ cookie.stats.successCount + cookie.stats.failureCount }}</span
                  >
                  <span class="success-count"
                    ><strong>成功:</strong> {{ cookie.stats.successCount }}</span
                  >
                  <span class="failure-count"
                    ><strong>失败:</strong> {{ cookie.stats.failureCount }}</span
                  >
                </div>
                <div class="detail-row">
                  <span><strong>当前值:</strong></span>
                  <span class="cookie-value">{{ cookie.value || "N/A" }}</span>
                </div>
                <div class="detail-row">
                  <span><strong>过期时间:</strong></span>
                  <span>{{
                    cookie.expirationDate ? formatTime(cookie.expirationDate) : "Session"
                  }}</span>
                </div>
                <div class="history-section">
                  <h4>最近活动 <span class="history-limit-notice">（仅保留最近20条）</span></h4>
                  <ul v-if="cookie.stats.history.length > 0" class="history-list">
                    <li
                      v-for="(item, index) in cookie.stats.history.slice(0, 5)"
                      :key="index"
                      class="history-item"
                    >
                      <span :class="['history-status', item.status]">{{
                        formatStatus(item.status)
                      }}</span>
                      <span class="history-source"
                        >({{ formatChangeSource(item.changeSource) }})</span
                      >
                      <span class="history-timestamp">{{
                        formatTime(item.timestamp)
                      }}</span>
                      <div
                        class="history-item-error"
                        v-if="item.status === 'failure' && item.error"
                      >
                        原因: {{ item.error }}
                      </div>
                    </li>
                  </ul>
                  <p v-else>没有活动历史记录。</p>
                </div>
              </div>
            </transition>
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
const expandedDetails = ref<Record<string, boolean>>({});
const sortOrder = ref<'health' | 'lastActivity' | 'failureCount' | 'name'>('health');

const domains = computed(() => {
  const domainSet = new Set<string>();
  syncList.value.forEach(cookie => {
    domainSet.add(getRegistrableDomain(cookie.domain));
  });
  return Array.from(domainSet).sort();
});

const getHealthStatus = (cookie: any): { status: string; text: string; level: number } => {
  if (cookie.stats.failureCount > 0) {
    if (cookie.stats.history.length > 0 && cookie.stats.history[0].status === 'failure') {
      return { status: 'failure', text: '失败', level: 2 };
    }
    return { status: 'warning', text: '警告', level: 1 };
  }
  if (cookie.stats.successCount > 0 || (cookie.stats.history.length > 0 && cookie.stats.history[0].status === 'success')) {
    return { status: 'ok', text: '健康', level: 0 };
  }
  return { status: 'unknown', text: '未知', level: 3 };
};

const sortedCookies = computed(() => {
  if (!selectedDomain.value) return [];

  const cookiesInDomain = syncList.value.filter(
    (c) => getRegistrableDomain(c.domain) === selectedDomain.value
  );

  const enrichedCookies = cookiesInDomain.map((cookie) => {
    const key = `${cookie.name}|${cookie.domain}|${cookie.path}`;
    const stats = rawStats.value[key] || { successCount: 0, failureCount: 0, history: [] };
    return {
      key,
      name: cookie.name,
      value: rawStats.value[key]?.value || cookie.value,
      expirationDate: rawStats.value[key]?.expirationDate || cookie.expirationDate,
      stats,
      isDetailsExpanded: expandedDetails.value[key] || false,
    };
  });

  return enrichedCookies.sort((a, b) => {
    switch (sortOrder.value) {
      case 'health':
        return getHealthStatus(a).level - getHealthStatus(b).level;
      case 'lastActivity':
        const lastActivityA = a.stats.history.length > 0 ? new Date(a.stats.history[0].timestamp).getTime() : 0;
        const lastActivityB = b.stats.history.length > 0 ? new Date(b.stats.history[0].timestamp).getTime() : 0;
        return lastActivityB - lastActivityA;
      case 'failureCount':
        return b.stats.failureCount - a.stats.failureCount;
      case 'name':
        return a.name.localeCompare(b.name);
      default:
        return 0;
    }
  });
});

const formatTime = (timestamp?: number | string) => {
  if (!timestamp) return "N/A";
  const date = typeof timestamp === 'string' ? new Date(timestamp) : new Date(timestamp * 1000);
  return date.toLocaleString();
};

const formatStatus = (status: "success" | "failure" | "no-change") => {
  const map = { success: "成功", failure: "失败", 'no-change': "无变化" };
  return map[status] || '未知';
};

const formatChangeSource = (source: 'keep-alive' | 'on-change') => {
  const map = { 'keep-alive': "后台保活", 'on-change': "监听变更" };
  return map[source] || '未知';
};

const toggleDetails = (cookieKey: string) => {
  expandedDetails.value[cookieKey] = !expandedDetails.value[cookieKey];
};

const clearDomainStats = async () => {
  if (!selectedDomain.value) return;
  if (window.confirm(`确定要清除域名 "${selectedDomain.value}" 下的所有统计数据吗？此操作不可撤销。`)) {
    try {
      const response = await sendMessage("clearDomainStats", { domain: selectedDomain.value });
      if (response.success) {
        // Optimistically clear the stats in the UI
        const newRawStats = { ...rawStats.value };
        for (const key in newRawStats) {
          if (getRegistrableDomain(key.split('|')[1]) === selectedDomain.value) {
            delete newRawStats[key];
          }
        }
        rawStats.value = newRawStats;
        alert("统计数据已清除。");
      }
    } catch (e: any) {
      alert(`清除失败: ${e.message}`);
    }
  }
};

const getRegistrableDomain = (domain: string): string => {
  if (domain.startsWith(".")) domain = domain.substring(1);
  const parts = domain.split(".");
  if (parts.length <= 2) return domain;
  const twoLevelTlds = new Set(["com.cn", "org.cn", "net.cn", "gov.cn", "co.uk", "co.jp"]);
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
  background-color: #f4f5f7;
}
.sidebar {
  width: 180px;
  flex-shrink: 0;
  border-right: 1px solid #e0e0e0;
  overflow-y: auto;
  background-color: #fff;
}
.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid #e0e0e0;
}
.sidebar-header h3 {
  margin: 0;
  font-size: 16px;
  color: #333;
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
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  border-bottom: 1px solid #e8e8e8;
}
.domain-list li:hover {
  background-color: #f0f5ff;
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
.controls {
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 16px;
  justify-content: space-between;
}
.sort-control {
  display: flex;
  align-items: center;
  gap: 8px;
}
.sort-control label {
  font-size: 14px;
  font-weight: 500;
}
.sort-control select {
  padding: 6px 10px;
  border-radius: 6px;
  border: 1px solid #ccc;
}
.clear-btn {
  padding: 6px 12px;
  border-radius: 6px;
  border: 1px solid #ff4d4f;
  background-color: #fff1f0;
  color: #ff4d4f;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s;
}
.clear-btn:hover {
  background-color: #ff4d4f;
  color: white;
}
.cookie-stats-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.cookie-card {
  background: white;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  cursor: pointer;
  transition: box-shadow 0.2s;
}
.cookie-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.cookie-name {
  font-weight: 600;
  font-size: 16px;
  color: #333;
}
.health-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
}
.health-status .indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.health-status .indicator.ok {
  background-color: #52c41a;
}
.health-status .indicator.warning {
  background-color: #faad14;
}
.health-status .indicator.failure {
  background-color: #f5222d;
}
.health-status .indicator.unknown {
  background-color: #bfbfbf;
}
.last-activity {
  font-size: 12px;
  color: #888;
  display: flex;
  justify-content: space-between;
}
.card-details {
  margin-top: 16px;
  border-top: 1px solid #f0f0f0;
  padding-top: 16px;
}
.details-summary {
  display: flex;
  justify-content: space-around;
  background-color: #fafafa;
  padding: 8px;
  border-radius: 6px;
  margin-bottom: 12px;
  font-size: 13px;
}
.success-count {
  color: #2e7d32;
}
.failure-count {
  color: #c62828;
}
.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-size: 13px;
  color: #555;
  padding: 4px 0;
}
.cookie-value {
  font-family: monospace;
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}
.history-section {
  margin-top: 12px;
}
.history-section h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: 600;
}
.history-limit-notice {
  font-size: 11px;
  color: #999;
  font-weight: 400;
}
.history-list {
  list-style: none;
  padding: 0;
  margin: 0;
  font-size: 12px;
}
.history-item {
  padding: 6px 0;
  border-bottom: 1px solid #f0f0f0;
}
.history-item:last-child {
  border-bottom: none;
}
.history-status {
  font-weight: bold;
}
.history-status.success {
  color: #2e7d32;
}
.history-status.failure {
  color: #c62828;
}
.history-timestamp {
  color: #777;
  float: right;
}
.history-source {
  color: #555;
  margin-left: 8px;
}
.history-item-error {
  margin-top: 4px;
  color: #c62828;
  font-size: 11px;
  word-break: break-all;
  background-color: #fff1f0;
  padding: 4px;
  border-radius: 4px;
}
.slide-fade-enter-active {
  transition: all 0.3s ease-in-out;
}
.slide-fade-leave-active {
  transition: all 0.3s cubic-bezier(1, 0.5, 0.8, 1);
}
.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateY(-10px);
  opacity: 0;
}
</style>
