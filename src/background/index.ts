/**
 * CookieSyncer - "Automatic & Non-Intrusive" Backend
 */

import CryptoJS from 'crypto-js';

import { Cookie } from '../../types/extension';

const LOGS_STORAGE_KEY = 'cookieSyncerLogs';
const SYNC_LIST_STORAGE_KEY = 'syncList';
const STATS_STORAGE_KEY = 'keepAliveStats';
const SYNC_QUEUE_STORAGE_KEY = 'syncQueue';
const MAX_LOGS = 100;

const RETRY_ALARM_NAME = 'syncRetryAlarm';
// =================================================================
// Enhanced Statistics Interfaces
// =================================================================
interface StatHistory {
    status: 'success' | 'failure' | 'no-change';
    timestamp: string;
    changeSource: 'keep-alive' | 'on-change';
    intervalSeconds?: number;
    error?: string; // Add error message field
}

interface KeepAliveStat {
    successCount: number;
    failureCount: number;
    history: StatHistory[];
    expirationDate?: number;
    value?: string;
    lastChangeTimestamp?: string;
}

async function addLog(message: string, type: 'info' | 'success' | 'error' = 'info') {
    const logEntry = {
        message,
        type,
        timestamp: new Date().toISOString()
    };
    console.log(`[CookieSyncer Log - ${type}]: ${message}`);
    try {
        const { [LOGS_STORAGE_KEY]: logs = [] } = await chrome.storage.local.get(LOGS_STORAGE_KEY);
        logs.unshift(logEntry);
        if (logs.length > MAX_LOGS) {
            logs.length = MAX_LOGS;
        }
        await chrome.storage.local.set({ [LOGS_STORAGE_KEY]: logs });
    } catch (e) {
        console.error("Failed to write to log storage:", e);
    }
}

// Main message listener
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    const { action, payload } = message;
    let isAsync = false;

    const actionMap: { [key: string]: (payload: any) => Promise<any> } = {
        getCookiesForCurrentTab: handleGetCookiesForCurrentTab,
        syncSingleCookie: handleSyncCookies,
        syncAllCookiesForDomain: handleSyncCookies,
        getCookiesForDomain: handleGetCookiesForDomain,
        removeCookieFromSyncList: handleRemoveCookieFromSync,
        removeDomainFromSyncList: handleRemoveDomainFromSync,
        testApiConnection: handleTestApiConnection,
        manualSync: handleManualSync,
        getLogs: async () => {
            const result = await chrome.storage.local.get(LOGS_STORAGE_KEY);
            return { logs: result[LOGS_STORAGE_KEY] || [] };
        },
        clearLogs: async () => {
            await chrome.storage.local.remove(LOGS_STORAGE_KEY);
            addLog('日志已清空。', 'info');
            return {};
        },
        addLog: async (p) => { if (p) addLog(p.message, p.type); return {}; },
        getKeepAliveStats: handleGetKeepAliveStats,
        keepAliveTaskFinished: handleKeepAlivePostTasks,
        updateSyncList: async (p) => {
            if (!p || !Array.isArray(p.syncList)) {
                throw new Error("Invalid syncList provided for update.");
            }
            await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: p.syncList });
            await triggerFullSync();
            return { success: true };
        },
        exportAllData: handleExportAllData,
        importAllData: handleImportAllData,
        getUserSettings: handleGetUserSettings,
        updateUserSettings: handleUpdateUserSettings,
        updateCookieRemark: handleUpdateCookieRemark,
    };

    if (actionMap[action]) {
        isAsync = true;
        actionMap[action](payload)
            .then(response => sendResponse({ success: true, ...response }))
            .catch(error => {
                const errorMessage = error.message || 'An unknown error occurred';
                addLog(`${action} failed: ${errorMessage}`, 'error');
                sendResponse({ success: false, error: errorMessage });
            });
    }
    
    return isAsync;
});


async function handleGetCookiesForCurrentTab() {
    const [currentTab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (!currentTab?.id || !currentTab.url) throw new Error('没有找到活动的标签页。');
    if (isProtectedUrl(currentTab.url)) return { groupedCookies: {}, domain: '受保护页面' };

    const injectionResults = await chrome.scripting.executeScript({
        target: { tabId: currentTab.id, allFrames: true },
        func: () => document.domain,
    }).catch(error => {
        console.warn(`[CookieSyncer] Scripting injection failed: ${error.message}`);
        return [{ result: new URL(currentTab.url!).hostname }];
    });

    const domains = new Set<string>();
    if (injectionResults) {
        injectionResults.forEach(item => { if (item.result) domains.add(item.result); });
    }
    domains.add(new URL(currentTab.url).hostname);

    const allBrowserCookies = await chrome.cookies.getAll({});
    const relevantCookies = filterCookiesByDomains(allBrowserCookies, domains);
    const uniqueCookies = Array.from(new Map(relevantCookies.map(c => [getCookieKey(c), c])).values());
    
    const groupedCookies = uniqueCookies.reduce((acc, cookie) => {
        const groupKey = getRegistrableDomain(cookie.domain);
        if (!acc[groupKey]) acc[groupKey] = [];
        acc[groupKey].push(cookie);
        return acc;
    }, {} as { [key: string]: Cookie[] });

    return {
        groupedCookies: groupedCookies,
        domain: new URL(currentTab.url).hostname,
    };
}

function filterCookiesByDomains(allCookies: Cookie[], domains: Set<string>): Cookie[] {
    const relevantCookies: Cookie[] = [];
    const pageDomains = Array.from(domains).map(d => d.toLowerCase());
    for (const cookie of allCookies) {
        const cookieDomain = cookie.domain.startsWith('.') ? cookie.domain.substring(1).toLowerCase() : cookie.domain.toLowerCase();
        if (pageDomains.some(pd => pd === cookieDomain || pd.endsWith(`.${cookieDomain}`))) {
            relevantCookies.push(cookie);
        }
    }
    return relevantCookies;
}

function isProtectedUrl(url: string): boolean {
    const protectedSchemes = ['chrome://', 'about:', 'edge://', 'moz-extension://'];
    return protectedSchemes.some(scheme => url.startsWith(scheme));
}

const getCookieKey = (cookie: Cookie): string => `${cookie.name}|${cookie.domain}|${cookie.path}`;

function getRegistrableDomain(domain: string): string {
    if (domain.startsWith('.')) domain = domain.substring(1);
    const parts = domain.split('.');
    if (parts.length <= 2) return domain;
    const twoLevelTlds = new Set(['com.cn', 'org.cn', 'net.cn', 'gov.cn', 'co.uk', 'co.jp']);
    const lastTwo = parts.slice(-2).join('.');
    if (twoLevelTlds.has(lastTwo) && parts.length > 2) {
        return parts.slice(-3).join('.');
    }
    return lastTwo;
}

async function handleSyncCookies(payload: { cookie?: Cookie, cookies?: Cookie[] }) {
    if (!payload || (!payload.cookie && !payload.cookies)) throw new Error('无效的推送请求负载。');
    const newCookies = payload.cookies || (payload.cookie ? [payload.cookie] : []);
    if (newCookies.length === 0) return { message: "没有需要推送的Cookie。" };
    
    const { [SYNC_LIST_STORAGE_KEY]: currentSyncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    const syncMap = new Map(currentSyncList.map(c => [getCookieKey(c), c]));

    let addedCount = 0;
    newCookies.forEach(c => {
        const key = getCookieKey(c);
        if (!syncMap.has(key)) addedCount++;
        // Preserve isSharable state if the cookie already exists
        const existingCookie = syncMap.get(key);
        const isSharable = c.isSharable ?? existingCookie?.isSharable ?? false;
        const remark = c.remark ?? existingCookie?.remark ?? "";
        syncMap.set(key, { ...c, isSharable, remark });
    });

    const updatedSyncList = Array.from(syncMap.values());
    await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedSyncList });
    
    await triggerFullSync(); // Trigger sync after local update
    
    const logMessage = `推送列表更新: 新增 ${addedCount}, 总计 ${updatedSyncList.length}`;
    addLog(logMessage, 'success');
    return { added: addedCount, total: updatedSyncList.length, message: logMessage };
}

async function handleGetCookiesForDomain(payload: { domain: string }) {
    if (!payload?.domain) throw new Error('无效的域名。');
    return { cookies: await chrome.cookies.getAll({ domain: payload.domain }) };
}

async function handleRemoveCookieFromSync(payload: { cookie: Cookie }) {
    if (!payload?.cookie) throw new Error('无效的Cookie以进行移除。');
    const { cookie: cookieToRemove } = payload;
    const { [SYNC_LIST_STORAGE_KEY]: currentSyncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    const keyToRemove = getCookieKey(cookieToRemove);
    const updatedSyncList = currentSyncList.filter(c => getCookieKey(c) !== keyToRemove);
    
    await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedSyncList });
    await triggerFullSync(); // Trigger sync after local update

    const logMessage = `从推送列表移除: ${cookieToRemove.name}.`;
    addLog(logMessage, 'info');
    return { removed: true, total: updatedSyncList.length, message: logMessage };
}

async function handleRemoveDomainFromSync(payload: { domain: string }) {
    if (!payload?.domain) throw new Error('无效的域名以进行移除。');
    const { domain: domainToRemove } = payload;
    const { [SYNC_LIST_STORAGE_KEY]: currentSyncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    const updatedSyncList = currentSyncList.filter(c => getRegistrableDomain(c.domain) !== domainToRemove);
    
    await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedSyncList });
    await triggerFullSync(); // Trigger sync after local update

    const logMessage = `从推送列表移除域名: ${domainToRemove}.`;
    addLog(logMessage, 'info');
    return { removed: true, total: updatedSyncList.length, message: logMessage, syncList: updatedSyncList };
}

const SECRET_KEY = 'cookie-syncer-secret-key';

async function getDecryptedSettings() {
    const { syncSettings } = await chrome.storage.local.get('syncSettings');
    if (!syncSettings?.apiEndpoint || !syncSettings?.authToken) throw new Error('API端点或Auth Token未设置。');
    try {
        const bytes = CryptoJS.AES.decrypt(syncSettings.authToken, SECRET_KEY);
        const authToken = bytes.toString(CryptoJS.enc.Utf8);
        if (!authToken) throw new Error("解密后的Token为空。");
        return {
            apiEndpoint: syncSettings.apiEndpoint,
            authToken: authToken,
            keepAliveFrequency: syncSettings.keepAliveFrequency
        };
    } catch (e) {
        throw new Error('Auth Token解密失败，请检查设置。');
    }
}

async function handleTestApiConnection() {
    const { apiEndpoint, authToken } = await getDecryptedSettings();
    const url = new URL(apiEndpoint);
    url.pathname = '/api/v1/auth/test';
    
    const response = await fetch(url.toString(), {
        method: 'GET',
        headers: { 'x-api-key': authToken, 'Content-Type': 'application/json' }
    });

    if (!response.ok) {
        if (response.status === 401) {
            throw new Error('认证失败: API密钥无效或已过期。');
        }
        throw new Error(`服务器响应错误: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();
    if (data.code !== 200 || data.message !== 'Token is valid') {
        throw new Error(`认证接口响应异常: ${JSON.stringify(data)}`);
    }
    
    addLog('API连接和认证测试成功！', 'success');
    return { connected: true };
}

// Debounced sync scheduler
let syncDebounceTimer: NodeJS.Timeout | null = null;
function scheduleSync() {
    const debounceInterval = 15000; // 15 seconds, as per user's manual change

    if (syncDebounceTimer) {
        clearTimeout(syncDebounceTimer);
    } else {
        // Only log when a new debounce window starts.
        addLog(`同步窗口已开启，将在 ${debounceInterval / 1000} 秒后合并所有变更。`, 'info');
    }

    syncDebounceTimer = setTimeout(() => {
        addLog('变更合并窗口结束，触发云端推送。', 'info');
        triggerFullSync().catch(() => { /* Errors are handled inside */ });
        syncDebounceTimer = null; // Clear the timer reference after execution
    }, debounceInterval);
}

// Centralized sync function with retry queue
async function triggerFullSync() {
    try {
        const { apiEndpoint, authToken } = await getDecryptedSettings();
        
        // --- Start: Queueing Logic ---
        const { [SYNC_LIST_STORAGE_KEY]: localSyncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
        const { [SYNC_QUEUE_STORAGE_KEY]: queuedSyncList = [] } = await chrome.storage.local.get(SYNC_QUEUE_STORAGE_KEY) as { [SYNC_QUEUE_STORAGE_KEY]: Cookie[] };

        const combinedList = [...localSyncList];
        const combinedMap = new Map(combinedList.map(c => [getCookieKey(c), c]));

        // Merge queued items, giving precedence to newer items from localSyncList
        queuedSyncList.forEach((c: Cookie) => {
            const key = getCookieKey(c);
            if (!combinedMap.has(key)) {
                combinedMap.set(key, c);
            }
        });
        const listToSync = Array.from(combinedMap.values());

        if (listToSync.length === 0) {
            addLog("推送列表为空，跳过云端推送。", 'info');
            await chrome.storage.local.remove(SYNC_QUEUE_STORAGE_KEY); // Clear any old queue
            return;
        }

        // Place the final list into the queue before attempting to sync
        await chrome.storage.local.set({ [SYNC_QUEUE_STORAGE_KEY]: listToSync });
        // --- End: Queueing Logic ---

        const syncUrl = new URL(apiEndpoint);
        syncUrl.pathname = '/api/v1/sync';

        const response = await fetch(syncUrl.toString(), {
            method: 'POST',
            headers: { 'x-api-key': authToken, 'Content-Type': 'application/json' },
            body: JSON.stringify(listToSync.map(transformCookieForAPI))
        });

        if (!response.ok) {
            const errorBody = await response.text();
            throw new Error(`推送到云端失败: ${response.status} - ${errorBody}`);
        }
        
        const responseData = await response.json();

        if (responseData.code !== 200) {
            throw new Error(`推送响应格式错误: ${JSON.stringify(responseData)}`);
        }
        
        addLog(`云端推送成功，同步了 ${listToSync.length} 个Cookie。`, 'success');
        
        // --- Success: Clear Queue and Retry Alarm ---
        await chrome.storage.local.remove(SYNC_QUEUE_STORAGE_KEY);
        await chrome.alarms.clear(RETRY_ALARM_NAME);
        
    } catch (e: any) {
        addLog(`云端推送失败: ${e.message}。数据已暂存，将在稍后重试。`, 'error');
        
        // --- Failure: Schedule Retry ---
        chrome.alarms.create(RETRY_ALARM_NAME, {
            delayInMinutes: 5 // Retry after 5 minutes
        });
        
        // Re-throw the error so the original caller can catch it.
        throw e;
    }
}

async function handleManualSync() {
    await addLog('手动推送开始...', 'info');
    await triggerFullSync();
    return { message: "手动推送已触发。" };
}

let monitoredCookies = new Set<string>();

async function refreshMonitoredCookies() {
    const { syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    monitoredCookies = new Set(syncList.map(c => getCookieKey(c)));
}

chrome.cookies.onChanged.addListener(async (changeInfo) => {
    if (changeInfo.cause !== 'explicit' || changeInfo.removed) return;
    const key = getCookieKey(changeInfo.cookie);

    if (monitoredCookies.has(key)) {
        // Log every detected change to provide rich feedback, then let the scheduler handle debouncing.
        addLog(`检测到变更: ${key}，已加入推送队列。`, 'info');
        try {
            // Immediately update the local list to stage the change
            const { [SYNC_LIST_STORAGE_KEY]: syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
            const updatedList = syncList.map((c: Cookie) => getCookieKey(c) === key ? { ...c, ...changeInfo.cookie } : c);
            
            if (!updatedList.some(c => getCookieKey(c) === key)) {
                updatedList.push(changeInfo.cookie);
            }
            await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedList });

            // Unified debounce call
            scheduleSync();

        } catch (e: any) {
            addLog(`处理Cookie变更暂存时出错: ${e.message || '未知错误'}`, 'error');
        }
    }
});

const KEEPALIVE_ALARM_NAME = 'cookieKeepAlive';

let keepAliveTaskData: {
    preSnapshot: Map<string, { value: string, expirationDate: number }>,
    syncList: Cookie[]
} | null = null;


async function performDataIntegrityCheck() {
    addLog('正在执行数据完整性检查...', 'info');
    try {
        const {
            [SYNC_LIST_STORAGE_KEY]: syncList = [],
            [STATS_STORAGE_KEY]: stats = {}
        } = await chrome.storage.local.get([SYNC_LIST_STORAGE_KEY, STATS_STORAGE_KEY]);

        if (!Array.isArray(syncList) || typeof stats !== 'object' || stats === null) {
            addLog('数据结构损坏，检查中止。', 'error');
            return;
        }

        const validCookieKeys = new Set(syncList.map(getCookieKey));
        let orphanCount = 0;
        const cleanedStats = { ...stats };

        for (const statKey in cleanedStats) {
            if (!validCookieKeys.has(statKey)) {
                delete cleanedStats[statKey];
                orphanCount++;
            }
        }

        if (orphanCount > 0) {
            await chrome.storage.local.set({ [STATS_STORAGE_KEY]: cleanedStats });
            addLog(`数据完整性检查完成。移除了 ${orphanCount} 个孤立的统计条目。`, 'success');
        } else {
            addLog('数据完整性检查完成。未发现孤立数据。', 'success');
        }
    } catch (e: any) {
        addLog(`数据完整性检查失败: ${e.message}`, 'error');
    }
}

chrome.runtime.onInstalled.addListener((details) => {
    addLog('插件已安装/更新。', 'info');
    if (details.reason === 'install' || details.reason === 'update') {
        performDataIntegrityCheck();
        // On install/update, check if a sync was pending
        chrome.storage.local.get(SYNC_QUEUE_STORAGE_KEY, (result) => {
            if (result[SYNC_QUEUE_STORAGE_KEY] && result[SYNC_QUEUE_STORAGE_KEY].length > 0) {
                addLog('检测到未完成的推送任务，将立即尝试同步。', 'info');
                triggerFullSync().catch(() => {}); // Suppress error, retry is scheduled inside
            }
        });
    }
    setupAlarms();
});
chrome.storage.onChanged.addListener((changes, area) => {
    if (area !== 'local') return;

    if (changes.syncList) {
        refreshMonitoredCookies();
    }
    // Re-setup alarms only if settings change. This prevents spamming on every cookie update.
    if (changes.syncSettings) {
        // Also refresh cookies in case the list was changed in a backup import that includes settings
        refreshMonitoredCookies();
        setupAlarms();
    }
});

const MIN_KEEPALIVE_FREQUENCY_MINUTES = 1;

async function setupAlarms() {
    const { syncSettings } = await chrome.storage.local.get('syncSettings');
    let keepAliveFrequency = syncSettings?.keepAliveFrequency || MIN_KEEPALIVE_FREQUENCY_MINUTES;

    if (keepAliveFrequency < MIN_KEEPALIVE_FREQUENCY_MINUTES) {
        addLog(`配置的保活频率 (${keepAliveFrequency}分钟) 低于最低限制 (${MIN_KEEPALIVE_FREQUENCY_MINUTES}分钟)，将使用最低限制。`, 'error');
        keepAliveFrequency = MIN_KEEPALIVE_FREQUENCY_MINUTES;
    }

    const existingAlarm = await chrome.alarms.get(KEEPALIVE_ALARM_NAME);

    // Only update the alarm if it doesn't exist or its period has changed. This makes the function idempotent.
    if (!existingAlarm || existingAlarm.periodInMinutes !== keepAliveFrequency) {
        await chrome.alarms.clear(KEEPALIVE_ALARM_NAME);
        chrome.alarms.create(KEEPALIVE_ALARM_NAME, {
            delayInMinutes: 1, // Run shortly after setup
            periodInMinutes: keepAliveFrequency
        });
        addLog(`保活任务已（重新）设置，频率: ${keepAliveFrequency} 分钟。`, 'info');
    }
}

chrome.alarms.onAlarm.addListener(async (alarm) => {
    if (alarm.name === KEEPALIVE_ALARM_NAME) {
        try {
            await handleKeepAlive();
        } catch (e: any) {
            await addLog(`[保活任务严重失败] ${e.message}`, 'error');
            keepAliveTaskData = null;
        }
    } else if (alarm.name === RETRY_ALARM_NAME) {
        try {
            await addLog('重试警报触发，尝试再次推送暂存的Cookie。', 'info');
            await triggerFullSync();
        } catch (e: any) {
            // Error is already logged and retry is rescheduled inside triggerFullSync
        }
    }
});

async function hasOffscreenDocument() {
    // @ts-ignore
    if (chrome.runtime.getContexts) {
        // @ts-ignore
        const contexts = await chrome.runtime.getContexts({ contextTypes: ['OFFSCREEN_DOCUMENT'] });
        return contexts.length > 0;
    }
    return false;
}

async function handleKeepAlive() {
    if (await hasOffscreenDocument()) {
        await addLog('[保活警告] 上一个保活任务仍在运行，本次任务跳过。', 'error');
        return;
    }
    
    const { [SYNC_LIST_STORAGE_KEY]: syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    if (syncList.length === 0) {
        await addLog('推送列表为空，跳过保活任务。', 'info');
        return;
    }
    
    const preSnapshot = new Map<string, { value: string, expirationDate: number }>();
    for (const cookie of syncList) {
        try {
            const foundCookie = await chrome.cookies.get({ url: `https://${cookie.domain.replace(/^\./, '')}/`, name: cookie.name });
            if (foundCookie) {
                preSnapshot.set(getCookieKey(foundCookie), { value: foundCookie.value, expirationDate: foundCookie.expirationDate || 0 });
            }
        } catch (e: any) {
            await addLog(`[快照警告] 获取Cookie '${cookie.name}' for '${cookie.domain}' 失败: ${e.message}`, 'error');
        }
    }
    await addLog(`创建了 ${preSnapshot.size} 个Cookie的保活前快照。`, 'info');

    keepAliveTaskData = { preSnapshot, syncList };

    const domainsToRefresh = new Set(syncList.map(c => c.domain.replace(/^\./, '')));
    const urlsToVisit = Array.from(domainsToRefresh).map(d => `https://${d}`);
    
    await addLog(`定时保活任务触发，准备为 ${urlsToVisit.length} 个域进行静默访问: ${Array.from(domainsToRefresh).join(', ')}`, 'info');
    
    const offscreenUrl = chrome.runtime.getURL(`offscreen/index.html?urls=${encodeURIComponent(JSON.stringify(urlsToVisit))}`);
    
    await chrome.offscreen.createDocument({
        url: offscreenUrl,
        reasons: [chrome.offscreen.Reason.DOM_PARSER],
        justification: 'Required to create iframes for silent cookie refresh.',
    });
}

async function handleKeepAlivePostTasks() {
    if (!keepAliveTaskData) {
        await addLog('[保活后处理警告] 任务数据丢失，无法对比快照。', 'error');
        return;
    }
    const { preSnapshot, syncList } = keepAliveTaskData;
    
    await addLog('静默访问完成，开始对比Cookie快照。', 'info');

    const successfulCookies: string[] = [];
    const failedCookies: { name: string; error: string }[] = [];
    const updatedCookies: Cookie[] = [];

    for (const oldCookieKey of preSnapshot.keys()) {
        const cookieInfo = syncList.find(c => getCookieKey(c) === oldCookieKey);
        if (!cookieInfo) continue;

        const cookieDisplayName = `${cookieInfo.name} (${cookieInfo.domain})`;
        let currentStatus: 'success' | 'failure' | 'no-change' = 'no-change';
        let errorMessage: string | undefined;

        try {
            const newCookie = await chrome.cookies.get({ url: `https://${cookieInfo.domain.replace(/^\./, '')}/`, name: cookieInfo.name });
            const oldSnapshot = preSnapshot.get(oldCookieKey)!;

            if (!newCookie) {
                errorMessage = `已失效或被移除`;
                currentStatus = 'failure';
            } else {
                const expirationChanged = (newCookie.expirationDate || 0) > (oldSnapshot.expirationDate || 0);
                const valueChanged = newCookie.value !== oldSnapshot.value;

                if (expirationChanged || valueChanged) {
                    currentStatus = 'success';
                    // Preserve metadata by merging new data into the existing cookie object
                    updatedCookies.push({ ...cookieInfo, ...newCookie });
                }
            }
        } catch (e: any) {
            errorMessage = e.message || '未知错误';
            currentStatus = 'failure';
        }
        
        if (currentStatus === 'success') {
            successfulCookies.push(cookieDisplayName);
            await updateCookieStat(oldCookieKey, 'success', 'keep-alive');
        } else if (currentStatus === 'failure') {
            const finalError = errorMessage || '未知错误';
            failedCookies.push({ name: cookieDisplayName, error: finalError });
            await updateCookieStat(oldCookieKey, 'failure', 'keep-alive', finalError);
        }
    }

    const totalProcessed = preSnapshot.size;
    const successCount = successfulCookies.length;
    const failureCount = failedCookies.length;
    const noChangeCount = totalProcessed - successCount - failureCount;

    let summaryMessage = `保活任务完成: 处理 ${totalProcessed}个, 成功 ${successCount}个, 失败 ${failureCount}个, 无变化 ${noChangeCount}个。`;
    
    if (successCount > 0) {
        summaryMessage += `\n  [成功]: ${successfulCookies.join('; ')}`;
    }
    if (failureCount > 0) {
        const failureDetails = failedCookies.map(f => `${f.name}: ${f.error}`).join('; ');
        summaryMessage += `\n  [失败]: ${failureDetails}`;
    }

    await addLog(summaryMessage, successCount > 0 || failureCount > 0 ? 'success' : 'info');

    if (updatedCookies.length > 0) {
        try {
            // Log that changes are queued, not immediately synced
            await addLog(`检测到 ${updatedCookies.length} 个Cookie因保活而更新，已加入推送队列。`, 'info');
            
            const syncMap = new Map(syncList.map((c: Cookie) => [getCookieKey(c), c]));
            updatedCookies.forEach((c: Cookie) => syncMap.set(getCookieKey(c), c));
            
            const updatedSyncList = Array.from(syncMap.values());
            await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedSyncList });
            
            // Use the unified scheduler
            // Use the scheduler without a specific trigger source for background tasks
            scheduleSync();
        } catch (e: any) {
            await addLog(`处理保活更新暂存时出错: ${e.message}`, 'error');
        }
    }

    keepAliveTaskData = null;
}

async function handleGetKeepAliveStats() {
    const { [STATS_STORAGE_KEY]: stats = {} } = await chrome.storage.local.get(STATS_STORAGE_KEY);

    for (const key in stats) {
        const cookieInfo = stats[key];
        const [name, domain, ] = key.split('|');
        try {
            const liveCookie = await chrome.cookies.get({ url: `https://${domain.replace(/^\./, '')}/`, name });
            if(liveCookie) {
                cookieInfo.expirationDate = liveCookie.expirationDate;
                cookieInfo.value = liveCookie.value;
            }
        } catch (e) {
            // Ignore if cookie not found
        }
    }
    return { stats };
}

function transformCookieForAPI(cookie: Cookie): object {
    return {
        domain: cookie.domain,
        name: cookie.name,
        value: cookie.value,
        path: cookie.path,
        http_only: cookie.httpOnly,
        secure: cookie.secure,
        same_site: cookie.sameSite || 'unspecified',
        is_sharable: cookie.isSharable || false,
        expires: cookie.expirationDate ? new Date(cookie.expirationDate * 1000).toISOString() : null,
        last_updated_from_extension_at: new Date().toISOString(),
    };
}

// =================================================================
// Centralized Statistics Update Function
// =================================================================
async function updateCookieStat(
    statKey: string,
    status: 'success' | 'failure',
    changeSource: 'keep-alive' | 'on-change',
    error?: string
) {
    try {
        const { [STATS_STORAGE_KEY]: stats = {} } = await chrome.storage.local.get(STATS_STORAGE_KEY);
        const timestamp = new Date().toISOString();

        if (!stats[statKey]) {
            stats[statKey] = { successCount: 0, failureCount: 0, history: [] };
        }
        const stat = stats[statKey];

        if (status === 'success') {
            stat.successCount++;
        } else {
            stat.failureCount++;
        }

        let intervalSeconds: number | undefined;
        if (stat.lastChangeTimestamp) {
            const lastTime = new Date(stat.lastChangeTimestamp).getTime();
            const now = new Date(timestamp).getTime();
            intervalSeconds = Math.round((now - lastTime) / 1000);
        }
        stat.lastChangeTimestamp = timestamp;

        const historyEntry: StatHistory = { status, timestamp, changeSource, intervalSeconds };
        if (status === 'failure' && error) {
            historyEntry.error = error;
        }

        stat.history.unshift(historyEntry);
        if (stat.history.length > 20) {
            stat.history.length = 20;
        }

        await chrome.storage.local.set({ [STATS_STORAGE_KEY]: stats });
    } catch (e: any) {
        await addLog(`更新统计数据失败 for ${statKey}: ${e.message}`, 'error');
    }
}


// =================================================================
// Data Backup & Restore Functions
// =================================================================
async function handleExportAllData() {
    const keysToExport = [
        LOGS_STORAGE_KEY,
        SYNC_LIST_STORAGE_KEY,
        STATS_STORAGE_KEY,
        'syncSettings'
    ];
    const data = await chrome.storage.local.get(keysToExport);
    return { data };
}

async function handleImportAllData(payload: { data: any }) {
    if (!payload || typeof payload.data !== 'object' || payload.data === null) {
        throw new Error("导入的数据格式无效。");
    }
    const { data } = payload;
    
    // Basic validation
    const requiredKeys = [SYNC_LIST_STORAGE_KEY, 'syncSettings'];
    for (const key of requiredKeys) {
        if (!(key in data)) {
            throw new Error(`导入数据缺少关键字段: ${key}`);
        }
    }

    // Clear existing data and set new data
    await chrome.storage.local.clear();
    await chrome.storage.local.set(data);
    addLog('数据已从备份文件成功导入。', 'success');

    // Trigger necessary re-initializations after import
    await refreshMonitoredCookies();
    await setupAlarms();
    await performDataIntegrityCheck();

    return { success: true };
}


async function handleGetUserSettings() {
    const { apiEndpoint, authToken } = await getDecryptedSettings();
    const url = new URL(apiEndpoint);
    url.pathname = '/api/v1/user/settings';

    const response = await fetch(url.toString(), {
        method: 'GET',
        headers: { 'x-api-key': authToken, 'Content-Type': 'application/json' }
    });

    const result = await response.json();

    if (!response.ok) {
        const errorMessage = result?.message || response.statusText;
        throw new Error(`获取用户设置失败: ${response.status} ${errorMessage}`);
    }

    if (result.code !== 200) {
        throw new Error(`获取用户设置接口响应异常: ${result.message}`);
    }
    
    return { data: result.data };
}

async function handleUpdateUserSettings(payload: { sharing_enabled: boolean }) {
    const { apiEndpoint, authToken } = await getDecryptedSettings();
    const url = new URL(apiEndpoint);
    url.pathname = '/api/v1/user/settings';

    const response = await fetch(url.toString(), {
        method: 'PUT',
        headers: { 'x-api-key': authToken, 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
    });

    if (!response.ok) {
        throw new Error(`更新用户设置失败: ${response.status} ${response.statusText}`);
    }
    const data = await response.json();
     if (data.code !== 200) {
        throw new Error(`更新用户设置接口响应异常: ${data.message}`);
    }
    addLog(`云端共享设置已更新为: ${payload.sharing_enabled ? '开启' : '关闭'}`, 'success');
    return data;
}


async function handleUpdateCookieRemark(payload: { cookieKey: string; remark: string }) {
    if (!payload || typeof payload.cookieKey !== 'string' || typeof payload.remark !== 'string') {
        throw new Error('无效的备注更新负载。');
    }
    const { cookieKey, remark } = payload;
    const { [SYNC_LIST_STORAGE_KEY]: syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };

    const cookieIndex = syncList.findIndex(c => getCookieKey(c) === cookieKey);
    if (cookieIndex === -1) {
        throw new Error('无法在推送列表中找到要更新备注的Cookie。');
    }

    // Update the remark
    syncList[cookieIndex].remark = remark;

    // Save the updated list. No sync is needed as this is a local-only feature.
    await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: syncList });

    addLog(`Cookie "${syncList[cookieIndex].name}" 的本地备注已更新。`, 'info');
    return { success: true, updatedCookie: syncList[cookieIndex] };
}


// Initial load
refreshMonitoredCookies();
addLog('后台服务已启动。', 'info');
