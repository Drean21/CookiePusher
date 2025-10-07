(async () => {
    const params = new URLSearchParams(location.search);
    const urlsToVisit = JSON.parse(params.get('urls'));

    // Simplified message sender
    const sendMessage = (action, payload) => chrome.runtime.sendMessage({ action, payload });

    await sendMessage('addLog', { message: `Offscreen document started for silent access to ${urlsToVisit.length} domains...`, type: 'info' });

    // Function to load a single iframe
    const loadIframe = (url) => new Promise((resolve) => {
        const iframe = document.createElement('iframe');
        iframe.src = url;

        const timeoutId = setTimeout(() => {
            iframe.remove();
            resolve({ status: 'timeout', url });
        }, 30000); // 30 seconds timeout

        iframe.onload = () => {
            clearTimeout(timeoutId);
            iframe.remove();
            resolve({ status: 'loaded', url });
        };

        iframe.onerror = () => {
            clearTimeout(timeoutId);
            iframe.remove();
            resolve({ status: 'error', url });
        };
        document.body.appendChild(iframe);
    });

    // Process all URLs
    const results = await Promise.allSettled(urlsToVisit.map(loadIframe));

    // Report failures back to the service worker
    const failedLoads = results
        .filter(r => r.status === 'fulfilled' && r.value.status !== 'loaded')
        .map(r => r.value.url);

    if (failedLoads.length > 0) {
        await sendMessage('addLog', {
            message: `[Keep-Alive Warning] The following domains failed to load or timed out in iframe: ${failedLoads.join(', ')}. This might be due to the sites' security policies (e.g., X-Frame-Options).`,
            type: 'error'
        });
    }

    // Notify the background script that the task is complete
    await sendMessage('keepAliveTaskFinished');

    // Close the offscreen document
    window.close();
})();
