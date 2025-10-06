// This script runs in the offscreen document.
// It listens for messages from the service worker to create iframes.
chrome.runtime.onMessage.addListener(async (request) => {
    if (request.action === 'keepAlive') {
        await keepAlive(request.urls);
        chrome.runtime.sendMessage({ action: 'keepAliveDone' });
    }
});

async function keepAlive(urls) {
    if (!urls || urls.length === 0) {
        return;
    }

    console.log('[Offscreen] Starting keep-alive for URLs:', urls);

    const iframes = urls.map(url => {
        const iframe = document.createElement('iframe');
        iframe.src = url;
        document.body.appendChild(iframe);
        return iframe;
    });

    // Wait for a reasonable amount of time for cookies to be refreshed.
    // This is a heuristic. 30 seconds should be more than enough for most sites.
    await new Promise(resolve => setTimeout(resolve, 30000));

    console.log('[Offscreen] Keep-alive task finished. Cleaning up iframes.');
    iframes.forEach(iframe => iframe.remove());
}
