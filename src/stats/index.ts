import { createApp } from 'vue';
import StatsPage from './StatsPage.vue';

// A simple global CSS reset and basic styles
const style = document.createElement('style');
style.textContent = `
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    margin: 0;
    background-color: #f0f2f5;
    color: #333;
  }
`;
document.head.appendChild(style);


createApp(StatsPage).mount('#app');
