/**
 * CookieSyncer - 最终架构后台服务 V4 (带智能分组)
 */

// 扩展安装事件
chrome.runtime.onInstalled.addListener((details) => {
    console.log('[CookieSyncer] 扩展已安装/更新:', details.reason);
});

// Chrome扩展消息监听器
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    const { action } = message;
    if (action === 'getCurrentTabCookies') {
        handleGetCurrentTabCookies()
            .then(sendResponse)
            .catch(error => {
                console.error('[CookieSyncer] 获取Cookie时发生顶层错误:', error);
                sendResponse({ success: false, error: error.message });
            });
        return true; // 异步响应
    }
    return false;
});

/**
 * 使用 Debugger API 监听网络请求，收集所有相关URL，然后获取这些URL的Cookie，并进行智能分组。
 */
async function handleGetCurrentTabCookies() {
    const [currentTab] = await chrome.tabs.query({ active: true, currentWindow: true });

    if (!currentTab || !currentTab.id) {
        throw new Error('没有找到活动的标签页。');
    }
    
    const tabUrl = currentTab.url;
    if (tabUrl && (tabUrl.startsWith('chrome://') || tabUrl.startsWith('about:'))) {
        return { success: true, groupedCookies: {}, domain: '特殊页面' };
    }

    const debuggee = { tabId: currentTab.id };
    const protocolVersion = "1.3";
    const requestedUrls = new Set<string>();
    if (tabUrl) {
        requestedUrls.add(tabUrl); // 初始加入当前页URL
    }

    const onDebuggerEvent = (source: chrome.debugger.Debuggee, method: string, params: any) => {
        if (source.tabId !== currentTab.id) return;
        if (method === "Network.requestWillBeSent") {
            requestedUrls.add(params.request.url);
        }
    };
    
    try {
        await new Promise<void>((resolve, reject) => {
            chrome.debugger.attach(debuggee, protocolVersion, () => {
                if (chrome.runtime.lastError) {
                    reject(new Error(chrome.runtime.lastError.message || '无法附加调试器。'));
                } else {
                    chrome.debugger.sendCommand(debuggee, "Network.enable", {}, () => {
                        if (chrome.runtime.lastError) reject(new Error('开启网络监听失败。'));
                        else resolve();
                    });
                }
            });
        });
        
        chrome.debugger.onEvent.addListener(onDebuggerEvent);
        await chrome.tabs.reload(currentTab.id);
        await new Promise(resolve => setTimeout(resolve, 2000));

        chrome.debugger.onEvent.removeListener(onDebuggerEvent);
        
        const cookiePromises = Array.from(requestedUrls).map(url => 
            chrome.cookies.getAll({ url }).catch(() => [])
        );
        const settledCookies = await Promise.all(cookiePromises);
        const allCookies = settledCookies.flat();
        
        const uniqueCookies = Array.from(new Map(allCookies.map(cookie => [cookie.name + cookie.domain + cookie.path, cookie])).values());
        
        // --- 智能分组逻辑 ---
        const groupedCookies = uniqueCookies.reduce((acc, cookie) => {
            const groupKey = getRegistrableDomain(cookie.domain);
            if (!acc[groupKey]) {
                acc[groupKey] = [];
            }
            acc[groupKey].push(cookie);
            return acc;
        }, {} as { [key: string]: chrome.cookies.Cookie[] });

        return {
            success: true,
            groupedCookies: groupedCookies,
            domain: new URL(tabUrl!).hostname
        };

    } finally {
        await new Promise<void>(resolve => {
            chrome.debugger.onEvent.removeListener(onDebuggerEvent);
            chrome.debugger.detach(debuggee, () => resolve());
        });
    }
}

/**
 * 获取一个域名的可注册域 (eTLD+1)
 * 例如, 'www.bilibili.com' -> 'bilibili.com'
 * 例如, 'a.b.github.io' -> 'b.github.io'
 * 这是一个简化的实现，对于复杂的eTLD列表可能不完美，但能处理绝大多数情况。
 */
function getRegistrableDomain(domain: string): string {
    if (domain.startsWith('.')) {
        domain = domain.substring(1);
    }
    const parts = domain.split('.');
    if (parts.length <= 2) {
        return domain;
    }
    // 简单的处理 com.cn, org.cn 等情况
    const twoLevelTlds = new Set(['com.cn', 'org.cn', 'net.cn', 'gov.cn']);
    const lastTwo = parts.slice(-2).join('.');
    if (twoLevelTlds.has(lastTwo) && parts.length > 3) {
        return parts.slice(-3).join('.');
    }
    return lastTwo;
}

console.log('[CookieSyncer] 最终架构V4后台服务已加载 (带智能分组)。');
