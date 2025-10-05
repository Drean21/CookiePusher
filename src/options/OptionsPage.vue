<template>
  <div class="options-page">
    <header class="page-header">
      <h1>Cookie Syncer 设置</h1>
    </header>

    <div class="settings-section">
      <h2>同步设置</h2>
      
      <div class="setting-item">
        <label class="setting-label">
          <input 
            type="checkbox" 
            v-model="settings.autoSync" 
            @change="saveSettings"
          />
          自动同步Cookie
        </label>
        <span class="setting-description">启用后，插件会自动同步已配置域名的Cookie</span>
      </div>

      <div class="setting-item">
        <label class="setting-label">同步间隔（分钟）</label>
        <input 
          type="number" 
          v-model="settings.syncInterval" 
          min="1" 
          max="60"
          @change="saveSettings"
          class="setting-input"
        />
        <span class="setting-description">设置Cookie同步的时间间隔</span>
      </div>

      <div class="setting-item">
        <label class="setting-label">
          <input 
            type="checkbox" 
            v-model="settings.filterSensitive" 
            @change="saveSettings"
          />
          过滤敏感Cookie
        </label>
        <span class="setting-description">自动过滤包含session、token、auth等敏感信息的Cookie</span>
      </div>
    </div>

    <div class="settings-section">
      <h2>域名管理</h2>
      
      <div class="domain-management">
        <div class="add-domain">
          <input 
            v-model="newDomain" 
            type="text" 
            placeholder="输入域名（如：example.com）"
            class="domain-input"
            @keyup.enter="addDomain"
          />
          <button @click="addDomain" class="btn btn-primary">添加域名</button>
        </div>

        <div class="domain-list">
          <div 
            v-for="domain in settings.domains" 
            :key="domain"
            class="domain-item"
          >
            <span class="domain-name">{{ domain }}</span>
            <button 
              @click="removeDomain(domain)" 
              class="btn btn-danger btn-sm"
            >
              删除
            </button>
          </div>
          
          <div v-if="settings.domains.length === 0" class="empty-state">
            暂无配置的域名
          </div>
        </div>
      </div>
    </div>

    <div class="settings-section">
      <h2>数据管理</h2>
      
      <div class="data-actions">
        <button @click="exportData" class="btn btn-secondary">
          导出配置数据
        </button>
        <button @click="importData" class="btn btn-secondary">
          导入配置数据
        </button>
        <button @click="clearData" class="btn btn-danger">
          清除所有数据
        </button>
      </div>
    </div>

    <div class="settings-section">
      <h2>关于</h2>
      
      <div class="about-info">
        <p><strong>版本：</strong> {{ version }}</p>
        <p><strong>最后同步：</strong> {{ formatLastSync(settings.lastSync) }}</p>
        <p><strong>同步域名数量：</strong> {{ settings.domains.length }}</p>
      </div>
    </div>

    <!-- 导入数据模态框 -->
    <div v-if="showImportModal" class="modal-overlay" @click="closeImportModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>导入配置数据</h3>
          <button @click="closeImportModal" class="close-btn">×</button>
        </div>
        <div class="modal-body">
          <textarea 
            v-model="importDataText" 
            placeholder="粘贴导出的JSON数据..."
            class="import-textarea"
          ></textarea>
          <div class="modal-actions">
            <button @click="confirmImport" class="btn btn-primary">确认导入</button>
            <button @click="closeImportModal" class="btn btn-secondary">取消</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'

interface Settings {
  autoSync: boolean
  syncInterval: number
  filterSensitive: boolean
  domains: string[]
  lastSync?: number
}

// 响应式数据
const version = '1.0.0'
const newDomain = ref('')
const showImportModal = ref(false)
const importDataText = ref('')

const settings = reactive<Settings>({
  autoSync: true,
  syncInterval: 5,
  filterSensitive: true,
  domains: [],
  lastSync: undefined
})

// 方法
const loadSettings = async () => {
  try {
    const result = await chrome.storage.local.get([
      'autoSync',
      'syncInterval', 
      'filterSensitive',
      'domains',
      'lastSync'
    ])
    
    Object.assign(settings, {
      autoSync: result.autoSync ?? true,
      syncInterval: result.syncInterval ?? 5,
      filterSensitive: result.filterSensitive ?? true,
      domains: result.domains ?? [],
      lastSync: result.lastSync
    })
  } catch (error) {
    console.error('加载设置失败:', error)
  }
}

const saveSettings = async () => {
  try {
    await chrome.storage.local.set({
      autoSync: settings.autoSync,
      syncInterval: settings.syncInterval,
      filterSensitive: settings.filterSensitive,
      domains: settings.domains,
      lastSync: settings.lastSync
    })
    
    console.log('设置已保存')
  } catch (error) {
    console.error('保存设置失败:', error)
  }
}

const addDomain = () => {
  if (newDomain.value.trim() && !settings.domains.includes(newDomain.value.trim())) {
    settings.domains.push(newDomain.value.trim())
    newDomain.value = ''
    saveSettings()
  }
}

const removeDomain = (domain: string) => {
  const index = settings.domains.indexOf(domain)
  if (index > -1) {
    settings.domains.splice(index, 1)
    saveSettings()
  }
}

const exportData = async () => {
  try {
    const data = await chrome.storage.local.get(null)
    const blob = new Blob([JSON.stringify(data, null, 2)], { 
      type: 'application/json' 
    })
    
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `cookiesyncer-backup-${new Date().toISOString().split('T')[0]}.json`
    a.click()
    
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error('导出数据失败:', error)
    alert('导出数据失败，请重试')
  }
}

const importData = () => {
  showImportModal.value = true
}

const confirmImport = async () => {
  try {
    const data = JSON.parse(importDataText.value)
    await chrome.storage.local.clear()
    await chrome.storage.local.set(data)
    
    await loadSettings()
    closeImportModal()
    alert('数据导入成功')
  } catch (error) {
    console.error('导入数据失败:', error)
    alert('导入数据失败，请检查数据格式')
  }
}

const closeImportModal = () => {
  showImportModal.value = false
  importDataText.value = ''
}

const clearData = async () => {
  if (confirm('确定要清除所有数据吗？此操作不可逆！')) {
    try {
      await chrome.storage.local.clear()
      await loadSettings()
      alert('数据已清除')
    } catch (error) {
      console.error('清除数据失败:', error)
      alert('清除数据失败')
    }
  }
}

const formatLastSync = (timestamp?: number) => {
  if (!timestamp) return '从未同步'
  return new Date(timestamp).toLocaleString('zh-CN')
}

// 生命周期
onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.options-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
  font-family: system-ui, -apple-system, sans-serif;
}

.page-header {
  margin-bottom: 32px;
  text-align: center;
}

.page-header h1 {
  margin: 0;
  color: #333;
  font-size: 28px;
}

.settings-section {
  margin-bottom: 40px;
  padding: 24px;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  background: white;
}

.settings-section h2 {
  margin: 0 0 20px 0;
  color: #333;
  font-size: 20px;
}

.setting-item {
  margin-bottom: 20px;
}

.setting-label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: #333;
}

.setting-description {
  display: block;
  font-size: 14px;
  color: #666;
  margin-top: 4px;
}

.setting-input {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
  width: 100px;
}

.domain-management {
  margin-top: 16px;
}

.add-domain {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.domain-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.domain-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.domain-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 4px;
}

.domain-name {
  font-weight: 500;
  color: #333;
}

.btn-sm {
  padding: 4px 8px;
  font-size: 12px;
}

.btn-danger {
  background: #dc3545;
  color: white;
}

.btn-danger:hover {
  background: #c82333;
}

.empty-state {
  text-align: center;
  color: #666;
  font-style: italic;
  padding: 20px;
}

.data-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.about-info p {
  margin: 8px 0;
  color: #333;
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
  max-width: 500px;
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

.import-textarea {
  width: 100%;
  height: 200px;
  padding: 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-family: monospace;
  font-size: 14px;
  resize: vertical;
  margin-bottom: 16px;
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}
</style>