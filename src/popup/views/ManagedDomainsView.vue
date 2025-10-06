<template>
  <div class="managed-view-container">
    <div v-if="loading" class="loading-state">
      <p>正在加载同步列表...</p>
    </div>
    <div v-else-if="Object.keys(managedDomains).length === 0 && !isAdding" class="empty-state">
      <p>同步列表为空。</p>
      <p class="tip">
        请在“当前页面”视图中添加Cookie，或在此处手动添加一个新域。
      </p>
       <div class="add-domain-form">
        <input v-model="newDomainInput" @keyup.enter="addNewDomain" placeholder="例如: example.com" />
        <button @click="addNewDomain">添加域</button>
      </div>
    </div>
    <div v-else class="two-column-layout">
      <!-- Left Sidebar -->
      <aside class="sidebar">
        <div class="sidebar-header">
          <p>已同步域名</p>
        </div>
        <div class="add-domain-form">
          <input v-model="newDomainInput" @keyup.enter="addNewDomain" placeholder="添加新域..." />
          <button @click="addNewDomain" :disabled="isAdding">
            <span v-if="!isAdding">添加</span>
            <span v-else class="spinner-small"></span>
          </button>
        </div>
        <ul>
          <li
            v-for="domain in Object.keys(managedDomains)"
            :key="domain"
            :class="{ active: domain === selectedDomain }"
            class="domain-item"
             @click="selectDomain(domain)"
          >
            <div class="domain-info">
                {{ domain }}
            </div>
            <div class="domain-actions">
                 <button
                    @click.stop="copyAllCookiesForDomain(domain)"
                    class="action-btn"
                    title="复制该域下的所有Cookie (HTTP Header格式)"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                </button>
                 <button @click.stop="removeDomain(domain)" class="action-btn remove-btn" title="移除该域下的所有同步项">
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                 </button>
            </div>
          </li>
        </ul>
      </aside>
      <!-- Right Main Content -->
      <section class="main-content">
        <div v-if="loadingCookies" class="loading-state-inner">
          <div class="spinner"></div>
          <p>正在获取实时Cookie...</p>
        </div>
        <div v-else-if="selectedDomain" class="cookie-group">
          <div class="search-bar">
            <input v-model="searchQuery" placeholder="筛选Cookie的名称或值..." />
          </div>
          <ul class="cookie-list">
            <li
              v-for="cookie in filteredCookies"
              :key="getCookieKey(cookie)"
              class="cookie-item"
            >
              <div class="cookie-info">
                <strong class="cookie-name">{{ cookie.name }}</strong>
                <span class="cookie-value-small">{{ cookie.value }}</span>
              </div>
              <div class="cookie-actions">
                 <div class="copy-action">
                    <button @click="toggleCopyMenu(getCookieKey(cookie))" class="action-btn" title="复制">
                       <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                    </button>
                    <div v-if="activeCopyMenu === getCookieKey(cookie)" class="copy-menu">
                        <a @click.prevent="copyCookie(cookie, 'value')">复制值</a>
                        <a @click.prevent="copyCookie(cookie, 'name-value')">复制 Name=Value</a>
                        <a @click.prevent="copyCookie(cookie, 'json')">复制 JSON</a>
                    </div>
                </div>
                <label class="switch">
                  <input
                    type="checkbox"
                    :checked="isCookieInSyncList(cookie)"
                    @change="toggleSync(cookie)"
                  />
                  <span class="slider round"></span>
                </label>
              </div>
            </li>
          </ul>
        </div>
        <div v-else class="empty-state">
          <p>请在左侧选择一个域名以管理其Cookie同步状态。</p>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted, inject } from "vue";
import type { Cookie } from "../../../types/extension.d";
import { sendMessage } from "../../utils/message";

type ShowNotification = (message: string, type?: 'success' | 'error', duration?: number) => void;
const showNotification = inject<ShowNotification>('showNotification', () => {});

const loading = ref(true);
const loadingCookies = ref(false);
const isAdding = ref(false);
const syncList = ref<Cookie[]>([]);
const managedDomains = ref<{ [key: string]: boolean }>({});
const selectedDomain = ref<string | null>(null);
const cookiesForSelectedDomain = ref<Cookie[]>([]);
const syncListMap = ref<Map<string, Cookie>>(new Map());
const newDomainInput = ref("");
const searchQuery = ref("");
const activeCopyMenu = ref<string | null>(null);


const getCookieKey = (cookie: Cookie): string => `${cookie.name}|${cookie.domain}|${cookie.path}`;

const isCookieInSyncList = (cookie: Cookie): boolean => syncListMap.value.has(getCookieKey(cookie));

const filteredCookies = computed(() => {
    if (!searchQuery.value) {
        return cookiesForSelectedDomain.value;
    }
    const lowerCaseQuery = searchQuery.value.toLowerCase();
    return cookiesForSelectedDomain.value.filter(cookie => 
        cookie.name.toLowerCase().includes(lowerCaseQuery) || 
        cookie.value.toLowerCase().includes(lowerCaseQuery)
    );
});

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

const groupAndRenderSyncList = (list: Cookie[]) => {
  syncList.value = list;
  syncListMap.value = new Map(list.map((c) => [getCookieKey(c), c]));

  const domains = list.reduce((acc, cookie) => {
    const regDomain = getRegistrableDomain(cookie.domain);
    acc[regDomain] = true;
    return acc;
  }, {} as { [key: string]: boolean });
  
  managedDomains.value = Object.keys(domains).sort().reduce((obj, key) => { 
      obj[key] = domains[key]; 
      return obj;
  }, {} as { [key: string]: boolean });

  if (!selectedDomain.value && Object.keys(managedDomains.value).length > 0) {
    selectDomain(Object.keys(managedDomains.value)[0]);
  }
};

const selectDomain = async (domain: string) => {
  selectedDomain.value = domain;
  loadingCookies.value = true;
  cookiesForSelectedDomain.value = [];
  try {
    const response = await sendMessage("getCookiesForDomain", { domain });
    if (response.success) {
      cookiesForSelectedDomain.value = response.cookies.sort((a: Cookie, b: Cookie) =>
        a.name.localeCompare(b.name)
      );
    }
  } catch (e) {
    console.error(`Failed to get cookies for domain ${domain}`, e);
  } finally {
    loadingCookies.value = false;
  }
};

const toggleSync = async (cookie: Cookie) => {
  const key = getCookieKey(cookie);
  try {
    if (syncListMap.value.has(key)) {
        await sendMessage("removeCookieFromSyncList", { cookie });
        showNotification(`已从同步列表移除 ${cookie.name}`, 'success', 2000);
    } else {
        await sendMessage("syncSingleCookie", { cookie });
        showNotification(`已添加 ${cookie.name} 到同步列表`, 'success', 2000);
    }
  } catch(e: any) {
    showNotification(`操作失败: ${e.message}`, 'error');
  }
};

const addNewDomain = async () => {
    const domainToAdd = newDomainInput.value.trim();
    if(!domainToAdd || isAdding.value) return;

    isAdding.value = true;
    try {
        const response = await sendMessage("getCookiesForDomain", { domain: domainToAdd });
        if(response.success && response.cookies.length > 0) {
            await sendMessage("syncAllCookiesForDomain", { cookies: response.cookies });
        }
        newDomainInput.value = "";
        // The storage listener will update the domain list and select the new one.
    } catch(e: any) {
        console.error("Failed to add new domain", e);
        showNotification(`添加域失败: ${e.message}`, 'error');
    } finally {
        isAdding.value = false;
    }
};

const copyCookie = async (cookie: Cookie, format: "value" | "name-value" | "json") => {
  let textToCopy = "";
  switch (format) {
    case "value": textToCopy = cookie.value; break;
    case "name-value": textToCopy = `${cookie.name}=${cookie.value}`; break;
    case "json": textToCopy = JSON.stringify(cookie, null, 2); break;
  }
  await navigator.clipboard.writeText(textToCopy);
  activeCopyMenu.value = null;
};

const toggleCopyMenu = (cookieKey: string) => {
    activeCopyMenu.value = activeCopyMenu.value === cookieKey ? null : cookieKey;
};

const copyAllCookiesForDomain = async (domain: string) => {
    const response = await sendMessage("getCookiesForDomain", { domain });
    if (response.success && response.cookies.length > 0) {
        const textToCopy = response.cookies.map((c: Cookie) => `${c.name}=${c.value}`).join('; ');
        await navigator.clipboard.writeText(textToCopy);
    }
};

const removeDomain = async (domain: string) => {
    // We can't get the cookie count from here easily, so we just show a generic confirm.
    if (window.confirm(`确定要从同步列表中移除域名 "${domain}" 吗？`)) {
        try {
            const response = await sendMessage('removeDomainFromSyncList', { domain });
            if (response.success) {
                showNotification(`已移除域名 ${domain}`, 'success');
                if(selectedDomain.value === domain){
                    selectedDomain.value = null;
                }
            }
        } catch(e: any) {
            showNotification(`移除失败: ${e.message}`, 'error');
        }
    }
};

const handleStorageChange = (changes: { [key: string]: chrome.storage.StorageChange }, area: string) => {
    if (area === "local" && changes.syncList) {
      const newList = changes.syncList.newValue || [];
      const oldSelected = selectedDomain.value;
      groupAndRenderSyncList(newList);
      // If a new domain was added, select it
      if(oldSelected && !managedDomains.value[oldSelected]){
          const newDomains = Object.keys(managedDomains.value);
          if(newDomains.length > 0) selectDomain(newDomains[0]);
      }
    }
};

onMounted(async () => {
  loading.value = true;
  const { syncList: storedSyncList } = await chrome.storage.local.get("syncList");
  if (storedSyncList) {
    groupAndRenderSyncList(storedSyncList);
  }
  loading.value = false;
  chrome.storage.onChanged.addListener(handleStorageChange);
});

onUnmounted(() => {
    chrome.storage.onChanged.removeListener(handleStorageChange);
});

</script>

<style scoped>
/* Using a similar layout to CurrentPageView */
.managed-view-container {
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
  display: flex;
  flex-direction: column;
}
.sidebar-header {
  padding: 12px;
  border-bottom: 1px solid #e0e0e0;
  text-align: center;
  font-weight: 600;
  color: #424242;
  flex-shrink: 0;
}
.sidebar ul {
  list-style: none;
  padding: 8px;
  margin: 0;
  overflow-y: auto;
  flex-grow: 1;
}
.sidebar li.domain-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  cursor: pointer;
  border-radius: 4px;
  margin-bottom: 4px;
  font-size: 14px;
}
.domain-info {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex-grow: 1;
}
.domain-actions {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
    margin-left: 8px;
}
.remove-btn:hover {
    color: #f44336;
}
.sidebar li:hover {
  background-color: #f0f0f0;
}
.sidebar li.active {
  background-color: #e0e7ff;
  color: #4f46e5;
  font-weight: 600;
}
.main-content {
  flex: 1;
  height: 100%;
  overflow-y: auto;
  padding: 12px;
}
.search-bar {
  margin-bottom: 12px;
}
.search-bar input {
  width: 100%;
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid #ccc;
  font-size: 14px;
}
.cookie-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.cookie-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px;
  border-radius: 6px;
  background-color: #fff;
  margin-bottom: 8px;
  border: 1px solid #e0e0e0;
}
.cookie-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: hidden;
  flex-grow: 1;
  margin-right: 10px;
}
.cookie-name {
  font-weight: 600;
  font-size: 14px;
  color: #333;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cookie-value-small {
  font-size: 12px;
  color: #555;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cookie-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;
}
.action-btn {
  background: none;
  border: none;
  padding: 4px;
  cursor: pointer;
  color: #555;
}
.action-btn:hover {
    color: #000;
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
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  z-index: 10;
  width: 150px;
  padding: 6px;
}
.copy-menu a {
    display: block;
    padding: 8px 12px;
    font-size: 13px;
    color: #333;
    border-radius: 4px;
    text-decoration: none;
}
.copy-menu a:hover {
    background-color: #f0f2f5;
}
.loading-state,
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
.loading-state-inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 10px;
  color: #757575;
}
.tip {
  font-size: 12px;
  color: #9e9e9e;
  max-width: 300px;
}
.spinner {
  border: 4px solid #f3f3f3;
  border-top: 4px solid #667eea;
  border-radius: 50%;
  width: 30px;
  height: 30px;
  animation: spin 1s linear infinite;
}
.spinner-small {
    display: inline-block;
    border: 2px solid #f3f3f3;
    border-top: 2px solid #fff;
    border-radius: 50%;
    width: 12px;
    height: 12px;
    animation: spin 1s linear infinite;
}
@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.add-domain-form {
    padding: 8px;
    border-bottom: 1px solid #e0e0e0;
    display: flex;
    gap: 4px;
}
.add-domain-form input {
    flex-grow: 1;
    border: 1px solid #ccc;
    border-radius: 4px;
    padding: 6px 8px;
    font-size: 13px;
    min-width: 0; /* Prevents overflow */
}
.add-domain-form button {
    border: 1px solid #667eea;
    background-color: #667eea;
    color: white;
    font-size: 13px;
    padding: 6px 10px;
    border-radius: 4px;
    cursor: pointer;
    flex-shrink: 0;
}
.add-domain-form button:disabled {
    background-color: #ccc;
    border-color: #ccc;
    cursor: not-allowed;
}

/* The switch - the box around the slider */
.switch {
  position: relative;
  display: inline-block;
  width: 34px;
  height: 20px;
}

/* Hide default HTML checkbox */
.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

/* The slider */
.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
  -webkit-transition: 0.4s;
  transition: 0.4s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 12px;
  width: 12px;
  left: 4px;
  bottom: 4px;
  background-color: white;
  -webkit-transition: 0.4s;
  transition: 0.4s;
}

input:checked + .slider {
  background-color: #667eea;
}

input:focus + .slider {
  box-shadow: 0 0 1px #667eea;
}

input:checked + .slider:before {
  -webkit-transform: translateX(14px);
  -ms-transform: translateX(14px);
  transform: translateX(14px);
}

/* Rounded sliders */
.slider.round {
  border-radius: 20px;
}

.slider.round:before {
  border-radius: 50%;
}
</style>
