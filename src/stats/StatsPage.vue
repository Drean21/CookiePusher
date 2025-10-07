<template>
  <div class="stats-page">
    <header class="stats-header">
      <h1>续期统计</h1>
    </header>
    <main class="stats-content">
      <div v-if="loading" class="loading-spinner"></div>
      <div v-if="!loading && Object.keys(stats).length === 0" class="no-data">
        暂无统计数据。请等待保活任务运行后查看。
      </div>
      <div v-if="!loading && Object.keys(stats).length > 0" class="charts-grid">
        <div class="chart-container">
          <h2>域名成功率</h2>
          <div ref="domainSuccessChart" class="chart"></div>
        </div>
        <div class="chart-container">
          <h2>总续期次数</h2>
          <div ref="totalActivityChart" class="chart"></div>
        </div>
        <div class="chart-container full-width">
          <h2>最近续期活动</h2>
          <div class="timeline-container">
            <div
              v-for="item in recentHistory"
              :key="item.timestamp"
              class="timeline-item"
            >
              <span class="timestamp">{{ formatTime(item.timestamp) }}</span>
              <span class="status-dot" :class="item.status"></span>
              <span class="cookie-name">{{ item.cookieKey }}</span>
              <span class="status-label" :class="item.status">{{
                formatStatus(item.status)
              }}</span>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import * as echarts from "echarts";
import { nextTick, onMounted, ref } from "vue";
import { sendMessage } from "../utils/message";

const loading = ref(true);
const stats = ref<any>({});
const recentHistory = ref<any[]>([]);

const domainSuccessChart = ref<HTMLElement | null>(null);
const totalActivityChart = ref<HTMLElement | null>(null);

const formatTime = (isoString: string) => new Date(isoString).toLocaleString();
const formatStatus = (status: string) => {
  switch (status) {
    case "success":
      return "成功";
    case "failure":
      return "失败";
    default:
      return "无变化";
  }
};

const getDomainFromKey = (key: string) => key.split("|")[1] || "未知域名";

onMounted(async () => {
  try {
    const response = await sendMessage("getKeepAliveStats");
    if (response.success) {
      stats.value = response.stats;
      prepareChartData();
    }
  } catch (e) {
    console.error("Failed to fetch stats:", e);
  } finally {
    loading.value = false;
  }
});

function prepareChartData() {
  const allHistory: any[] = [];
  const domainData: { [key: string]: { success: number; total: number } } = {};

  for (const key in stats.value) {
    const stat = stats.value[key];
    const domain = getDomainFromKey(key);

    if (!domainData[domain]) {
      domainData[domain] = { success: 0, total: 0 };
    }
    domainData[domain].success += stat.successCount;
    domainData[domain].total += stat.successCount + stat.failureCount;

    stat.history.forEach((h: any) => {
      allHistory.push({ ...h, cookieKey: key });
    });
  }

  recentHistory.value = allHistory
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    .slice(0, 50);

  nextTick(() => {
    initDomainSuccessChart(domainData);
    initTotalActivityChart(stats.value);
  });
}

function initDomainSuccessChart(domainData: { [key: string]: { success: number; total: number } }) {
  if (!domainSuccessChart.value) return;
  const chart = echarts.init(domainSuccessChart.value);
  const chartData = Object.entries(domainData).map(([domain, data]: [string, { success: number; total: number }]) => ({
    name: domain,
    value: data.total > 0 ? (data.success / data.total) * 100 : 0,
  }));

  const option = {
    tooltip: {
      formatter: "{b}: {c.toFixed(1)}%",
    },
    series: [
      {
        type: "pie",
        radius: ["40%", "70%"],
        data: chartData,
        label: {
          show: true,
          formatter: "{b}\n({c.toFixed(1)}%)",
        },
      },
    ],
  };
  chart.setOption(option);
}

function initTotalActivityChart(statsData: any) {
  if (!totalActivityChart.value) return;
  const chart = echarts.init(totalActivityChart.value);
  const chartData = Object.entries(statsData)
    .map(([key, stat]: [string, any]) => ({
      name: key.split("|")[0], // a smaller name
      value: stat.successCount + stat.failureCount,
    }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 15); // Top 15

  const option = {
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "shadow" },
    },
    xAxis: {
      type: "value",
      boundaryGap: [0, 0.01],
    },
    yAxis: {
      type: "category",
      data: chartData.map((d) => d.name).reverse(),
    },
    series: [
      {
        type: "bar",
        data: chartData.map((d) => d.value).reverse(),
      },
    ],
    grid: {
      left: "3%",
      right: "4%",
      bottom: "3%",
      containLabel: true,
    },
  };
  chart.setOption(option);
}
</script>

<style scoped>
.stats-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
}
.stats-header {
  flex-shrink: 0;
  padding: 16px;
  background-color: #1e88e5;
  color: white;
  text-align: center;
}
.stats-header h1 {
  margin: 0;
  font-size: 24px;
}
.stats-content {
  flex-grow: 1;
  padding: 24px;
  overflow-y: auto;
}
.loading-spinner {
  border: 4px solid rgba(0, 0, 0, 0.1);
  border-left-color: #1e88e5;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin: 40px auto;
}
.no-data {
  text-align: center;
  font-size: 18px;
  color: #999;
  margin-top: 40px;
}
.charts-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}
.chart-container {
  background: white;
  padding: 16px;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}
.chart-container.full-width {
  grid-column: 1 / -1;
}
.chart-container h2 {
  margin: 0 0 16px 0;
  font-size: 18px;
  text-align: center;
}
.chart {
  width: 100%;
  height: 300px;
}
.timeline-container {
  max-height: 400px;
  overflow-y: auto;
  padding-right: 10px;
}
.timeline-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #eee;
}
.timestamp {
  font-size: 12px;
  color: #999;
  min-width: 140px;
}
.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin: 0 12px;
}
.status-dot.success {
  background-color: #4caf50;
}
.status-dot.failure {
  background-color: #f44336;
}
.status-dot.no-change {
  background-color: #ccc;
}
.cookie-name {
  flex-grow: 1;
  font-family: monospace;
  font-size: 13px;
}
.status-label {
  font-size: 12px;
  font-weight: 500;
  padding: 2px 6px;
  border-radius: 4px;
  color: white;
}
.status-label.success {
  background-color: #4caf50;
}
.status-label.failure {
  background-color: #f44336;
}
.status-label.no-change {
  background-color: #ccc;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
