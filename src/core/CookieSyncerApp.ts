/**
 * CookieSyncer - 现代化核心应用类
 * 使用TypeScript和现代JavaScript特性，提供类型安全和更好的架构
 */

import type {
    DomainConfig,
    ExtensionState
} from '../../types/extension';

// 现代化域名管理器类
class ModernDomainManager {
    private domains: Map<string, DomainConfig> = new Map();

    async initialize(): Promise<void> {
        // 从Chrome存储加载域名配置
        const result = await chrome.storage.local.get(['cookiesyncer_domains']);
        const storedDomains = result.cookiesyncer_domains || {};

        for (const [domain, config] of Object.entries(storedDomains)) {
            this.domains.set(domain, config as DomainConfig);
        }

        console.log(`[ModernDomainManager] 加载了 ${this.domains.size} 个域名配置`);
    }

    getAllDomains(): DomainConfig[] {
        return Array.from(this.domains.values());
    }

    getDomain(domain: string): DomainConfig | null {
        return this.domains.get(domain) || null;
    }

    async addDomain(domain: string, config: Partial<DomainConfig>): Promise<void> {
        const newConfig: DomainConfig = {
            domain,
            enabled: config.enabled ?? true,
            syncInterval: config.syncInterval ?? 60,
            autoRenewal: config.autoRenewal ?? true,
            cookieFields: config.cookieFields ?? [],
            renewalStrategy: config.renewalStrategy ?? 'fetch',
            lastSyncTime: null,
            lastSyncStatus: 'never',
            syncLogs: [],
            createdTime: new Date().toISOString(),
            updatedTime: new Date().toISOString()
        };

        this.domains.set(domain, newConfig);
        await this.saveToStorage();
    }

    async updateDomain(domain: string, updates: Partial<DomainConfig>): Promise<void> {
        const config = this.domains.get(domain);
        if (!config) {
            throw new Error(`域名不存在: ${domain}`);
        }

        Object.assign(config, updates, { updatedTime: new Date().toISOString() });
        await this.saveToStorage();
    }

    async removeDomain(domain: string): Promise<void> {
        if (!this.domains.has(domain)) {
            throw new Error(`域名不存在: ${domain}`);
        }
        this.domains.delete(domain);
        await this.saveToStorage();
    }
 
    private async saveToStorage(): Promise<void> {
        const data = Object.fromEntries(this.domains);
        await chrome.storage.local.set({ cookiesyncer_domains: data });
    }
}

// 现代化推送管理器类
class ModernSyncManager {
    private domainManager: ModernDomainManager;

    async initialize(domainManager: ModernDomainManager): Promise<void> {
        this.domainManager = domainManager;
        await this.restoreAlarms();
    }

    async restoreAlarms(): Promise<void> {
        // 清除现有定时器
        await chrome.alarms.clearAll();

        const enabledDomains = this.domainManager.getAllDomains().filter(d => d.enabled);

        for (const domainConfig of enabledDomains) {
            const alarmName = `sync_${domainConfig.domain}`;
            await chrome.alarms.create(alarmName, {
                delayInMinutes: domainConfig.syncInterval,
                periodInMinutes: domainConfig.syncInterval
            });
        }

        console.log(`[ModernSyncManager] 恢复了 ${enabledDomains.length} 个定时任务`);
    }

    async syncAllDomains(): Promise<any> {
        const domains = this.domainManager.getAllDomains().filter(d => d.enabled);
        const results = [];

        for (const domainConfig of domains) {
            try {
                const result = await this.syncDomain(domainConfig.domain);
                results.push({ domain: domainConfig.domain, success: result.success });
            } catch (error) {
                results.push({ domain: domainConfig.domain, success: false, error: (error as Error).message });
            }
        }

        return {
            success: true,
            results,
            summary: `${results.filter(r => r.success).length}/${results.length} 个域名推送成功`
        };
    }

    async syncDomain(domain: string): Promise<any> {
        const config = this.domainManager.getDomain(domain);
        if (!config || !config.enabled) {
            throw new Error('域名未配置或已禁用');
        }

        // 更新状态为推送中
        await this.domainManager.updateDomain(domain, { lastSyncStatus: 'syncing' });

        // 执行续期策略
        const renewalResult = await this.executeRenewalStrategy(domain, config);

        // 获取Cookie
        const cookies = await chrome.cookies.getAll({ domain });

        // 更新状态
        await this.domainManager.updateDomain(domain, {
            lastSyncTime: new Date().toISOString(),
            lastSyncStatus: renewalResult.success ? 'success' : 'failed'
        });

        return {
            success: renewalResult.success,
            renewal: renewalResult,
            cookies: cookies
        };
    }

    private async executeRenewalStrategy(domain: string, config: DomainConfig): Promise<any> {
        if (!config.autoRenewal) {
            return { success: true, strategy: 'disabled' };
        }

        try {
            const response = await fetch(`https://${domain}`, {
                credentials: 'include',
                headers: {
                    'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
                }
            });

            return {
                success: response.ok,
                strategy: 'fetch',
                status: response.status
            };
        } catch (error) {
            return {
                success: false,
                strategy: 'fetch',
                error: (error as Error).message
            };
        }
    }
}

// 主应用类
export class CookieSyncerApp {
    private domainManager: ModernDomainManager;
    private syncManager: ModernSyncManager;
    private state: ExtensionState;

    constructor() {
        this.domainManager = new ModernDomainManager();
        this.syncManager = new ModernSyncManager();
        this.state = {
            isInitialized: false,
            domains: new Map(),
            logs: new Map()
        };
    }

    /**
     * 初始化应用
     */
    async initialize(): Promise<void> {
        try {
            console.log('[CookieSyncer] 🚀 开始现代化初始化...');

            // 初始化管理器
            await this.domainManager.initialize();
            await this.syncManager.initialize(this.domainManager);

            // 更新状态
            this.state.isInitialized = true;
            this.state.domains = new Map(
                this.domainManager.getAllDomains().map(config => [config.domain, config])
            );

            console.log('[CookieSyncer] 🎉 现代化初始化完成！');
        } catch (error) {
            console.error('[CookieSyncer] ❌ 初始化失败:', error);
            throw error;
        }
    }

    /**
     * 处理扩展消息
     */
    async handleMessage(action: string, data?: any): Promise<any> {
        switch (action) {
            case 'syncAllDomains':
                return await this.syncManager.syncAllDomains();
 
            case 'syncDomain':
                return await this.syncManager.syncDomain(data.domain);
 
            case 'getAllDomains':
                return this.domainManager.getAllDomains();
            
            case 'addDomain':
                await this.domainManager.addDomain(data.domain, data.config || {});
                return { success: true };
            
            case 'updateDomain':
                await this.domainManager.updateDomain(data.domain, data.updates);
                return { success: true };

            case 'removeDomain':
                await this.domainManager.removeDomain(data.domain);
                return { success: true };
 
            case 'getDomainCookies':
                const cookies = await chrome.cookies.getAll({ domain: data.domain });
                return { success: true, cookies };

            case 'getCurrentTabCookies':
                const [currentTab] = await chrome.tabs.query({ active: true, currentWindow: true });
                if (currentTab && currentTab.url) {
                    const url = new URL(currentTab.url);
                    const tabCookies = await chrome.cookies.getAll({ domain: url.hostname });
                    return { success: true, cookies: tabCookies, domain: url.hostname };
                }
                return { success: false, error: '无法获取当前标签页' };
 
            case 'deleteCookie':
                const result = await chrome.cookies.remove(data.cookie);
                return { success: !!result };
 
            default:
                throw new Error(`未知操作: ${action}`);
        }
    }

    /**
     * 获取应用状态
     */
    getState(): ExtensionState {
        return { ...this.state };
    }
}
