<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue';
import { Chart, registerables } from 'chart.js';
import { Loader2 } from 'lucide-vue-next';

Chart.register(...registerables);

const props = defineProps<{
  data: { time: number; count: number }[];
  interval: number;
  duration: number;
  loading?: boolean;
}>();

const chartRef = ref<HTMLCanvasElement | null>(null);
let chartInstance: Chart | null = null;

const pad = (n: number) => String(n).padStart(2, '0');

const initChart = () => {
  if (!chartRef.value) return;
  const ctx = chartRef.value.getContext('2d');
  if (!ctx) return;

  chartInstance = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: props.data.map((p) => p.time),
      datasets: [{
        label: 'Log Volume',
        data: props.data.map((p) => p.count),
        backgroundColor: '#06b6d4', // cyan-500
        borderRadius: 2,
        borderSkipped: false,
        hoverBackgroundColor: '#0891b2'
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      layout: {
        padding: { top: 25 }
      },
      interaction: {
        mode: 'index',
        intersect: false,
      },
      scales: {
        x: {
          ticks: {
            color: '#6b7280',
            maxTicksLimit: 10,
            callback: function (value) {
              const label = this.getLabelForValue(value as number);
              if (!label) return '';
              const ts = Number(label);
              const d = new Date(ts / 1000000);
              
              if (props.duration > 30 * 86400) return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
              if (props.duration > 86400) return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;

              if (props.interval < 60) {
                return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
              }
              return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
            }
          },
          grid: { display: false }
        },
        y: {
          ticks: { color: '#6b7280' },
          grid: { color: '#1f2937' }
        }
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: 'rgba(255, 255, 255, 0.95)',
          titleColor: '#1f2937', 
          bodyColor: '#374151', 
          borderColor: '#e5e7eb',
          borderWidth: 1,
          padding: 12,
          boxPadding: 4,
          usePointStyle: true,
          callbacks: {
            title: (items) => {
              if (items.length > 0) {
                const startTs = parseInt(items[0].label);
                const startD = new Date(startTs / 1000000);
                const endD = new Date((startTs / 1000000) + (props.interval * 1000));

                const fmtTime = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
                const fmtDate = (d: Date) => `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

                if (props.duration > 86400) {
                  return `${fmtDate(startD)} ${fmtTime(startD)} - ${fmtTime(endD)}`;
                }
                return `${fmtTime(startD)} - ${fmtTime(endD)}`;
              }
              return '';
            },
            label: (context) => {
              const count = context.raw;
              let intervalStr = props.interval + 's';
              if (props.interval >= 3600) intervalStr = Math.round(props.interval / 3600 * 10) / 10 + 'h';
              else if (props.interval >= 60) intervalStr = Math.round(props.interval / 60 * 10) / 10 + 'm';

              return [
                ` Logs          ${count}`,
                ` Interval      ${intervalStr}`
              ];
            }
          }
        }
      }
    }
  });
};

watch(() => props.data, (newData: any[]) => {
  if (chartInstance) {
    chartInstance.data.labels = newData.map((p) => p.time);
    chartInstance.data.datasets[0].data = newData.map((p) => p.count);
    
    // Update options to reflect new interval/duration in tooltips and axes
    if (chartInstance.options.scales?.x?.ticks) {
      // Re-assigning or forcing update logic if needed
      // Chart.js options are reactive if updated like this:
      chartInstance.update();
    }
  }
});

onMounted(initChart);
onUnmounted(() => {
  if (chartInstance) chartInstance.destroy();
});
</script>

<template>
  <div class="h-full w-full relative">
    <canvas ref="chartRef"></canvas>
  </div>
</template>
