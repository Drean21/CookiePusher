<template>
  <div class="managed-view-container">
    <div v-if="loading" class="loading-state">
      <p>正在加载推送列表...</p>
    </div>
    <div
      v-else-if="Object.keys(managedDomains).length === 0 && !isAdding"
      class="empty-state"
    >
      <p>推送列表为空。</p>
      <p class="tip">请在“当前页面”视图中添加Cookie，或在此处手动添加一个新域。</p>
      <div class="add-domain-form">
        <input
          v-model="newDomainInput"
          @keyup.enter="addNewDomain"
          placeholder="例如: example.com"
        />
        <button @click="addNewDomain" :disabled="isAdding">
          <span v-if="!isAdding">添加域</span>
          <span v-else class="spinner-small"></span>
        </button>
      </div>
    </div>
    <div v-else class="two-column-layout">
      <!-- Left Sidebar -->
      <aside class="sidebar">
        <div class="sidebar-header">
          <p>已推送域名</p>
        </div>
        <div class="add-domain-form">
          <input
            v-model="newDomainInput"
            @keyup.enter="addNewDomain"
            placeholder="添加新域..."
          />
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
            @click="selectDomain(domain)"
            class="domain-item"
          >
            <div class="domain-info">{{ domain }}</div>
            <div class="domain-actions">
              <div class="copy-action">
                <button
                  @click.stop="toggleDomainCopyMenu(domain)"
                  class="action-btn"
                  title="复制"
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
                <div v-if="activeDomainCopyMenu === domain" class="copy-menu">
                  <a
                    @click.stop="copyCookiesForDomain(domain, 'selected')"
                    :class="{ disabled: !hasSelectedCookies(domain) }"
                    >复制选中</a
                  >
                  <a @click.stop="copyCookiesForDomain(domain, 'synced')">复制推送序列</a>
                  <a @click.stop="copyCookiesForDomain(domain, 'all')">复制全部</a>
                </div>
              </div>
              <button
                @click.stop="removeDomain(domain)"
                class="action-btn remove-btn"
                title="移除该域下的所有推送项"
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
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path
                    d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                  ></path>
                </svg>
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
          <div class="controls-container">
            <div class="control-row">
              <div class="search-bar">
                <input v-model="searchQuery" placeholder="筛选Cookie的名称或值..." />
              </div>
              <div class="filter-item">
                <select v-model="selectedSubDomain">
                  <option
                    v-for="option in subDomainFilterOptions"
                    :key="option.value"
                    :value="option.value"
                  >
                    {{ option.text }}
                  </option>
                </select>
              </div>
            </div>
            <div class="control-row">
              <div class="filter-item">
                <label>
                  <input type="checkbox" v-model="showOnlySelected" />
                  只显示选中
                </label>
              </div>
              <div class="filter-item">
                <label>
                  <input type="checkbox" v-model="showOnlyInSync" />
                  只显示已推送
                </label>
              </div>
            </div>
          </div>
          <!-- START: Headers for the cookie list -->
          <div class="cookie-list-header">
            <span class="cookie-info-header">Cookie 信息</span>
            <div class="cookie-actions-header">
              <span title="是否将此Cookie加入推送列表？">推送</span>
              <span title="是否将此Cookie共享到公共池？">共享</span>
            </div>
          </div>
          <!-- END: Headers for the cookie list -->
          <ul class="cookie-list">
            <li
              v-for="cookie in filteredCookies"
              :key="getCookieKey(cookie)"
              class="cookie-item"
              :class="{ selected: isCookieSelected(cookie) }"
              @click="toggleCookieSelection(cookie)"
            >
              <div class="cookie-info">
                <div class="cookie-name-line">
                  <strong class="cookie-name">{{ cookie.name }}</strong>
                  <span class="cookie-domain-badge">{{ cookie.domain }}</span>
                </div>
                <span class="cookie-value-small">{{ cookie.value }}</span>
              </div>
              <div class="cookie-actions">
                <!-- Remark Button -->
                <button
                  @click.stop="editRemark(cookie)"
                  class="action-btn"
                  :title="getRemark(cookie) || '添加/编辑备注'"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 20h9"></path>
                    <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path>
                  </svg>
                </button>
                <div class="copy-action">
                  <button
                    @click.stop="toggleCopyMenu(getCookieKey(cookie))"
                    class="action-btn"
                    title="复制"
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
                  <div v-if="activeCopyMenu === getCookieKey(cookie)" class="copy-menu">
                    <a @click.prevent="copyCookie(cookie, 'value')">复制值</a>
                    <a @click.prevent="copyCookie(cookie, 'name-value')"
                      >复制 Name=Value</a
                    >
                    <a @click.prevent="copyCookie(cookie, 'json')">复制 JSON</a>
                  </div>
                </div>
                <label class="switch" title="推送此Cookie">
                  <input
                    type="checkbox"
                    :checked="isCookieInSyncList(cookie)"
                    @change.stop="toggleSync(cookie)"
                  />
                  <span class="slider round"></span>
                </label>
                <label
                  class="switch"
                  :title="
                    isCookieInSyncList(cookie)
                      ? '共享此Cookie到公共池'
                      : '请先推送此Cookie'
                  "
                >
                  <input
                    type="checkbox"
                    :checked="isCookieSharable(cookie)"
                    :disabled="!isCookieInSyncList(cookie)"
                    @change.stop="toggleShare(cookie)"
                  />
                  <span class="slider round"></span>
                </label>
              </div>
            </li>
          </ul>
          <div v-if="filteredCookies.length === 0" class="empty-state-inner">
            <p>没有匹配的Cookie。</p>
          </div>
        </div>
        <div v-else class="empty-state">
          <p>请在左侧选择一个域名以管理其Cookie推送状态。</p>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted, onUnmounted, ref } from "vue";
import type { Cookie } from "../../../types/extension.d";
import { sendMessage } from "../../utils/message";

type ShowNotification = (
  message: string,
  type?: "success" | "error" | "info",
  duration?: number
) => void;
const showNotification = inject<ShowNotification>("showNotification", () => {});

const loading = ref(true);
const loadingCookies = ref(false);
const isAdding = ref(false);
const managedDomains = ref<{ [key: string]: boolean }>({});
const selectedDomain = ref<string | null>(null);
const cookiesForSelectedDomain = ref<Cookie[]>([]);
const syncListMap = ref<{ [key: string]: Cookie }>({});
const cookieRemarks = ref<{ [key: string]: string }>({});
const newDomainInput = ref("");
const searchQuery = ref("");
const selectedCookies = ref(new Set<string>());
const activeDomainCopyMenu = ref<string | null>(null);
const activeCopyMenu = ref<string | null>(null);
const showOnlySelected = ref(false);
const showOnlyInSync = ref(false);
const selectedSubDomain = ref("all");

const getCookieKey = (cookie: Cookie): string =>
  `${cookie.name}|${cookie.domain}|${cookie.path}`;
const isCookieInSyncList = (cookie: Cookie): boolean =>
  !!syncListMap.value[getCookieKey(cookie)];

const isCookieSharable = (cookie: Cookie): boolean => {
  const key = getCookieKey(cookie);
  const syncedCookie = syncListMap.value[key];
  return syncedCookie ? (syncedCookie as any).isSharable || false : false;
};

const isCookieSelected = (cookie: Cookie): boolean =>
  selectedCookies.value.has(getCookieKey(cookie));

const getRemark = (cookie: Cookie): string => {
  return cookieRemarks.value[getCookieKey(cookie)] || "";
};

const editRemark = async (cookie: Cookie) => {
  const currentRemark = getRemark(cookie);
  const newRemark = window.prompt(`为 "${cookie.name}" 添加/编辑备注:`, currentRemark);

  if (newRemark !== null && newRemark !== currentRemark) {
    try {
      const cookieKey = getCookieKey(cookie);
      const response = await sendMessage("updateCookieRemark", {
        cookieKey,
        remark: newRemark,
      });
      if (response.success) {
        // Optimistically update the local state
        const newRemarks = { ...cookieRemarks.value };
        if (newRemark) {
          newRemarks[cookieKey] = newRemark;
        } else {
          delete newRemarks[cookieKey];
        }
        cookieRemarks.value = newRemarks;
        showNotification("备注已更新。", "success", 2000);
      }
    } catch (e: any) {
      showNotification(`更新备注失败: ${e.message}`, "error");
    }
  }
};

const subDomainFilterOptions = computed(() => {
  const counts: { [key: string]: number } = {};
  for (const cookie of cookiesForSelectedDomain.value) {
    counts[cookie.domain] = (counts[cookie.domain] || 0) + 1;
  }
  const options = Object.entries(counts).map(([domain, count]) => ({
    value: domain,
    text: `${domain} (${count})`,
  }));
  return [
    { value: "all", text: `全部域名 (${cookiesForSelectedDomain.value.length})` },
    ...options,
  ];
});

const filteredCookies = computed(() => {
  let cookies = cookiesForSelectedDomain.value;

  if (selectedSubDomain.value !== "all") {
    cookies = cookies.filter((c) => c.domain === selectedSubDomain.value);
  }
  if (searchQuery.value) {
    const lowerCaseQuery = searchQuery.value.toLowerCase();
    cookies = cookies.filter(
      (cookie) =>
        cookie.name.toLowerCase().includes(lowerCaseQuery) ||
        cookie.value.toLowerCase().includes(lowerCaseQuery)
    );
  }
  if (showOnlySelected.value) {
    cookies = cookies.filter((c) => isCookieSelected(c));
  }
  if (showOnlyInSync.value) {
    cookies = cookies.filter((c) => isCookieInSyncList(c));
  }
  return cookies;
});

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
  if (twoLevelTlds.has(lastTwo) && parts.length > 2) return parts.slice(-3).join(".");
  return lastTwo;
};

const groupAndRenderSyncList = (list: Cookie[]) => {
  const newMap: { [key: string]: Cookie } = {};
  for (const cookie of list) {
    newMap[getCookieKey(cookie)] = cookie;
  }
  syncListMap.value = newMap;
  const domains = list.reduce((acc, cookie) => {
    acc[getRegistrableDomain(cookie.domain)] = true;
    return acc;
  }, {} as { [key: string]: boolean });

  managedDomains.value = Object.keys(domains)
    .sort()
    .reduce((obj, key) => {
      obj[key] = domains[key];
      return obj;
    }, {} as { [key: string]: boolean });

  if (!selectedDomain.value && Object.keys(managedDomains.value).length > 0) {
    selectDomain(Object.keys(managedDomains.value)[0]);
  }
};

const selectDomain = async (domain: string) => {
  selectedDomain.value = domain;
  selectedCookies.value.clear();
  searchQuery.value = "";
  showOnlySelected.value = false;
  selectedSubDomain.value = "all";
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
  const isInSync = isCookieInSyncList(cookie);
  // Optimistically update the UI
  const newSyncMap = { ...syncListMap.value };
  if (isInSync) {
    // If it's in sync, also ensure it's not sharable anymore when removed
    if (newSyncMap[key]) delete newSyncMap[key];
  } else {
    newSyncMap[key] = { ...cookie, isSharable: false }; // Add with default share state
  }
  syncListMap.value = newSyncMap;
  showNotification(
    `已${isInSync ? "从推送列表移除" : "添加"} ${cookie.name}`,
    "success",
    2000
  );
  try {
    // Send the entire updated list to the background script for consistency
    await sendMessage("updateSyncList", { syncList: Object.values(newSyncMap) });
  } catch (e: any) {
    showNotification(`操作失败: ${e.message}`, "error");
    // Revert UI on failure
    const revertedMap = { ...syncListMap.value };
    if (isInSync) {
      // This part is tricky, as the original cookie might not have had isSharable.
      // Re-fetching from storage on error is safer.
      const { syncList: storedSyncList } = await chrome.storage.local.get("syncList");
      if (storedSyncList) groupAndRenderSyncList(storedSyncList);
    } else {
      delete revertedMap[key]; // It was added, so remove it
      syncListMap.value = revertedMap;
    }
  }
};

const toggleShare = (cookie: Cookie) => {
  if (!isCookieInSyncList(cookie)) {
    showNotification("请先推送此Cookie才能开启共享", "info");
    return;
  }
  const key = getCookieKey(cookie);

  // Optimistically update the UI
  const newSyncMap = { ...syncListMap.value };
  const updatedCookie = { ...newSyncMap[key] };
  updatedCookie.isSharable = !updatedCookie.isSharable; // Toggle the state
  newSyncMap[key] = updatedCookie;
  syncListMap.value = newSyncMap; // Trigger reactivity

  showNotification(
    `Cookie "${cookie.name}" 的共享状态已${updatedCookie.isSharable ? "开启" : "关闭"}`,
    "success",
    2000
  );

  // Send the entire updated list to the background script
  sendMessage("updateSyncList", { syncList: Object.values(newSyncMap) }).catch(
    async (e: any) => {
      showNotification(`更新共享状态失败: ${e.message}`, "error");
      // Revert UI on failure by re-fetching from storage
      const { syncList: storedSyncList } = await chrome.storage.local.get("syncList");
      if (storedSyncList) groupAndRenderSyncList(storedSyncList);
    }
  );
};

const addNewDomain = async () => {
  const domainToAdd = newDomainInput.value.trim();
  if (!domainToAdd || isAdding.value) return;
  isAdding.value = true;
  try {
    const response = await sendMessage("getCookiesForDomain", { domain: domainToAdd });
    if (response.success && response.cookies.length > 0) {
      await sendMessage("syncAllCookiesForDomain", { cookies: response.cookies });
      showNotification(`已成功添加域 ${domainToAdd} 并推送其所有Cookie。`, "success");
    } else {
      showNotification(`域 ${domainToAdd} 下未找到任何Cookie。`, "info");
    }
    newDomainInput.value = "";
  } catch (e: any) {
    console.error("Failed to add new domain", e);
    showNotification(`添加域失败: ${e.message}`, "error");
  } finally {
    isAdding.value = false;
    // Manually refetch to show the new domain in the list
    const { syncList: updatedSyncList } = await chrome.storage.local.get("syncList");
    groupAndRenderSyncList(updatedSyncList || []);
  }
};

const toggleCookieSelection = (cookie: Cookie) => {
  const key = getCookieKey(cookie);
  if (selectedCookies.value.has(key)) {
    selectedCookies.value.delete(key);
  } else {
    selectedCookies.value.add(key);
  }
};

const hasSelectedCookies = (domain: string): boolean => {
  return cookiesForSelectedDomain.value.some(
    (c) => getRegistrableDomain(c.domain) === domain && isCookieSelected(c)
  );
};

const toggleDomainCopyMenu = (domain: string) => {
  activeDomainCopyMenu.value = activeDomainCopyMenu.value === domain ? null : domain;
};

const copyCookiesForDomain = async (
  domain: string,
  type: "selected" | "all" | "synced"
) => {
  activeDomainCopyMenu.value = null;
  let cookiesToCopy: Cookie[] = [];

  if (type === "all") {
    const response = await sendMessage("getCookiesForDomain", { domain });
    if (response.success) cookiesToCopy = response.cookies;
  } else if (type === "selected") {
    cookiesToCopy = cookiesForSelectedDomain.value.filter((c) => isCookieSelected(c));
  } else if (type === "synced") {
    const allSynced = Object.values(syncListMap.value);
    cookiesToCopy = allSynced.filter((c) => getRegistrableDomain(c.domain) === domain);
  }

  if (cookiesToCopy.length > 0) {
    const textToCopy = cookiesToCopy
      .map((c: Cookie) => `${c.name}=${c.value}`)
      .join("; ");
    await navigator.clipboard.writeText(textToCopy);
    showNotification(`已复制 ${cookiesToCopy.length} 个Cookie`, "success");
  } else {
    showNotification("没有可复制的Cookie", "info");
  }
};

const removeDomain = async (domain: string) => {
  if (window.confirm(`确定要从推送列表中移除域名 "${domain}" 吗？`)) {
    try {
      const response = await sendMessage("removeDomainFromSyncList", { domain });
      showNotification(`已移除域名 ${domain}`, "success");

      if (selectedDomain.value === domain) {
        selectedDomain.value = null;
      }
      // Use the authoritative list returned from the background script
      groupAndRenderSyncList(response.syncList || []);
    } catch (e: any) {
      showNotification(`移除失败: ${e.message}`, "error");
    }
  }
};

const copyCookie = async (cookie: Cookie, format: "value" | "name-value" | "json") => {
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
  showNotification("已复制!", "success", 1500);
};

const toggleCopyMenu = (cookieKey: string) => {
  activeCopyMenu.value = activeCopyMenu.value === cookieKey ? null : cookieKey;
};

onMounted(async () => {
  loading.value = true;
  const { syncList: storedSyncList, cookieRemarks: storedRemarks } =
    await chrome.storage.local.get(["syncList", "cookieRemarks"]);
  if (storedSyncList) groupAndRenderSyncList(storedSyncList);
  if (storedRemarks) cookieRemarks.value = storedRemarks;
  loading.value = false;
  // chrome.storage.onChanged.addListener(handleStorageChange);
});

onUnmounted(() => {
  // chrome.storage.onChanged.removeListener(handleStorageChange);
});
</script>

<style scoped>
.managed-view-container,
.two-column-layout {
  width: 100%;
  height: 100%;
  display: flex;
}
.sidebar {
  width: 200px;
  height: 100%;
  border-right: 1px solid #e0e0e0;
  background-color: #fafafa;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
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
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow-y: auto;
  padding: 12px;
}
.controls-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}
.control-row {
  display: flex;
  gap: 12px;
  align-items: center;
}
.search-bar {
  flex-grow: 1;
}
.search-bar input {
  width: 100%;
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid #ccc;
  font-size: 14px;
  box-sizing: border-box;
}
.filter-item {
  font-size: 13px;
}
.filter-item select {
  border: 1px solid #ccc;
  border-radius: 4px;
  padding: 4px 6px;
}
.filter-item label {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}
.cookie-list {
  list-style: none;
  padding: 0;
  margin: 0;
  overflow-y: auto;
  flex-grow: 1;
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
  cursor: pointer;
  transition: background-color 0.2s;
}
.cookie-item:hover {
  background-color: #f9f9f9;
}
.cookie-item.selected {
  background-color: #e0e7ff;
  border-color: #c7d2fe;
}
.cookie-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: hidden;
  flex-grow: 1;
  margin-right: 10px;
}
.cookie-name-line {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.cookie-name {
  font-weight: 600;
  font-size: 14px;
  color: #333;
}
.cookie-domain-badge {
  font-size: 11px;
  background-color: #eee;
  color: #555;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  white-space: nowrap;
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
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
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
.copy-menu a.disabled {
  color: #ccc;
  cursor: not-allowed;
  background-color: transparent;
}
.loading-state,
.empty-state,
.loading-state-inner,
.empty-state-inner {
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
  flex-grow: 1;
  justify-content: center;
}
.tip {
  font-size: 12px;
  color: #9e9e9e;
  max-width: 300px;
  margin-top: 10px;
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
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
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
  min-width: 0;
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
.switch {
  position: relative;
  display: inline-block;
  width: 34px;
  height: 20px;
}
.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}
.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
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
  transition: 0.4s;
}
input:checked + .slider {
  background-color: #667eea;
}
input:focus + .slider {
  box-shadow: 0 0 1px #667eea;
}
input:checked + .slider:before {
  transform: translateX(14px);
}
.slider.round {
  border-radius: 20px;
}
.slider.round:before {
  border-radius: 50%;
}
.cookie-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 10px;
  margin-bottom: 8px;
  font-size: 13px;
  font-weight: 600;
  color: #333;
}
.cookie-info-header {
  flex-grow: 1;
  margin-right: 10px; /* Match .cookie-info margin */
}
.cookie-actions-header {
  display: flex;
  align-items: center;
  gap: 12px; /* Match .cookie-actions gap */
  flex-shrink: 0;
  /* The space for the copy button (approx 22px) + gap (12px) */
  padding-left: 34px;
}
.cookie-actions-header span {
  width: 34px; /* Match .switch width */
  text-align: center;
}
</style>
