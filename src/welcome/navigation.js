// 自动重定向到正确的欢迎页面
if (window.location.pathname.endsWith('/welcome.html')) {
    window.location.href = './welcome/index.html';
}

// 导入路由配置
import { NavigationHelper } from '../core/routes.js';

// 全局导航函数
window.navigateTo = function(route) {
    if (route === 'popup') {
        // 打开弹窗
        chrome.action.openPopup?.() || 
        window.open(NavigationHelper.getNavigationLinks().find(link => link.name === 'popup')?.url || '#', '_blank');
    } else if (route === 'options') {
        // 打开设置页面
        chrome.runtime.openOptionsPage?.() || 
        window.open(NavigationHelper.getNavigationLinks().find(link => link.name === 'options')?.url || '#', '_blank');
    }
};

// 添加事件监听器
document.addEventListener('DOMContentLoaded', function() {
    // 为所有带有 data-route 属性的链接添加点击事件
    const links = document.querySelectorAll('a[data-route]');
    links.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const route = this.getAttribute('data-route');
            navigateTo(route);
        });
    });
});