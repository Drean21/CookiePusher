<template>
  <div class="settings-container">
    <header class="settings-header">
      <button
        v-if="activeSubView !== 'main'"
        @click="activeSubView = 'main'"
        class="back-btn"
      >
        <
      </button>
      <h1>{{ headerTitle }}</h1>
    </header>
    <main class="settings-content">
      <div v-if="activeSubView === 'main'">
        <ul class="settings-list">
          <li @click="activeSubView = 'sync'">
            <span>云端同步</span>
            <span class="arrow">></span>
          </li>
          <li @click="activeSubView = 'keep-alive'">
            <span>Cookie保活</span>
            <span class="arrow">></span>
          </li>
          <li @click="activeSubView = 'logs'">
            <span>操作日志</span>
            <span class="arrow">></span>
          </li>
        </ul>
      </div>

      <div v-if="activeSubView === 'sync'" class="sub-view">
        <div class="setting-item">
          <label for="api-endpoint">API 端点</label>
          <input
            id="api-endpoint"
            type="text"
            v-model="syncSettings.apiEndpoint"
            placeholder="例如: http://localhost:8080"
          />
        </div>
        <div class="setting-item">
          <label for="auth-token">Auth Token</label>
          <input
            id="auth-token"
            type="password"
            v-model="syncSettings.authToken"
            placeholder="输入您的Bearer Token"
          />
        </div>
        <div class="setting-actions">
          <button @click="testConnection" :disabled="isTesting" class="action-btn">
            <span v-if="!isTesting">测试连接</span>
            <span v-else class="spinner-small"></span>
          </button>
          <button @click="saveSettings" :disabled="isSaving" class="action-btn primary">
            <span v-if="!isSaving">保存设置</span>
            <span v-else class="spinner-small"></span>
          </button>
        </div>
        <div class="setting-category">
          <h3>手动操作</h3>
          <div class="setting-actions">
            <button @click="manualSync" :disabled="isSyncing" class="action-btn">
              <span v-if="!isSyncing">立即同步</span>
              <span v-else class="spinner-small"></span>
            </button>
          </div>
        </div>
      </div>

      <div v-if="activeSubView === 'keep-alive'" class="sub-view">
        <div class="setting-item">
          <label for="keep-alive-frequency">保活任务频率</label>
          <select
            id="keep-alive-frequency"
            v-model.number="syncSettings.keepAliveFrequency"
          >
            <option :value="1">每分钟 (仅用于测试)</option>
            <option :value="60">每小时</option>
            <option :value="180">每3小时</option>
            <option :value="720">每12小时</option>
            <option :value="1440">每天</option>
          </select>
          <p class="tip">设置后台静默访问网站以刷新Cookie有效期的频率。</p>
        </div>
        <div class="setting-actions">
          <button @click="saveSettings" :disabled="isSaving" class="action-btn primary">
            <span v-if="!isSaving">保存</span>
            <span v-else class="spinner-small"></span>
          </button>
        </div>
      </div>

      <LogView v-if="activeSubView === 'logs'" />
    </main>
  </div>
</template>

<script setup lang="ts">
import CryptoJS from "crypto-js";
import { computed, inject, onMounted, ref } from "vue";
import { sendMessage } from "../../utils/message";
import LogView from "./LogView.vue";

type ShowNotification = (
  message: string,
  type?: "success" | "error" | "info",
  duration?: number
) => void;
const showNotification = inject<ShowNotification>("showNotification", () => {});

type SubView = "main" | "sync" | "logs" | "keep-alive";

const activeSubView = ref<SubView>("main");

const headerTitle = computed(() => {
  switch (activeSubView.value) {
    case "main":
      return "设置";
    case "sync":
      return "云端同步";
    case "logs":
      return "操作日志";
    case "keep-alive":
      return "Cookie保活";
    default:
      return "设置";
  }
});

const syncSettings = ref({
  apiEndpoint: "",
  authToken: "",
  keepAliveFrequency: 1, // Default to 1 minute for testing
});

const isTesting = ref(false);
const isSaving = ref(false);
const isSyncing = ref(false);

const SECRET_KEY = "cookie-syncer-secret-key";

const saveSettings = async () => {
  isSaving.value = true;
  try {
    const dataToStore = {
      apiEndpoint: syncSettings.value.apiEndpoint,
      authToken: CryptoJS.AES.encrypt(
        syncSettings.value.authToken,
        SECRET_KEY
      ).toString(),
      keepAliveFrequency: syncSettings.value.keepAliveFrequency,
    };
    await chrome.storage.local.set({ syncSettings: dataToStore });
    showNotification("设置已保存！", "success");
  } catch (e: any) {
    showNotification(`保存失败: ${e.message}`, "error");
  } finally {
    isSaving.value = false;
  }
};

const testConnection = async () => {
  isTesting.value = true;
  await saveSettings();
  try {
    const response = await sendMessage("testApiConnection");
    if (response.success) {
      showNotification("连接成功！", "success");
    } else {
      throw new Error(response.error || "未知错误");
    }
  } catch (e: any) {
    showNotification(`连接失败: ${e.message}`, "error");
  } finally {
    isTesting.value = false;
  }
};

const manualSync = async () => {
  isSyncing.value = true;
  try {
    const response = await sendMessage("manualSync");
    if (response.success) {
      showNotification(response.message || "同步成功！", "success");
    } else {
      throw new Error(response.error || "未知同步错误");
    }
  } catch (e: any) {
    showNotification(`同步失败: ${e.message}`, "error");
  } finally {
    isSyncing.value = false;
  }
};

onMounted(async () => {
  const { syncSettings: storedSettings } = await chrome.storage.local.get("syncSettings");
  if (storedSettings) {
    syncSettings.value.apiEndpoint = storedSettings.apiEndpoint || "";
    syncSettings.value.keepAliveFrequency = storedSettings.keepAliveFrequency || 1;
    if (storedSettings.authToken) {
      try {
        const bytes = CryptoJS.AES.decrypt(storedSettings.authToken, SECRET_KEY);
        const decryptedToken = bytes.toString(CryptoJS.enc.Utf8);
        if (decryptedToken) {
          syncSettings.value.authToken = decryptedToken;
        }
      } catch (e) {
        console.error("Failed to decrypt auth token", e);
        showNotification("无法解密Auth Token，请重新输入。", "error");
      }
    }
  }
});
</script>

<style scoped>
.settings-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #f0f2f5;
}
.settings-header {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
  background-color: #1e88e5;
  color: white;
  position: relative;
  flex-shrink: 0;
}
.settings-header h1 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
}
.back-btn {
  position: absolute;
  left: 12px;
  background: none;
  border: none;
  color: white;
  font-size: 24px;
  cursor: pointer;
  padding: 0 10px;
}
.settings-content {
  flex-grow: 1;
  overflow-y: auto;
}
.settings-list {
  list-style: none;
  padding: 0;
  margin: 0;
  background-color: white;
}
.settings-list li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e0e0e0;
  cursor: pointer;
  transition: background-color 0.2s;
}
.settings-list li:hover {
  background-color: #f5f5f5;
}
.arrow {
  color: #ccc;
  font-weight: bold;
}
.sub-view {
  padding: 16px;
}
.setting-category {
  margin-top: 12px;
  background-color: white;
  padding: 16px;
}
.setting-category:first-of-type {
  margin-top: 0;
}
.setting-category h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: #555;
}
.setting-item {
  display: flex;
  flex-direction: column;
  margin-bottom: 16px;
}
.setting-item label {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 6px;
}
.setting-item input,
.setting-item select {
  width: 100%;
  box-sizing: border-box;
  padding: 8px 12px;
  border: 1px solid #ccc;
  border-radius: 4px;
  font-size: 14px;
}
.setting-item .tip {
  font-size: 12px;
  color: #9e9e9e;
  margin-top: 8px;
}
.setting-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
}
.action-btn {
  border: 1px solid #ccc;
  background-color: #f9f9f9;
  color: #333;
  font-size: 14px;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 90px;
}
.action-btn.primary {
  border-color: #667eea;
  background-color: #667eea;
  color: white;
}
.action-btn:disabled {
  background-color: #ccc;
  border-color: #ccc;
  cursor: not-allowed;
}
.spinner-small {
  display: inline-block;
  border: 2px solid rgba(255, 255, 255, 0.6);
  border-top: 2px solid #fff;
  border-radius: 50%;
  width: 14px;
  height: 14px;
  animation: spin 1s linear infinite;
}
.action-btn:not(.primary) .spinner-small {
  border-top-color: #333;
  border-left-color: #333;
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
