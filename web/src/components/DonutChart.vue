<script setup lang="ts">
import { computed } from 'vue';
import { useAppStore } from '@/store';

const store = useAppStore();

interface PropItem {
  name: string;
  value: number;
  color: string;
}

const props = defineProps<{
  title: string;
  data: PropItem[];
}>();

const total = computed(() => props.data.reduce((acc, item) => acc + item.value, 0));

const segments = computed(() => {
  let cumulativeValue = 0;
  const radius = 70;
  const circumference = 2 * Math.PI * radius;
  
  return props.data.map(item => {
    const percentage = total.value > 0 ? (item.value / total.value) : 0;
    const strokeDasharray = `${percentage * circumference} ${circumference}`;
    const offset = (cumulativeValue / total.value) * circumference;
    cumulativeValue += item.value;
    
    return {
      ...item,
      strokeDasharray,
      offset: -offset, // Negative because SVG rotation is clockwise starting from 3 o'clock
      percentage: (percentage * 100).toFixed(1)
    };
  });
});
</script>

<template>
  <div class="bg-gray-800 rounded-xl border border-gray-700 p-8 shadow-xl h-full flex flex-col">
    <h3 class="text-white font-bold text-xl mb-8">{{ title }}</h3>
    
    <div class="flex-1 flex items-center justify-center">
      <div class="flex items-center space-x-12">
        <!-- Donut SVG -->
        <div class="relative w-48 h-48">
          <svg viewBox="0 0 200 200" class="w-full h-full transform -rotate-90">
            <!-- Background circle -->
            <circle
              cx="100"
              cy="100"
              r="70"
              fill="none"
              stroke="rgba(255,255,255,0.05)"
              stroke-width="25"
            />
            
            <!-- Segments -->
            <circle
              v-for="(segment, i) in segments"
              :key="i"
              cx="100"
              cy="100"
              r="70"
              fill="none"
              :stroke="segment.color"
              stroke-width="25"
              :stroke-dasharray="segment.strokeDasharray"
              :stroke-dashoffset="segment.offset"
              class="transition-all duration-1000 ease-in-out"
              stroke-linecap="butt"
            />
          </svg>
          
          <!-- Center Text -->
          <div class="absolute inset-0 flex flex-col items-center justify-center">
            <span class="text-2xl font-black text-white">{{ total > 1000 ? (total/1000).toFixed(1) + 'k' : total }}</span>
            <span class="text-[10px] text-gray-500 uppercase font-bold tracking-tighter">{{ store.t('search.total_logs') }}</span>
          </div>
        </div>
        
        <!-- Legend -->
        <div class="space-y-4">
          <div v-for="(item, i) in segments" :key="i" class="flex items-center space-x-3">
            <div class="w-3 h-3 rounded-full" :style="{ backgroundColor: item.color }"></div>
            <div class="flex flex-col">
              <span class="text-xs font-black text-gray-300 uppercase leading-none">{{ item.name }}</span>
              <span class="text-[10px] text-gray-500 font-bold">{{ item.percentage }}%</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
