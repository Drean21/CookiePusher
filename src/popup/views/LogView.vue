<template>
  <div class="log-view">
    <div class="log-header">
      <button @click="fetchLogs" title="刷新日志">
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
          <path d="M3 2v6h6" />
          <path d="M21 12A9 9 0 0 0 6 5.3L3 8" />
        </svg>
      </button>
      <button @click="clearLogs" :disabled="logs.length === 0" title="清空日志">
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
          <polyline points="3 6 5 6 21 6"></polyline>
          <path
            d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
          ></path>
        </svg>
      </button>
    </div>
    <ul class="log-list">
      <li v-if="loading" class="log-item empty">正在加载日志...</li>
      <li v-else-if="logs.length === 0" class="log-item empty">暂无日志记录。</li>
      <li
        v-for="(log, index) in logs"
        :key="index"
        class="log-item"
        :class="`log-${log.type}`"
      >
        <span class="log-timestamp">{{ formatTimestamp(log.timestamp) }}</span>
        <span class="log-message">{{ log.message }}</span>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { inject, onMounted, ref } from "vue";
import { sendMessage } from "../../utils/message";

interface LogEntry {
  message: string;
  type: "info" | "success" | "error";
  timestamp: string;
}

type ShowNotification = (
  message: string,
  type?: "success" | "error",
  duration?: number
) => void;
const showNotification = inject<ShowNotification>("showNotification", () => {});

const logs = ref<LogEntry[]>([]);
const loading = ref(true);

const fetchLogs = async () => {
  loading.value = true;
  try {
    const response = await sendMessage("getLogs");
    if (response.success) {
      logs.value = response.logs;
    }
  } catch (e: any) {
    showNotification(`加载日志失败: ${e.message}`, "error");
  } finally {
    loading.value = false;
  }
};

const clearLogs = async () => {
  if (window.confirm("确定要清空所有日志记录吗？")) {
    try {
      await sendMessage("clearLogs");
      logs.value = []; // Optimistic update
      showNotification("日志已清空", "success");
    } catch (e: any) {
      showNotification(`清空失败: ${e.message}`, "error");
    }
  }
};

const formatTimestamp = (timestamp: string) => {
  return new Date(timestamp).toLocaleString();
};

onMounted(fetchLogs);
</script>

<style scoped>
.log-view {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.log-header {
  padding: 12px;
  border-bottom: 1px solid #e0e0e0;
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
  background-color: white;
}
.log-header button {
  border: 1px solid #ccc;
  background-color: #f9f9f9;
  color: #333;
  font-size: 13px;
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  margin-left: 8px;
}
.log-header button:disabled {
  background-color: #eee;
  cursor: not-allowed;
}
.log-list {
  list-style: none;
  padding: 0;
  margin: 0;
  overflow-y: auto;
  flex-grow: 1;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, Courier, monospace;
  font-size: 12px;
  background-color: #fff;
}
.log-item {
  padding: 8px 12px;
  display: flex;
  gap: 16px;
  border-bottom: 1px solid #f0f0f0;
}
.log-item.log-error {
  color: #d8000c;
}
.log-item.log-success {
  color: #008000;
}
.log-timestamp {
  color: #666;
  flex-shrink: 0;
}
.log-message {
  white-space: pre-wrap;
  word-break: break-word;
}
.log-item.empty {
  justify-content: center;
  color: #999;
  padding: 20px;
}
</style>
