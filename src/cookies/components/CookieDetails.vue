<template>
  <div class="cookie-details">
    <div class="detail-item">
      <label>名称:</label>
      <span>{{ cookie.name }}</span>
    </div>
    
    <div class="detail-item">
      <label>值:</label>
      <span class="cookie-value">{{ cookie.value }}</span>
    </div>
    
    <div class="detail-item">
      <label>域名:</label>
      <span>{{ cookie.domain }}</span>
    </div>
    
    <div class="detail-item">
      <label>路径:</label>
      <span>{{ cookie.path }}</span>
    </div>
    
    <div class="detail-item">
      <label>过期时间:</label>
      <span>{{ formatExpiration(cookie.expirationDate) }}</span>
    </div>
    
    <div class="detail-item">
      <label>安全:</label>
      <span :class="{ 'secure': cookie.secure, 'not-secure': !cookie.secure }">
        {{ cookie.secure ? '是' : '否' }}
      </span>
    </div>
    
    <div class="detail-item">
      <label>HttpOnly:</label>
      <span>{{ cookie.httpOnly ? '是' : '否' }}</span>
    </div>
    
    <div class="detail-item">
      <label>SameSite:</label>
      <span>{{ formatSameSite(cookie.sameSite) }}</span>
    </div>
    
    <div class="detail-item">
      <label>会话Cookie:</label>
      <span>{{ cookie.session ? '是' : '否' }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from 'vue'
import type { Cookie } from '../../../types/extension'

const props = defineProps<{
  cookie: Cookie
}>()

const formatExpiration = (expirationDate?: number) => {
  if (!expirationDate) return '会话Cookie'
  return new Date(expirationDate * 1000).toLocaleString()
}
 
const formatSameSite = (sameSite?: chrome.cookies.SameSiteStatus) => {
  if (!sameSite) return '未指定'
  const mapping = {
    'unspecified': '未指定',
    'no_restriction': '无限制',
    'lax': '宽松',
    'strict': '严格'
  }
  return mapping[sameSite] || sameSite
}
</script>

<style scoped>
.cookie-details {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.detail-item:last-child {
  border-bottom: none;
}

.detail-item label {
  font-weight: 600;
  color: #333;
  min-width: 100px;
}

.detail-item span {
  color: #666;
  word-break: break-all;
}

.cookie-value {
  font-family: monospace;
  background: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.secure {
  color: #28a745;
  font-weight: 600;
}

.not-secure {
  color: #dc3545;
  font-weight: 600;
}
</style>
