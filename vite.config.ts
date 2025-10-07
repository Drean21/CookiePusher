import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import copy from 'rollup-plugin-copy'

export default defineConfig(({ mode }) => ({
  plugins: [
    vue(),
    copy({
      targets: [
        { src: 'manifest.json', dest: 'dist' },
        { src: 'icons', dest: 'dist' }
      ],
      hook: 'writeBundle' // 确保在所有文件写入后再执行复制
    })
  ],
  root: 'src',
  publicDir: '../public',
  build: {
    outDir: '../dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        popup: resolve(__dirname, 'src/popup/index.html'),
        background: resolve(__dirname, 'src/background/index.ts'),
        offscreen: resolve(__dirname, 'src/offscreen/index.html'),
        stats: resolve(__dirname, 'src/stats/index.html'),
      },
      output: {
        entryFileNames: (chunkInfo) => {
          if (chunkInfo.name === 'offscreen') return 'offscreen/index.js';
          if (chunkInfo.name === 'stats') return 'stats/index.js';
          return '[name]/index.js';
        },
        chunkFileNames: 'chunks/[name]-[hash].js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.name === 'index.html') {
             if (assetInfo.source.includes('<title>CookieSyncer Offscreen</title>')) {
               return 'offscreen/index.html';
             }
             if (assetInfo.source.includes('<title>续期统计</title>')) {
               return 'stats/index.html';
             }
          }
          return 'assets/[name]-[hash].[ext]';
        },
      },
    },
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 3000,
    open: false,
  },
}))
