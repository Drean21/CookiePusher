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
            <span>云端推送</span>
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
           <li @click="activeSubView = 'sharing'">
            <span>共享池</span>
            <span class="arrow">></span>
          </li>
          <li @click="activeSubView = 'data'">
            <span>数据管理</span>
            <span class="arrow">></span>
          </li>
        </ul>
      </div>

      <div v-if="activeSubView === 'data'" class="sub-view">
        <div class="setting-category">
          <h3>备份与恢复</h3>
          <p class="tip">
            将您的所有设置、推送列表和统计数据导出为JSON文件进行备份，或从备份文件中恢复。
          </p>
          <div class="setting-actions">
            <button @click="exportData" class="action-btn">导出数据</button>
            <button @click="importData" class="action-btn primary">导入数据</button>
          </div>
        </div>
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
              <span v-if="!isSyncing">立即推送</span>
              <span v-else class="spinner-small"></span>
            </button>
          </div>
        </div>
      </div>

      <div v-if="activeSubView === 'sharing'" class="sub-view">
        <div class="setting-item">
          <div class="label-with-switch">
            <label for="sharing-enabled">启用Cookie共享池 <span v-if="!isOnline" class="offline-indicator">（已离线）</span></label>
            <div class="toggle-switch">
              <label class="switch">
                <input type="checkbox" id="sharing-enabled" v-model="sharingEnabled" @change="updateSharingSettings">
                <span class="slider round"></span>
              </label>
            </div>
          </div>
          <p class="tip">
            <strong>请仔细阅读：</strong>启用此功能，即表示您同意将标记为“可共享”的Cookie上传至一个公共池中。这些Cookie可能会被本服务的其他用户用于身份验证或执行操作。虽然我们采取措施保护您的数据，但服务提供方不对因共享Cookie而导致的任何账户安全问题、数据泄露或潜在损失承担责任。如果您不希望承担此风险，请勿开启此功能。
          </p>
        </div>
      </div>

      <div v-if="activeSubView === 'keep-alive'" class="sub-view">
        <div class="setting-item">
          <label for="keep-alive-frequency">保活任务频率 (分钟)</label>
          <input
            id="keep-alive-frequency"
            type="number"
            v-model.number="syncSettings.keepAliveFrequency"
            min="1"
            placeholder="输入分钟数，例如 60"
          />
          <p class="tip">设置后台静默访问网站以刷新Cookie有效期的频率。最小值为 1 分钟。</p>
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

type SubView = "main" | "sync" | "logs" | "keep-alive" | "data" | "sharing";

const activeSubView = ref<SubView>("main");

const headerTitle = computed(() => {
  switch (activeSubView.value) {
    case "main":
      return "设置";
    case "sync":
      return "云端推送";
    case "logs":
      return "操作日志";
    case "keep-alive":
      return "Cookie保活";
    case "data":
      return "数据管理";
    case "sharing":
        return "共享池设置";
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
const sharingEnabled = ref(false);
const isOnline = ref(true);

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
      showNotification(response.message || "推送成功！", "success");
    } else {
      throw new Error(response.error || "未知推送错误");
    }
  } catch (e: any) {
    showNotification(`推送失败: ${e.message}`, "error");
  } finally {
    isSyncing.value = false;
  }
};

const exportData = async () => {
  try {
    const response = await sendMessage("exportAllData");
    if (!response.success) throw new Error(response.error);

    const dataStr = JSON.stringify(response.data, null, 2);
    const blob = new Blob([dataStr], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    const now = new Date();
    const dateStr = `${now.getFullYear()}${(now.getMonth() + 1).toString().padStart(2, '0')}${now.getDate().toString().padStart(2, '0')}`;
    a.download = `cookiesyncer-backup-${dateStr}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    showNotification("数据已成功导出！", "success");
  } catch (e: any) {
    showNotification(`导出失败: ${e.message}`, "error");
  }
};

const importData = () => {
  const input = document.createElement("input");
  input.type = "file";
  input.accept = "application/json";
  input.onchange = async (e) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = async (event) => {
      try {
        const data = JSON.parse(event.target?.result as string);
        const response = await sendMessage("importAllData", { data });
        if (response.success) {
          showNotification("数据导入成功！请重新加载插件。", "success");
          // Optionally, re-initialize the settings view after import
          await loadInitialSettings();
        } else {
          throw new Error(response.error);
        }
      } catch (err: any) {
        showNotification(`导入失败: ${err.message}`, "error");
      }
    };
    reader.readAsText(file);
  };
  input.click();
};

const updateSharingSettings = async () => {
    try {
        await sendMessage("updateUserSettings", { sharing_enabled: sharingEnabled.value });
        await chrome.storage.local.set({ sharingSettings: { enabled: sharingEnabled.value } });
        isOnline.value = true;
        showNotification("共享设置已更新！", "success");
    } catch (e: any) {
        showNotification(`更新失败: ${e.message}`, "error");
        // Revert the toggle on failure
        sharingEnabled.value = !sharingEnabled.value;
    }
};

const loadInitialSettings = async () => {
  // Load sync settings
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

  // Load sharing settings from the backend
  try {
    const response = await sendMessage("getUserSettings");
    if (response.success && typeof response.data.sharing_enabled === 'boolean') {
      sharingEnabled.value = response.data.sharing_enabled;
      await chrome.storage.local.set({ sharingSettings: { enabled: sharingEnabled.value } });
      isOnline.value = true;
    } else {
        throw new Error(response.error || "Invalid data from API");
    }
  } catch(e: any) {
      console.warn("Could not fetch initial sharing settings from API:", e.message);
      const { sharingSettings } = await chrome.storage.local.get("sharingSettings");
      if (sharingSettings && typeof sharingSettings.enabled === 'boolean') {
          sharingEnabled.value = sharingSettings.enabled;
      }
      isOnline.value = false;
  }
};

onMounted(loadInitialSettings);
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
.offline-indicator {
    color: #f44336; /* Red color for offline status */
    font-size: 12px;
    font-weight: normal;
}
.label-with-switch {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
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
  transition: .4s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 12px;
  width: 12px;
  left: 4px;
  bottom: 4px;
  background-color: white;
  transition: .4s;
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
</style>
