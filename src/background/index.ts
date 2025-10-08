/**
 * CookieSyncer - "Automatic & Non-Intrusive" Backend (V13 - Final Keep-Alive)
 */

import CryptoJS from 'crypto-js';

type Cookie = chrome.cookies.Cookie;

const LOGS_STORAGE_KEY = 'cookieSyncerLogs';
const SYNC_LIST_STORAGE_KEY = 'syncList';
const STATS_STORAGE_KEY = 'keepAliveStats';
const MAX_LOGS = 100;

interface KeepAliveStat {
    successCount: number;
    failureCount: number;
    history: {
        status: 'success' | 'failure' | 'no-change';
        timestamp: string;
    }[];
    // Dynamically added fields
    expirationDate?: number;
    value?: string;
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

    switch (action) {
        case 'getCookiesForCurrentTab':
            isAsync = true;
            handleGetCookiesForCurrentTab()
                .then(response => sendResponse({ success: true, ...response }))
                .catch(error => {
                    addLog(`分析失败: ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        case 'syncSingleCookie':
        case 'syncAllCookiesForDomain':
            isAsync = true;
            handleSyncCookies(payload)
                .then(response => sendResponse({ success: true, ...response }))
                .catch(error => {
                    addLog(`同步失败: ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        case 'getCookiesForDomain':
            isAsync = true;
            handleGetCookiesForDomain(payload)
                .then(response => sendResponse({ success: true, ...response }))
                .catch(error => {
                    addLog(`获取域Cookie失败: ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        case 'removeCookieFromSyncList':
            isAsync = true;
            handleRemoveCookieFromSync(payload)
                .then(response => sendResponse({ success: true, ...response }))
                .catch(error => {
                    addLog(`从同步列表移除Cookie失败: ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        case 'removeDomainFromSyncList':
            isAsync = true;
            handleRemoveDomainFromSync(payload)
                .then(response => sendResponse({ success: true, ...response }))
                .catch(error => {
                    addLog(`从同步列表移除域名失败: ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        case 'testApiConnection':
            isAsync = true;
            handleTestApiConnection()
                .then(response => sendResponse({ success: true, ...response }))
                .catch(error => {
                    addLog(`API连接测试失败: ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        case 'manualSync':
            isAsync = true;
            handleManualSync()
                .then(response => sendResponse({ success: true, ...response }))
                .catch(error => {
                    addLog(`手动同步失败: ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        case 'getLogs':
            isAsync = true;
            chrome.storage.local.get(LOGS_STORAGE_KEY, (result) => {
                sendResponse({ success: true, logs: result[LOGS_STORAGE_KEY] || [] });
            });
            break;
        case 'clearLogs':
            isAsync = true;
            chrome.storage.local.remove(LOGS_STORAGE_KEY, () => {
                addLog('日志已清空。', 'info');
                sendResponse({ success: true });
            });
            break;
        case 'addLog':
            if (payload) addLog(payload.message, payload.type);
            break;
        case 'getKeepAliveStats':
            isAsync = true;
            handleGetKeepAliveStats()
                .then(stats => sendResponse({ success: true, stats }))
                .catch(error => {
                    addLog(`获取统计数据失败: ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        case 'keepAliveTaskFinished':
            isAsync = true;
            handleKeepAlivePostTasks()
                .then(() => sendResponse({ success: true }))
                .catch(error => {
                    addLog(`[保活后处理失败] ${error.message}`, 'error');
                    sendResponse({ success: false, error: error.message });
                });
            break;
        default:
            break;
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
    if (!payload || (!payload.cookie && !payload.cookies)) throw new Error('无效的同步请求负载。');
    const newCookies = payload.cookies || (payload.cookie ? [payload.cookie] : []);
    if (newCookies.length === 0) return { message: "没有需要同步的Cookie。" };
    
    const result = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY);
    const currentSyncList: Cookie[] = result[SYNC_LIST_STORAGE_KEY] || [];
    const syncMap = new Map(currentSyncList.map(c => [getCookieKey(c), c]));

    let addedCount = 0;
    newCookies.forEach(c => {
        const key = getCookieKey(c);
        if (!syncMap.has(key)) addedCount++;
        syncMap.set(key, c);
    });

    const updatedSyncList = Array.from(syncMap.values());
    await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedSyncList });
    const logMessage = `同步列表更新: 新增 ${addedCount}, 总计 ${updatedSyncList.length}`;
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
    const logMessage = `从同步列表移除: ${cookieToRemove.name}. 总计: ${updatedSyncList.length}`;
    addLog(logMessage, 'info');
    return { removed: true, total: updatedSyncList.length, message: logMessage };
}

async function handleRemoveDomainFromSync(payload: { domain: string }) {
    if (!payload?.domain) throw new Error('无效的域名以进行移除。');
    const { domain: domainToRemove } = payload;
    const { [SYNC_LIST_STORAGE_KEY]: currentSyncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    const updatedSyncList = currentSyncList.filter(c => getRegistrableDomain(c.domain) !== domainToRemove);
    await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedSyncList });
    const logMessage = `从同步列表移除域名: ${domainToRemove}.`;
    addLog(logMessage, 'info');
    return { removed: true, total: updatedSyncList.length, message: logMessage };
}

const SECRET_KEY = 'cookie-syncer-secret-key';

async function getDecryptedSettings() {
    const { syncSettings } = await chrome.storage.local.get('syncSettings');
    if (!syncSettings?.apiEndpoint || !syncSettings?.authToken) throw new Error('API端点或Auth Token未设置。');
    try {
        const bytes = CryptoJS.AES.decrypt(syncSettings.authToken, SECRET_KEY);
        const authToken = bytes.toString(CryptoJS.enc.Utf8);
        if (!authToken) throw new Error("解密后的Token为空。");
        // Explicitly return a new object with the decrypted token.
        // This avoids the bug where the encrypted token overwrites the decrypted one.
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
        headers: { 'Authorization': `Bearer ${authToken}`, 'Content-Type': 'application/json' }
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

async function handleManualSync() {
    await addLog('手动同步开始...', 'info');
    const { apiEndpoint, authToken } = await getDecryptedSettings();
    const { [SYNC_LIST_STORAGE_KEY]: localSyncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };

    if (localSyncList.length === 0) {
        const message = "本地没有需要同步的Cookie，任务跳过。";
        await addLog(message, 'info');
        return { message, total: 0 };
    }

    const syncUrl = new URL(apiEndpoint);
    syncUrl.pathname = '/api/v1/sync';

    const response = await fetch(syncUrl.toString(), {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${authToken}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(localSyncList)
    });

    if (!response.ok) {
        const errorBody = await response.text();
        throw new Error(`推送到云端失败: ${response.status} - ${errorBody}`);
    }
    
    const responseData = await response.json();

    if (responseData.code !== 200 || !Array.isArray(responseData.data)) {
        throw new Error(`同步响应格式错误: ${JSON.stringify(responseData)}`);
    }
    
    // Save the authoritative list from the server back to local storage
    await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: responseData.data });

    const message = `手动同步成功！云端现在有 ${responseData.data.length} 个Cookie。`;
    addLog(message, 'success');
    return { message, total: responseData.data.length };
}

// --- V6.0 & V7.0 ---
let monitoredCookies = new Set<string>();

async function refreshMonitoredCookies() {
    const { syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    monitoredCookies = new Set(syncList.map(c => getCookieKey(c)));
}

chrome.cookies.onChanged.addListener(async (changeInfo) => {
    if (changeInfo.cause !== 'explicit' || changeInfo.removed) return;
    const key = getCookieKey(changeInfo.cookie);
    if (monitoredCookies.has(key)) {
        addLog(`检测到受监控的Cookie变更: ${key}，触发自动推送。`, 'info');
        try {
            const { apiEndpoint, authToken } = await getDecryptedSettings();
            const { [SYNC_LIST_STORAGE_KEY]: syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
            
            const updatedList = syncList.map((c: Cookie) => getCookieKey(c) === key ? changeInfo.cookie : c);
            if (!updatedList.some(c => getCookieKey(c) === key)) {
                updatedList.push(changeInfo.cookie);
            }

            await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedList });

            const syncUrl = new URL(apiEndpoint);
            syncUrl.pathname = '/api/v1/sync';
            const response = await fetch(syncUrl.toString(), {
                method: 'POST',
                headers: { 'Authorization': `Bearer ${authToken}`, 'Content-Type': 'application/json' },
                body: JSON.stringify(updatedList)
            });

            if (!response.ok) {
                const errorBody = await response.text();
                throw new Error(`自动推送失败: ${response.status} - ${errorBody}`);
            }
            addLog(`自动推送 ${key} 成功。`, 'success');
        } catch (e: any) {
            addLog(`自动推送失败: ${e.message}`, 'error');
        }
    }
});

const KEEPALIVE_ALARM_NAME = 'cookieKeepAlive';

// Temporary storage for keep-alive task data
let keepAliveTaskData: {
    preSnapshot: Map<string, { value: string, expirationDate: number }>,
    syncList: Cookie[]
} | null = null;


chrome.runtime.onInstalled.addListener(() => {
    addLog('插件已安装/更新。', 'info');
    setupAlarms();
});
chrome.storage.onChanged.addListener((changes, area) => {
    if (area === 'local' && (changes.syncList || changes.syncSettings)) {
        refreshMonitoredCookies();
        setupAlarms();
    }
});

async function setupAlarms(){
    const { syncSettings } = await chrome.storage.local.get('syncSettings');
    const keepAliveFrequency = syncSettings?.keepAliveFrequency || 1;
    await chrome.alarms.clear(KEEPALIVE_ALARM_NAME);
    chrome.alarms.create(KEEPALIVE_ALARM_NAME, {
        delayInMinutes: 1,
        periodInMinutes: keepAliveFrequency
    });
    addLog(`保活任务已设置，频率: ${keepAliveFrequency} 分钟。`, 'info');
}

chrome.alarms.onAlarm.addListener(async (alarm) => {
    if (alarm.name === KEEPALIVE_ALARM_NAME) {
        try {
            await handleKeepAlive();
        } catch (e: any) {
            await addLog(`[保活任务严重失败] ${e.message}`, 'error');
            // Clean up task data on failure
            keepAliveTaskData = null;
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
        await addLog('同步列表为空，跳过保活任务。', 'info');
        return;
    }
    
    // --- Step 1: Create Pre-Snapshot in Service Worker ---
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

    // Store data for post-task processing
    keepAliveTaskData = { preSnapshot, syncList };

    // --- Step 2: Start Offscreen Document for iframe loading ---
    const domainsToRefresh = new Set(syncList.map(c => getRegistrableDomain(c.domain)));
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
    const { [STATS_STORAGE_KEY]: stats = {} } = await chrome.storage.local.get(STATS_STORAGE_KEY);
    const timestamp = new Date().toISOString();

    await addLog('静默访问完成，开始对比Cookie快照并更新统计数据。', 'info');

    for (const oldCookieKey of preSnapshot.keys()) {
        const cookieInfo = syncList.find(c => getCookieKey(c) === oldCookieKey);
        if (!cookieInfo) continue;
        
        const statKey = getCookieKey(cookieInfo);
        if (!stats[statKey]) {
            stats[statKey] = { successCount: 0, failureCount: 0, history: [] };
        }
        let currentStatus: 'success' | 'failure' | 'no-change' = 'no-change';

        try {
            const newCookie = await chrome.cookies.get({ url: `https://${cookieInfo.domain.replace(/^\./, '')}/`, name: cookieInfo.name });
            const oldSnapshot = preSnapshot.get(oldCookieKey)!;

            if (!newCookie) {
                await addLog(`[保活警告] Cookie '${cookieInfo.name}' 在 ${cookieInfo.domain} 上已失效或被移除。`, 'error');
                stats[statKey].failureCount++;
                currentStatus = 'failure';
            } else {
                if (newCookie.expirationDate && oldSnapshot.expirationDate && newCookie.expirationDate > oldSnapshot.expirationDate) {
                    stats[statKey].successCount++;
                    currentStatus = 'success';
                    if (newCookie.value !== oldSnapshot.value) {
                        await addLog(`[保活更新] Cookie '${newCookie.name}' 的值已更新，有效期已延长。`, 'success');
                    } else {
                        await addLog(`[保活成功] Cookie '${newCookie.name}' 的有效期已延长。`, 'success');
                    }
                }
            }
        } catch (e: any) {
            await addLog(`[快照对比警告] 对比Cookie '${cookieInfo.name}' for '${cookieInfo.domain}' 失败: ${e.message}`, 'error');
            stats[statKey].failureCount++;
            currentStatus = 'failure';
        }
        
        stats[statKey].history.unshift({ status: currentStatus, timestamp });
        if (stats[statKey].history.length > 20) { // Limit history size
            stats[statKey].history.length = 20;
        }
    }

    await chrome.storage.local.set({ [STATS_STORAGE_KEY]: stats });

    // Clean up
    keepAliveTaskData = null;
    await addLog('保活任务与日志记录全部完成。', 'info');
}

async function handleGetKeepAliveStats() {
    const { [STATS_STORAGE_KEY]: stats = {} } = await chrome.storage.local.get(STATS_STORAGE_KEY);

    // Enhance stats with live expiration dates
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
            // Ignore if cookie not found, it might have been deleted.
        }
    }
    return stats;
}

// Initial load
refreshMonitoredCookies();
addLog('后台服务已启动。', 'info');
