<script setup lang="ts">
import { computed, ref } from 'vue';
import { useAppStore } from '@/store';

const store = useAppStore();

interface DataPoint {
  time: number;
  count: number;
}

const props = defineProps<{
  data: DataPoint[];
  title: string;
  refreshTime: string;
}>();

const hoveredIndex = ref<number | null>(null);

const maxCount = computed(() => {
  if (props.data.length === 0) return 1;
  return Math.max(...props.data.map(d => d.count), 1);
});

// Helper to calculate SVG points for smoothing
const getCoordinates = () => {
  const width = 800;
  const height = 200;
  const padding = 20;
  const stepX = (width - padding * 2) / (props.data.length - 1 || 1);
  const scaleY = (height - padding * 2) / maxCount.value;

  return props.data.map((d, i) => ({
    x: padding + i * stepX,
    y: height - padding - (d.count * scaleY)
  }));
};

const smoothPath = computed(() => {
  const coords = getCoordinates();
  if (coords.length < 2) return '';

  // Use simple quadratic bezier or cubic smoothing
  // For simplicity and "softer" look, we'll build a cubic path
  let path = `M ${coords[0].x},${coords[1].y}`; // Start at first point's Y but first point's X? Wait.
  path = `M ${coords[0].x},${coords[0].y}`;

  for (let i = 0; i < coords.length - 1; i++) {
    const p0 = coords[i === 0 ? i : i - 1];
    const p1 = coords[i];
    const p2 = coords[i + 1];
    const p3 = coords[i + 2 === coords.length ? i + 1 : i + 2];

    const cp1x = p1.x + (p2.x - p0.x) / 6;
    const cp1y = p1.y + (p2.y - p0.y) / 6;
    const cp2x = p2.x - (p3.x - p1.x) / 6;
    const cp2y = p2.y - (p3.y - p1.y) / 6;

    path += ` C ${cp1x},${cp1y} ${cp2x},${cp2y} ${p2.x},${p2.y}`;
  }
  return path;
});

const areaPath = computed(() => {
  const p = smoothPath.value;
  if (!p) return '';
  const height = 200;
  const padding = 20;
  const width = 800;
  const lastX = padding + (props.data.length - 1) * ((width - padding * 2) / (props.data.length - 1 || 1));
  return `${p} L ${lastX},${height - padding} L ${padding},${height - padding} Z`;
});

const formatTime = (nanos: number) => {
    // Backend returns nanos, convert to ms
    const date = new Date(nanos / 1000000);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
};

const timeLabels = computed(() => {
  if (props.data.length < 2) return [];
  const labels = [];
  const count = 7; // Show 7 labels (every 10m if 60 points)
  const step = (props.data.length - 1) / (count - 1);
  
  for (let i = 0; i < count; i++) {
    const index = Math.min(Math.round(i * step), props.data.length - 1);
    const date = new Date(props.data[index].time / 1000000);
    labels.push(date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false }));
  }
  return labels;
});

const formatNumber = (num: number) => new Intl.NumberFormat().format(num);
</script>

<template>
  <div class="bg-gray-800 rounded-xl border border-gray-700 p-6 shadow-xl relative overflow-hidden group/chart">
    <div class="flex items-center justify-between mb-6">
      <h3 class="text-gray-300 font-bold uppercase tracking-wider text-xs">{{ title }}</h3>
      <div class="flex items-center space-x-3">
        <div class="text-gray-500 text-[10px] font-mono tabular-nums bg-gray-900/50 px-2 py-0.5 rounded border border-gray-700/30">
          Last sync: {{ refreshTime }}
        </div>
        <div class="text-cyan-400 text-xs font-mono px-2 py-0.5 bg-cyan-400/10 rounded border border-cyan-400/20 flex items-center space-x-1.5">
          <div class="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse"></div>
          <span>{{ store.t('dashboard.chart_live') }}</span>
        </div>
      </div>
    </div>
    
    <div class="relative h-56 w-full group flex">
      <!-- Y-Axis Labels -->
      <div class="w-10 h-48 flex flex-col justify-between text-[10px] text-gray-500 font-mono pr-2 border-r border-gray-700/30 mt-5">
        <span>{{ formatNumber(maxCount) }}</span>
        <span>{{ formatNumber(Math.floor(maxCount * 0.75)) }}</span>
        <span>{{ formatNumber(Math.floor(maxCount * 0.5)) }}</span>
        <span>{{ formatNumber(Math.floor(maxCount * 0.25)) }}</span>
        <span>0</span>
      </div>

      <div class="flex-1 relative h-48 mt-5">
        <svg viewBox="0 0 800 200" class="w-full h-full preserve-3d" preserveAspectRatio="none">
          <defs>
            <linearGradient id="chartGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="rgba(34, 211, 238, 0.4)" />
              <stop offset="100%" stop-color="rgba(34, 211, 238, 0)" />
            </linearGradient>
          </defs>
          
          <!-- Grid lines -->
          <line v-for="i in 4" :key="i" x1="0" :y1="i * 50" x2="800" :y2="i * 50" stroke="rgba(255,255,255,0.05)" stroke-width="1" />
          
          <!-- Area path (Smooth) -->
          <path
            fill="url(#chartGradient)"
            :d="areaPath"
            class="transition-all duration-700 ease-in-out"
          />
          
          <!-- Line path (Smooth) -->
          <path
            fill="none"
            stroke="rgba(34, 211, 238, 1)"
            stroke-width="2.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            :d="smoothPath"
            class="transition-all duration-700 ease-in-out"
          />
          
          <!-- Intersection Hover Zones -->
          <rect
            v-for="(d, i) in data"
            :key="'rect-' + i"
            :x=" (800 / data.length) * i"
            y="0"
            :width="800 / data.length"
            height="200"
            fill="transparent"
            class="cursor-crosshair"
            @mouseenter="hoveredIndex = i"
            @mouseleave="hoveredIndex = null"
          />

          <!-- Data dots on hover -->
          <circle 
            v-if="data.length > 0 && hoveredIndex !== null"
            :cx="20 + hoveredIndex * ((800 - 40) / (data.length - 1 || 1))"
            :cy="200 - 20 - (data[hoveredIndex].count * ((200 - 40) / maxCount))"
            r="5"
            fill="rgba(34, 211, 238, 1)"
            class="shadow-lg pointer-events-none"
            style="filter: drop-shadow(0 0 8px rgba(34, 211, 238,1));"
          />
        </svg>
        
        <!-- Tooltip -->
        <div 
          v-if="hoveredIndex !== null && data[hoveredIndex]"
          class="absolute z-20 pointer-events-none bg-gray-900 border border-gray-700 p-2 rounded shadow-2xl transition-all duration-200"
          :style="{
            left: (hoveredIndex / (data.length - 1 || 1) * 100) + '%',
            bottom: (data[hoveredIndex].count / maxCount * 140) + 'px',
            transform: 'translateX(-50%) translateY(-20px)'
          }"
        >
          <div class="flex flex-col space-y-1">
             <span class="text-[10px] text-gray-500 font-mono">{{ formatTime(data[hoveredIndex].time) }}</span>
             <span class="text-xs font-black text-cyan-400">{{ formatNumber(data[hoveredIndex].count) }} logs</span>
          </div>
        </div>

        <!-- Empty State -->
        <div v-if="data.length === 0" class="absolute inset-0 flex items-center justify-center text-gray-600 italic text-sm">
          Waiting for data...
        </div>

        <!-- X-Axis Labels -->
        <div class="absolute -bottom-6 left-0 right-0 flex justify-between text-[10px] text-gray-500 font-mono">
          <span v-for="(label, i) in timeLabels" :key="i">{{ label }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
