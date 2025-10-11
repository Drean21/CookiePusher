// CookieSyncer 扩展类型定义

// 域名配置接口
export interface DomainConfig {
  domain: string;
  enabled: boolean;
  syncInterval: number;
  autoRenewal: boolean;
  cookieFields: string[];
  renewalStrategy: 'fetch' | 'tab' | 'script';
  lastSyncTime: string | null;
  lastSyncStatus: 'never' | 'success' | 'failed' | 'syncing';
  syncLogs: SyncLog[];
  maxLogs?: number;
  createdTime: string;
  updatedTime: string;
}

// 同步日志接口
export interface SyncLog {
  id: string;
  domain: string;
  action: 'sync' | 'renewal' | 'error';
  status: 'success' | 'failed' | 'info';
  details: Record<string, any>;
  timestamp: string;
}

// 同步结果接口
export interface SyncResult {
  success: boolean;
  renewal?: RenewalResult;
  cookies?: CookieResult;
  error?: string;
}

// 续期结果接口
export interface RenewalResult {
  success: boolean;
  strategy: string;
  status?: number;
  tabId?: number;
  error?: string;
}

// Cookie结果接口
export interface CookieResult {
  success: boolean;
  cookies: chrome.cookies.Cookie[];
  totalCount: number;
  filteredCount: number;
  error?: string;
}

// 消息接口
export interface ExtensionMessage {
  action: string;
  domain?: string;
  cookie?: chrome.cookies.Cookie;
  [key: string]: any;
}

// 扩展状态接口
export interface ExtensionState {
  isInitialized: boolean;
  domains: Map<string, DomainConfig>;
  logs: Map<string, SyncLog[]>;
}

// 事件类型
export type ExtensionEvent =
  | 'domain-added'
  | 'domain-removed'
  | 'domain-updated'
  | 'sync-started'
  | 'sync-completed'
  | 'cookie-renewed'
  | 'error-occurred';

// Chrome扩展API增强类型
declare global {
  interface Window {
    CookieSyncer?: CookieSyncerApp;
  }
}

export interface Cookie extends chrome.cookies.Cookie {
  isSharable?: boolean;
}
