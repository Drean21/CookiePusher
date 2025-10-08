<template>
  <div class="stats-view">
    <div class="sidebar">
      <ul class="domain-list">
        <li
          v-for="(cookies, domain) in groupedStats"
          :key="domain"
          :class="{ active: selectedDomain === domain }"
          @click="selectedDomain = String(domain)"
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
              <div class="detail-row" v-if="cookie.stats.history.length > 0">
                <span>上次续期:</span>
                <span :class="['last-status', cookie.stats.history[0].status]"
                  >{{ formatStatus(cookie.stats.history[0].status) }} @
                  {{ formatTime(cookie.stats.history[0].timestamp) }}</span
                >
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
import { sendMessage } from "../../utils/message";

interface StatHistory {
  status: "success" | "failure" | "no-change";
  timestamp: string;
}

interface CookieStat {
  successCount: number;
  failureCount: number;
  history: StatHistory[];
  expirationDate?: number;
  value?: string;
}

interface GroupedCookieStat {
  key: string;
  name: string;
  domain: string;
  path: string;
  stats: CookieStat;
  expirationDate?: number;
  value?: string;
}

const loading = ref(true);
const rawStats = ref<{ [key: string]: CookieStat }>({});
const selectedDomain = ref<string | null>(null);

const groupedStats = computed(() => {
  const groups: { [domain: string]: GroupedCookieStat[] } = {};
  for (const key in rawStats.value) {
    const [name, domain, path] = key.split("|");
    const registrableDomain = getRegistrableDomain(domain);
    if (!groups[registrableDomain]) {
      groups[registrableDomain] = [];
    }
    groups[registrableDomain].push({
      key,
      name,
      domain,
      path,
      stats: rawStats.value[key],
      expirationDate: rawStats.value[key].expirationDate,
      value: rawStats.value[key].value,
    });
  }
  return groups;
});

const sortedCookies = computed(() => {
  if (!selectedDomain.value || !groupedStats.value[selectedDomain.value]) {
    return [];
  }
  return groupedStats.value[selectedDomain.value].sort((a, b) => {
    const totalA = a.stats.successCount + a.stats.failureCount;
    const totalB = b.stats.successCount + b.stats.failureCount;
    return totalB - totalA; // Sort descending by total activity
  });
});

const formatTime = (timestamp: number | string) => {
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

const getRegistrableDomain = (domain: string): string => {
  if (domain.startsWith(".")) domain = domain.substring(1);
  const parts = domain.split(".");
  if (parts.length <= 2) return domain;
  // This is a simplified logic, a robust solution would use a public suffix list
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
  try {
    const response = await sendMessage("getKeepAliveStats");
    if (response.success && response.stats) {
      rawStats.value = response.stats;
      // Auto-select the first domain if available
      const firstDomain = Object.keys(groupedStats.value)[0];
      if (firstDomain) {
        selectedDomain.value = firstDomain;
      }
    }
  } catch (e) {
    console.error("Failed to fetch stats:", e);
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
</style>
