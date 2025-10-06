/**
 * CookieSyncer - "Automatic & Non-Intrusive" Backend (V11 - With Logging)
 */

import CryptoJS from 'crypto-js';

type Cookie = chrome.cookies.Cookie;

const LOGS_STORAGE_KEY = 'cookieSyncerLogs';
const SYNC_LIST_STORAGE_KEY = 'syncList';
const MAX_LOGS = 100;

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
        case 'keepAliveDone':
            addLog('静默保活任务完成。', 'info');
            chrome.offscreen.closeDocument();
            break;
        default:
            // Explicitly do nothing for unknown actions to avoid errors.
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
    const uniqueCookies = Array.from(new Map(relevantCookies.map(c => [c.name + c.domain + c.path, c])).values());
    
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
    const protectedSchemes = ['chrome://', 'about:', 'edge://'];
    return protectedSchemes.some(scheme => url.startsWith(scheme));
}

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
    const syncMap = new Map(currentSyncList.map(c => [c.name + c.domain + c.path, c]));

    let addedCount = 0;
    newCookies.forEach(c => {
        const key = c.name + c.domain + c.path;
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
    const keyToRemove = cookieToRemove.name + cookieToRemove.domain + cookieToRemove.path;
    const updatedSyncList = currentSyncList.filter(c => (c.name + c.domain + c.path) !== keyToRemove);
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
        return { apiEndpoint: syncSettings.apiEndpoint, authToken };
    } catch (e) {
        throw new Error('Auth Token解密失败，请检查设置。');
    }
}

async function handleTestApiConnection() {
    const { apiEndpoint, authToken } = await getDecryptedSettings();
    const response = await fetch(apiEndpoint, {
        method: 'GET',
        headers: { 'Authorization': `Bearer ${authToken}`, 'Content-Type': 'application/json' }
    });
    if (!response.ok) throw new Error(`服务器响应错误: ${response.status} ${response.statusText}`);
    addLog('API连接测试成功。', 'success');
    return { connected: true };
}

async function handleManualSync() {
    await addLog('手动同步开始...', 'info');
    const { apiEndpoint, authToken } = await getDecryptedSettings();
    
    const response = await fetch(apiEndpoint, { method: 'GET', headers: { 'Authorization': `Bearer ${authToken}` } });
    if (!response.ok) throw new Error(`从云端拉取数据失败: ${response.status}`);
    const remoteData = await response.json();
    const remoteSyncList: Cookie[] = remoteData?.data?.syncList || [];

    const { [SYNC_LIST_STORAGE_KEY]: localSyncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };

    const mergedMap = new Map(localSyncList.map((c: Cookie) => [c.name + c.domain + c.path, c]));
    remoteSyncList.forEach(c => { mergedMap.set(c.name + c.domain + c.path, c); });
    const finalSyncList = Array.from(mergedMap.values());

    const postResponse = await fetch(apiEndpoint, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${authToken}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ syncList: finalSyncList })
    });
    if (!postResponse.ok) throw new Error(`推送到云端失败: ${postResponse.status}`);
    
    await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: finalSyncList });
    const message = `手动同步成功！本地与云端共合并了 ${finalSyncList.length} 个Cookie。`;
    addLog(message, 'success');
    return { message, total: finalSyncList.length };
}

// --- V6.0 & V7.0 Combined ---
let monitoredCookies = new Set<string>();

async function refreshMonitoredCookies() {
    const { syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    monitoredCookies = new Set(syncList.map(c => c.name + c.domain + c.path));
    console.log(`[CookieSyncer] Monitored cookies refreshed. Total: ${monitoredCookies.size}`);
}

chrome.cookies.onChanged.addListener(async (changeInfo) => {
    if (changeInfo.cause !== 'explicit' || changeInfo.removed) return;
    const key = changeInfo.cookie.name + changeInfo.cookie.domain + changeInfo.cookie.path;
    if (monitoredCookies.has(key)) {
        addLog(`检测到受监控的Cookie变更: ${key}，触发自动推送。`, 'info');
        try {
            const { apiEndpoint, authToken } = await getDecryptedSettings();
            const { [SYNC_LIST_STORAGE_KEY]: syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
            
            const updatedList = syncList.map((c: Cookie) => (c.name + c.domain + c.path) === key ? changeInfo.cookie : c);
            if (!updatedList.some(c => (c.name + c.domain + c.path) === key)) {
                updatedList.push(changeInfo.cookie);
            }

            await chrome.storage.local.set({ [SYNC_LIST_STORAGE_KEY]: updatedList });

            await fetch(apiEndpoint, {
                method: 'POST',
                headers: { 'Authorization': `Bearer ${authToken}`, 'Content-Type': 'application/json' },
                body: JSON.stringify({ syncList: updatedList })
            });
            addLog(`自动推送 ${key} 成功。`, 'success');
        } catch (e: any) {
            addLog(`自动推送失败: ${e.message}`, 'error');
        }
    }
});

const KEEPALIVE_ALARM_NAME = 'cookieKeepAlive';

chrome.runtime.onInstalled.addListener(() => {
    addLog('插件已安装/更新。', 'info');
    chrome.alarms.get(KEEPALIVE_ALARM_NAME, (alarm) => {
        if (!alarm) {
            chrome.alarms.create(KEEPALIVE_ALARM_NAME, { periodInMinutes: 60 });
            addLog('定时保活任务已创建。', 'info');
        }
    });
});

chrome.alarms.onAlarm.addListener(async (alarm) => {
    if (alarm.name === KEEPALIVE_ALARM_NAME) {
        addLog('定时保活任务触发。', 'info');
        await handleKeepAlive();
    }
});

async function handleKeepAlive() {
    const { syncList = [] } = await chrome.storage.local.get(SYNC_LIST_STORAGE_KEY) as { syncList: Cookie[] };
    if(syncList.length === 0) {
        addLog('同步列表为空，跳过保活任务。', 'info');
        return;
    }

    const domainsToRefresh = new Set(syncList.map(c => getRegistrableDomain(c.domain)));
    const urlsToVisit = Array.from(domainsToRefresh).map(d => `https://${d}`);

    if (await hasOffscreenDocument()) {
        chrome.runtime.sendMessage({ action: 'keepAlive', urls: urlsToVisit });
    } else {
        await chrome.offscreen.createDocument({
            url: 'offscreen.html',
            reasons: [chrome.offscreen.Reason.DOM_PARSER],
            justification: 'Required to create iframes for silent cookie refresh.',
        });
        setTimeout(() => {
             chrome.runtime.sendMessage({ action: 'keepAlive', urls: urlsToVisit });
        }, 1000);
    }
}

async function hasOffscreenDocument() {
    // @ts-ignore
    if (chrome.runtime.getContexts) {
        // @ts-ignore
        const contexts = await chrome.runtime.getContexts({ contextTypes: ['OFFSCREEN_DOCUMENT'] });
        return contexts.length > 0;
    } else {
        const views = chrome.extension.getViews({ type: 'OFFSCREEN_DOCUMENT' });
        return views.length > 0;
    }
}

// Initial load
refreshMonitoredCookies();
addLog('后台服务已启动。', 'info');
