/**
 * CookieSyncer 路由配置
 * 统一管理浏览器插件的所有页面路由
 */

export interface RouteConfig {
  path: string;
  name: string;
  title: string;
  description: string;
  icon?: string;
  isExternal?: boolean;
}

// 定义所有路由
const ROUTES: Record<string, RouteConfig> = {
  WELCOME: {
    path: '/welcome/index.html',
    name: 'welcome',
    title: 'Cookie Syncer - 欢迎',
    description: '跨站点、跨设备的 Cookie/TOKEN 自动续期与推送助手',
    icon: '🎯'
  },
  POPUP: {
    path: '/app/index.html#/',
    name: 'popup',
    title: 'Cookie Syncer',
    description: '快速管理 Cookie 推送',
    icon: '🍪'
  },
  OPTIONS: {
    path: '/app/index.html#/options',
    name: 'options',
    title: 'Cookie Syncer - 设置',
    description: '配置域名和推送策略',
    icon: '⚙️'
  },
  BACKGROUND: {
    path: '/background/index.js',
    name: 'background',
    title: 'Cookie Syncer - 后台服务',
    description: '后台推送服务',
    isExternal: true
  }
} as const;

// 路由工具类
export class Router {
  /**
   * 获取完整的扩展URL
   */
  static getUrl(routeKey: keyof typeof ROUTES): string {
    const route = ROUTES[routeKey];
    return chrome.runtime.getURL(route.path);
  }

  /**
   * 获取路由配置
   */
  static getRoute(routeKey: keyof typeof ROUTES): RouteConfig {
    return ROUTES[routeKey];
  }

  /**
   * 打开指定路由页面
   */
  static open(routeKey: keyof typeof ROUTES, options?: chrome.tabs.CreateProperties): Promise<chrome.tabs.Tab> {
    return new Promise((resolve, reject) => {
      chrome.tabs.create({
        url: this.getUrl(routeKey),
        ...options
      }, (tab) => {
        if (chrome.runtime.lastError) {
          reject(new Error(chrome.runtime.lastError.message));
        } else {
          resolve(tab);
        }
      });
    });
  }

  /**
   * 获取当前页面的路由信息
   */
  static getCurrentRoute(): RouteConfig | null {
    const currentUrl = window.location.href;
    const extensionUrl = chrome.runtime.getURL('');
    
    if (!currentUrl.startsWith(extensionUrl)) {
      return null;
    }

    const relativePath = currentUrl.replace(extensionUrl, '');
    
    for (const route of Object.values(ROUTES)) {
      if (route.path === relativePath) {
        return route;
      }
    }
    
    return null;
  }

  /**
   * 检查是否是扩展内部页面
   */
  static isExtensionPage(url: string): boolean {
    return url.startsWith(chrome.runtime.getURL(''));
  }

  /**
   * 获取所有路由配置
   */
  static getAllRoutes(): RouteConfig[] {
    return Object.values(ROUTES);
  }

  /**
   * 根据路径查找路由
   */
  static findRouteByPath(path: string): RouteConfig | null {
    for (const route of Object.values(ROUTES)) {
      if (route.path === path) {
        return route;
      }
    }
    return null;
  }
}

// 路由导航助手
export class NavigationHelper {
  /**
   * 导航到欢迎页面（安装后自动调用）
   */
  static navigateToWelcome(): Promise<chrome.tabs.Tab> {
    return Router.open('WELCOME', {
      active: true
    });
  }

  /**
   * 导航到设置页面
   */
  static navigateToOptions(): Promise<chrome.tabs.Tab> {
    return Router.open('OPTIONS', {
      active: true
    });
  }

  /**
   * 导航到指定页面
   */
  static navigateTo(routeKey: keyof typeof ROUTES): Promise<chrome.tabs.Tab> {
    return Router.open(routeKey, {
      active: true
    });
  }

  /**
   * 在当前页面内导航（适用于单页应用）
   */
  static navigateWithinApp(path: string): void {
    if (typeof window !== 'undefined') {
      window.location.href = path;
    }
  }

  /**
   * 获取页面间的导航链接
   */
  static getNavigationLinks(): Array<{
    name: string;
    title: string;
    description: string;
    icon: string;
    url: string;
  }> {
    return Object.entries(ROUTES)
      .filter(([key, route]) => !route.isExternal)
      .map(([key, route]) => ({
        name: route.name,
        title: route.title,
        description: route.description,
        icon: route.icon || '🔗',
        url: Router.getUrl(key as keyof typeof ROUTES)
      }));
  }
}

// 默认导出
export default Router;
