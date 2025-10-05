/**
 * A promise-based wrapper for chrome.runtime.sendMessage.
 */
export function sendMessage(action: string, payload?: any): Promise<any> {
    return new Promise((resolve, reject) => {
        chrome.runtime.sendMessage({ action, payload }, (response) => {
            if (chrome.runtime.lastError) {
                // If an error occurred, reject the promise
                reject(chrome.runtime.lastError);
            } else {
                // Otherwise, resolve with the response
                resolve(response);
            }
        });
    });
}
