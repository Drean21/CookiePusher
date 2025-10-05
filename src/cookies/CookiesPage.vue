<template>
  <div class="cookies-page">
    <header class="page-header">
      <h1>Cookie 管理</h1>
      <div class="header-actions">
        <button @click="refreshCookies" class="btn btn-primary">刷新</button>
      </div>
    </header>

    <div class="search-section">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="搜索域名或Cookie名称..."
        class="search-input"
      />
    </div>

    <div class="domains-section">
      <h2>域名列表</h2>
      <div class="domain-list">
        <div v-for="domain in filteredDomains" :key="domain.domain" class="domain-card">
          <div class="domain-header" @click="toggleDomain(domain.domain)">
            <span class="domain-name">{{ domain.domain }}</span>
            <span class="cookie-count">
              {{ getDomainCookieCount(domain.domain) }} 个Cookie
            </span>
            <span class="expand-icon">
              {{ expandedDomains.has(domain.domain) ? "▼" : "▶" }}
            </span>
          </div>

          <div v-if="expandedDomains.has(domain.domain)" class="cookies-list">
            <div
              v-for="cookie in getDomainCookies(domain.domain)"
              :key="cookie.name + cookie.domain"
              class="cookie-item"
              @click="showCookieDetails(cookie)"
            >
              <span class="cookie-name">{{ cookie.name }}</span>
              <span class="cookie-value">{{ truncateValue(cookie.value) }}</span>
              <span class="cookie-expires">{{
                formatExpiration(cookie.expirationDate)
              }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Cookie详情模态框 -->
    <div v-if="selectedCookie" class="modal-overlay" @click="closeModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>Cookie 详情</h3>
          <button @click="closeModal" class="close-btn">×</button>
        </div>
        <div class="modal-body">
          <CookieDetails :cookie="selectedCookie" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { Cookie, DomainConfig } from "../../types/extension";
import { sendMessage } from "../utils/message";
import CookieDetails from "./components/CookieDetails.vue";
 
// 响应式数据
const searchQuery = ref("");
const domains = ref<DomainConfig[]>([]);
const cookies = ref<Cookie[]>([]);
const expandedDomains = ref(new Set<string>());
const selectedCookie = ref<Cookie | null>(null);

// 计算属性
const filteredDomains = computed(() => {
  if (!searchQuery.value) return domains.value;
  const lowerCaseQuery = searchQuery.value.toLowerCase();
  return domains.value.filter((d: DomainConfig) => {
    const domainMatch = d.domain.toLowerCase().includes(lowerCaseQuery);
    if (domainMatch) return true;

    const cookieMatch = cookies.value.some(
      (c: Cookie) =>
        c.domain === d.domain && c.name.toLowerCase().includes(lowerCaseQuery)
    );
    return cookieMatch;
  });
});

// 方法
const refreshCookies = async () => {
  try {
    const domainResponse = await sendMessage("getAllDomains");
    if (!domainResponse.success) {
      console.error("加载域名失败:", domainResponse.error);
      return;
    }
    domains.value = domainResponse.data;

    const allCookies: Cookie[] = [];
    for (const domainConfig of domains.value) {
      const cookieResponse = await sendMessage("getDomainCookies", {
        domain: domainConfig.domain,
      });
      if (cookieResponse.success) {
        allCookies.push(...cookieResponse.cookies);
      }
    }
    cookies.value = allCookies;
  } catch (error) {
    console.error("刷新Cookie失败:", error);
  }
};

const toggleDomain = (domain: string) => {
  if (expandedDomains.value.has(domain)) {
    expandedDomains.value.delete(domain);
  } else {
    expandedDomains.value.add(domain);
  }
};

const getDomainCookieCount = (domain: string) => {
  return cookies.value.filter((cookie: Cookie) => cookie.domain === domain).length;
};

const getDomainCookies = (domain: string) => {
  return cookies.value.filter((cookie: Cookie) => cookie.domain === domain);
};

const showCookieDetails = (cookie: Cookie) => {
  selectedCookie.value = cookie;
};

const closeModal = () => {
  selectedCookie.value = null;
};

const truncateValue = (value: string) => {
  return value.length > 20 ? value.substring(0, 20) + "..." : value;
};

const formatExpiration = (expirationDate?: number) => {
  if (!expirationDate) return "会话";
  return new Date(expirationDate * 1000).toLocaleDateString();
};

// 生命周期
onMounted(() => {
  refreshCookies();
});
</script>

<style scoped>
.cookies-page {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-header h1 {
  margin: 0;
  color: #333;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.search-section {
  margin-bottom: 24px;
}

.search-input {
  width: 100%;
  padding: 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
}

.domains-section h2 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 18px;
}

.domain-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.domain-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
}

.domain-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: #f8f9fa;
  cursor: pointer;
  transition: background-color 0.2s;
}

.domain-header:hover {
  background: #e9ecef;
}

.domain-name {
  font-weight: 600;
  color: #333;
}

.cookie-count {
  color: #666;
  font-size: 14px;
}

.expand-icon {
  color: #666;
}

.cookies-list {
  background: white;
}

.cookie-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background-color 0.2s;
}

.cookie-item:hover {
  background: #f8f9fa;
}

.cookie-item:last-child {
  border-bottom: none;
}

.cookie-name {
  font-weight: 500;
  color: #333;
  min-width: 150px;
}

.cookie-value {
  color: #666;
  flex: 1;
  margin: 0 12px;
}

.cookie-expires {
  color: #999;
  font-size: 12px;
  min-width: 80px;
  text-align: right;
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  max-width: 600px;
  width: 90%;
  max-height: 80vh;
  overflow: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #e0e0e0;
}

.modal-header h3 {
  margin: 0;
  color: #333;
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #666;
}

.close-btn:hover {
  color: #333;
}

.modal-body {
  padding: 20px;
}
</style>
