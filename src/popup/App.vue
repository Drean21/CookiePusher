<template>
  <div class="popup-container">
    <header class="popup-header">
      <h1>CookieSyncer</h1>
    </header>
    <main class="popup-content">
      <component :is="activeComponent" />
    </main>
    <div v-if="notification.visible" class="notification" :class="notification.type">
      {{ notification.message }}
    </div>
    <footer class="popup-footer">
      <nav class="tab-nav">
        <a @click="activeView = 'current'" :class="{ active: activeView === 'current' }"
          >当前页面</a
        >
        <a @click="activeView = 'managed'" :class="{ active: activeView === 'managed' }"
          >域名管理</a
        >
        <a @click="activeView = 'settings'" :class="{ active: activeView === 'settings' }"
          >设置</a
        >
      </nav>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, provide } from "vue";
import CurrentPageView from "./views/CurrentPageView.vue";
import ManagedDomainsView from "./views/ManagedDomainsView.vue";
import SettingsView from "./views/SettingsView.vue";

type View = "current" | "managed" | "settings";
type NotificationType = 'success' | 'error';

const activeView = ref<View>("current");

const notification = ref({
  visible: false,
  message: '',
  type: 'success' as NotificationType
});

let notificationTimer: any;

const showNotification = (message: string, type: NotificationType = 'success', duration: number = 3000) => {
  notification.value = { visible: true, message, type };
  if (notificationTimer) {
    clearTimeout(notificationTimer);
  }
  notificationTimer = setTimeout(() => {
    notification.value.visible = false;
  }, duration);
};

provide('showNotification', showNotification);


const activeComponent = computed(() => {
  switch (activeView.value) {
    case "current":
      return CurrentPageView;
    case "managed":
      return ManagedDomainsView;
    case "settings":
      return SettingsView;
    default:
      return CurrentPageView;
  }
});
</script>

<style>
html,
body {
  margin: 0;
  padding: 0;
  width: 600px;
  min-height: 500px;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue",
    Arial, sans-serif;
  color: #333;
}
.popup-container {
  display: flex;
  flex-direction: column;
  width: 600px;
  height: 500px;
  background-color: #f9f9f9;
}
.popup-header {
  padding: 12px;
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}
.popup-header h1 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #424242;
}
.popup-content {
  flex-grow: 1;
  overflow: hidden;
  display: flex;
  /* The padding is removed from here and will be handled by individual views */
}
.popup-footer {
  border-top: 1px solid #e0e0e0;
  background-color: #fff;
  box-shadow: 0 -1px 3px rgba(0, 0, 0, 0.05);
}
.tab-nav {
  display: flex;
  justify-content: space-around;
  padding: 4px 0;
}
.tab-nav a {
  flex: 1;
  padding: 10px 12px;
  cursor: pointer;
  transition: all 0.2s ease-in-out;
  color: #555;
  text-decoration: none;
  text-align: center;
  font-size: 14px;
  font-weight: 500;
}
.tab-nav a:hover {
  background-color: #f0f2f5;
}
.tab-nav a.active {
  color: #667eea;
  border-bottom: 3px solid #667eea;
  background-color: #f0f2f5;
}
</style>
<style scoped>
.notification {
  position: fixed;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  padding: 10px 20px;
  border-radius: 6px;
  color: white;
  font-size: 14px;
  font-weight: 500;
  z-index: 1000;
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
  transition: all 0.3s ease-in-out;
}
.notification.success {
  background-color: #4caf50;
}
.notification.error {
  background-color: #f44336;
}
</style>
