<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { useAppStore } from '@/store';
import { api } from '@/api';
import { Stats } from '@/types';
import Chart from '@/components/Chart.vue';
import DonutChart from '@/components/DonutChart.vue';
import { BarChart3, Activity, Database, LayoutGrid } from 'lucide-vue-next';

const store = useAppStore();
const stats = ref<Stats>({ ingestion_rate: 0, disk_usage: 0, total_logs: 0 });
const histogram = ref<any[]>([]);
const topServicesList = ref<{name: string, count: number}[]>([]);
const levelData = ref<any[]>([]);
const lastRefreshTime = ref(new Date().toLocaleTimeString());

const LEVEL_COLORS: Record<string, string> = {
    'ERROR': '#ef4444',
    'WARN': '#f59e0b',
    'INFO': '#10b981',
    'DEBUG': '#3b82f6',
    'FATAL': '#7f1d1d',
    'UNKNOWN': '#6b7280'
};

const formatBytes = (bytes: number) => {
  if (!+bytes) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const formatNumber = (num: number) => new Intl.NumberFormat().format(num);

const refreshData = async () => {
    try {
        const rawStats = await api.getStats();
        stats.value = rawStats;
        
        // Process Top Services
        if (rawStats.top_services) {
            topServicesList.value = Object.entries(rawStats.top_services)
                .map(([name, count]) => ({ name, count: count as number }))
                .sort((a, b) => b.count - a.count)
                .slice(0, 5);
        }

        // Process Level Distribution (Dynamic)
        if (rawStats.level_dist) {
            levelData.value = Object.entries(rawStats.level_dist)
                .filter(([_, count]) => (count as number) > 0)
                .map(([name, count]) => ({
                    name: store.t(`levels.${name.toLowerCase()}`) || name,
                    value: count as number,
                    color: LEVEL_COLORS[name] || '#6b7280'
                }))
                .sort((a, b) => b.value - a.value);
        }
        
        // Fetch last hour histogram
        const hRes = await api.getHistogram('interval=60');
        histogram.value = hRes.map((p: any) => ({
            time: p.time,
            count: p.count
        }));
        lastRefreshTime.value = new Date().toLocaleTimeString();
    } catch (e) {
        console.error('Dashboard refresh error:', e);
    }
};

let timer: any;
onMounted(() => {
  refreshData();
  timer = setInterval(refreshData, 5000);
});

onUnmounted(() => clearInterval(timer));
</script>

<template>
  <main class="flex-1 overflow-auto bg-gray-900 p-8 custom-scrollbar">
    <div class="flex items-center justify-between mb-8">
      <div>
        <h2 class="text-2xl font-black text-white tracking-tight">{{ store.t('dashboard.title') }}</h2>
        <p class="text-gray-500 text-sm mt-1">{{ store.t('dashboard.subtitle') }}</p>
      </div>
      <div class="flex space-x-2">
        <div class="px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700 flex items-center space-x-2 text-xs font-bold text-gray-400">
           <div class="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
           <span>{{ store.t('dashboard.system_live') }}</span>
        </div>
      </div>
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
      <div class="bg-gradient-to-br from-gray-800 to-gray-900 rounded-2xl p-6 border border-gray-800 shadow-xl relative overflow-hidden group">
        <div class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
           <Activity class="w-16 h-16 text-cyan-400" />
        </div>
        <div class="text-gray-500 text-[10px] font-black uppercase tracking-widest mb-2">{{ store.t('search.ingestion_rate') }}</div>
        <div class="text-4xl font-black text-white tracking-tighter">{{ stats.ingestion_rate.toFixed(1) }}</div>
        <div class="text-xs font-bold text-cyan-500/80 mt-1 uppercase">{{ store.t('dashboard.logs_sec') }}</div>
      </div>

      <div class="bg-gradient-to-br from-gray-800 to-gray-900 rounded-2xl p-6 border border-gray-800 shadow-xl relative overflow-hidden group">
        <div class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
           <LayoutGrid class="w-16 h-16 text-purple-400" />
        </div>
        <div class="text-gray-500 text-[10px] font-black uppercase tracking-widest mb-2">{{ store.t('search.total_logs') }}</div>
        <div class="text-4xl font-black text-white tracking-tighter">{{ formatNumber(stats.total_logs) }}</div>
        <div class="text-xs font-bold text-purple-500/80 mt-1 uppercase">{{ store.t('dashboard.stored_entries') }}</div>
      </div>

      <div class="bg-gradient-to-br from-gray-800 to-gray-900 rounded-2xl p-6 border border-gray-800 shadow-xl relative overflow-hidden group">
        <div class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
           <Database class="w-16 h-16 text-amber-400" />
        </div>
        <div class="text-gray-500 text-[10px] font-black uppercase tracking-widest mb-2">{{ store.t('search.disk_usage') }}</div>
        <div class="text-4xl font-black text-white tracking-tighter">{{ formatBytes(stats.disk_usage) }}</div>
        <div class="text-xs font-bold text-amber-500/80 mt-1 uppercase">{{ store.t('dashboard.storage_used') }}</div>
      </div>
    </div>

    <!-- Row 1: Ingestion Trend (Full Width) -->
    <div class="mb-8">
       <Chart :data="histogram" :title="store.t('dashboard.trend_title')" :refresh-time="lastRefreshTime" />
    </div>

    <!-- Row 2: Level Dist & Top Services (Side by Side) -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
       <div>
         <DonutChart :title="store.t('dashboard.level_dist')" :data="levelData" />
       </div>
       
       <div>
         <!-- Top Services (Horizontal Bar Chart Style) -->
         <div class="bg-gray-800 rounded-xl border border-gray-700 p-8 shadow-xl h-full">
            <h3 class="text-white font-bold text-xl mb-10">{{ store.t('dashboard.top_services') }}</h3>
            
            <div class="relative pb-16">
               <div v-for="(svc, i) in topServicesList" :key="i" 
                    class="flex items-center mb-6 last:mb-0 group">
                  <!-- Service Name -->
                  <div class="w-24 md:w-32 text-right pr-6 text-xs font-bold text-gray-500 truncate group-hover:text-gray-300 transition-colors uppercase tracking-tight">
                     {{ svc.name }}
                  </div>
                  
                  <!-- Bar Container -->
                  <div class="flex-1 h-6 bg-gray-900/30 rounded-r relative overflow-hidden">
                     <div class="h-full bg-cyan-500 rounded-r transition-all duration-1000 ease-out shadow-[0_0_15px_rgba(6,182,212,0.3)]"
                          :style="{ width: (svc.count / (topServicesList[0]?.count || 1) * 100) + '%' }">
                     </div>
                  </div>
               </div>

               <!-- X-Axis Labels -->
               <div class="absolute bottom-0 left-24 md:left-32 right-0 border-t border-gray-700 mt-4 flex justify-between pt-4">
                  <div v-for="n in 5" :key="n" class="relative">
                     <!-- Tick -->
                     <div class="absolute top-[-17px] left-0 w-px h-2 bg-gray-700"></div>
                     <!-- Label -->
                     <div class="text-[9px] text-gray-500 font-bold transform rotate-[-35deg] origin-top-left -ml-2 whitespace-nowrap">
                        {{ formatNumber(Math.floor((topServicesList[0]?.count || 0) * ((n-1)/4))) }}
                     </div>
                  </div>
               </div>

               <!-- Empty State -->
               <div v-if="topServicesList.length === 0" class="text-center py-12 text-gray-600 italic text-sm">
                  {{ store.t('dashboard.no_service_data') }}
               </div>
            </div>
         </div>
       </div>
    </div>
  </main>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #374151;
  border-radius: 10px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #4b5563;
}
</style>
