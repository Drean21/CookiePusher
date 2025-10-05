/**
 * CookieSyncer - "Automatic & Non-Intrusive" Backend (V9 - The Final, Final One)
 */

type Cookie = chrome.cookies.Cookie;

// Main message listener
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    const { action, payload } = message;

    if (action === 'getCookiesForCurrentTab') {
        handleGetCookiesForCurrentTab()
            .then(response => sendResponse({ success: true, ...response }))
            .catch(error => {
                console.error('[CookieSyncer] V9 Analysis Failed:', error);
                sendResponse({ success: false, error: error.message });
            });
        return true; // Indicates async response
    }

    if (action === 'syncSingleCookie' || action === 'syncAllCookiesForDomain') {
        handleSyncCookies(payload)
            .then(response => sendResponse({ success: true, ...response }))
            .catch(error => {
                console.error('[CookieSyncer] Sync failed:', error);
                sendResponse({ success: false, error: error.message });
            });
        return true;
    }

    return false;
});

/**
 * The definitive, non-intrusive, and correct way to get all relevant cookies.
 * This function is now called automatically by the popup on open.
 */
async function handleGetCookiesForCurrentTab() {
    const [currentTab] = await chrome.tabs.query({ active: true, currentWindow: true });

    if (!currentTab?.id || !currentTab.url) {
        throw new Error('没有找到活动的标签页或标签页没有URL。');
    }
    
    if (currentTab.url.startsWith('chrome://') || currentTab.url.startsWith('about:')) {
        return { groupedCookies: {}, domain: '特殊页面' };
    }

    // --- The Correct Way: Get all frame domains directly via scripting ---
    const injectionResults = await chrome.scripting.executeScript({
        target: { tabId: currentTab.id, allFrames: true },
        func: () => document.domain,
    }).catch(error => {
        console.warn(`[CookieSyncer] Scripting injection failed, likely due to a protected page. Falling back to main domain. Error: ${error.message}`);
        // Fallback to only the main domain if injection fails
        return [{ result: new URL(currentTab.url!).hostname }];
    });

    const domains = new Set<string>();
    if (injectionResults) {
        injectionResults.forEach(item => {
            if (item.result) {
                domains.add(item.result);
            }
        });
    }
    domains.add(new URL(currentTab.url).hostname);

    // --- The Correct Logic: Global Scan + Precise Filtering ---
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

/**
 * Filters a list of all browser cookies down to only those that are "visible"
 * to a given set of domains.
 */
function filterCookiesByDomains(allCookies: Cookie[], domains: Set<string>): Cookie[] {
    const relevantCookies: Cookie[] = [];
    const pageDomains = Array.from(domains).map(d => d.toLowerCase());

    for (const cookie of allCookies) {
        const cookieDomain = cookie.domain.startsWith('.') ? cookie.domain.substring(1).toLowerCase() : cookie.domain.toLowerCase();
        
        const isVisible = pageDomains.some(pageDomain =>
            pageDomain === cookieDomain || pageDomain.endsWith(`.${cookieDomain}`)
        );

        if (isVisible) {
            relevantCookies.push(cookie);
        }
    }
    return relevantCookies;
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
    const SYNC_LIST_STORAGE_KEY = 'syncList';
    if (!payload || (!payload.cookie && !payload.cookies)) {
        throw new Error('无效的同步请求负载。');
    }
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

    console.log(`[CookieSyncer] 同步列表已更新。新增 ${addedCount} 个, 总计 ${updatedSyncList.length} 个。`);
    return { added: addedCount, total: updatedSyncList.length };
}

console.log('[CookieSyncer] Automatic & Non-Intrusive Backend (V9) loaded.');
