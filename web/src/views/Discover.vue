<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue';
import { useAppStore } from '@/store';
import { api } from '@/api';
import SearchBar from '@/components/SearchBar.vue';
import LogTable from '@/components/LogTable.vue';
import Histogram from '@/components/Histogram.vue';
import ContextModal from '@/components/ContextModal.vue';
import { LogItem } from '@/types';
import { Shield } from 'lucide-vue-next';

const store = useAppStore();
const logs = ref<LogItem[]>([]);
const histogramData = ref<{ time: number; count: number }[]>([]);
const histogramTotal = ref(0);
const loading = ref(false);

const lastQuery = ref('');
const lastRange = ref<any>('15m');
const autoRefreshTimer = ref<any>(null);

const contextModal = ref({
  show: false,
  loading: false,
  anchor: null as LogItem | null,
  pre: [] as LogItem[],
  post: [] as LogItem[]
});

const handleViewContext = async (log: LogItem) => {
  contextModal.value = {
    show: true,
    loading: true,
    anchor: log,
    pre: [],
    post: []
  };

  try {
    const data = await api.getContext(log.timestamp, log.service);
    
    // Combine all received logs for stable re-sorting in frontend
    const allReceived = [
      ...(data.pre || []),
      data.anchor,
      ...(data.post || [])
    ].filter(Boolean) as LogItem[];

    // Stable sort by timestamp, then by message content to avoid jumbled overlap
    allReceived.sort((a, b) => {
      if (a.timestamp !== b.timestamp) return a.timestamp - b.timestamp;
      return a.message.localeCompare(b.message);
    });

    // Find our specific original log in the sorted pool
    // We compare both timestamp and message to be as precise as possible
    let myAnchorIdx = allReceived.findIndex(r => 
      r.timestamp === log.timestamp && r.message === log.message
    );

    // Fallback if not found precisely (unlikely unless data changed)
    if (myAnchorIdx === -1) {
        myAnchorIdx = allReceived.findIndex(r => r.timestamp === log.timestamp);
    }

    if (myAnchorIdx !== -1) {
        contextModal.value.anchor = allReceived[myAnchorIdx];
        contextModal.value.pre = allReceived.slice(0, myAnchorIdx);
        contextModal.value.post = allReceived.slice(myAnchorIdx + 1);
    } else {
        // Ultimate fallback to API structure
        contextModal.value.anchor = data.anchor;
        contextModal.value.pre = data.pre || [];
        contextModal.value.post = data.post || [];
    }
  } catch (e: any) {
    store.addToast(e.message || 'Failed to fetch context logs', 'error');
  } finally {
    contextModal.value.loading = false;
  }
};

const closeContextModal = () => {
  contextModal.value.show = false;
};

// Helper to calculate histogram interval based on range
const getHistogramParams = (range: any) => {
  // If absolute range
  if (typeof range === 'object' && range.start && range.end) {
     const delta = (range.end - range.start) / 1000000000;
     // Target ~50-100 bars
     const rawInterval = delta / 60;
     
     const standardIntervals = [
        1, 2, 5, 10, 30, 60, 300, 600, 1800, 3600, 7200, 14400, 21600, 43200, 86400
     ];
     
     let interval = standardIntervals.find(i => i >= rawInterval) || standardIntervals[standardIntervals.length - 1];
     if (rawInterval > 86400) interval = Math.ceil(rawInterval / 86400) * 86400;
     
     return { interval, duration: delta };
  }

  const map: Record<string, { interval: number; duration: number }> = {
    '5m': { interval: 5, duration: 300 },
    '15m': { interval: 15, duration: 900 },
    '30m': { interval: 30, duration: 1800 },
    '1h': { interval: 60, duration: 3600 },
    '6h': { interval: 360, duration: 21600 },
    '1d': { interval: 1440, duration: 86400 },
    '3d': { interval: 4320, duration: 259200 },
    '7d': { interval: 10080, duration: 604800 },
    '30d': { interval: 43200, duration: 2592000 },
    '90d': { interval: 129600, duration: 7776000 },
    'today': { interval: 1440, duration: 86400 },
    'yesterday': { interval: 1440, duration: 86400 },
    'day_before_yesterday': { interval: 1440, duration: 86400 },
    'this_week': { interval: 10080, duration: 604800 },
    'last_week': { interval: 10080, duration: 604800 },
    'this_month': { interval: 43200, duration: 2592000 },
    'last_month': { interval: 43200, duration: 2592000 },
  };
  return map[range] || map['15m'];
};

const fetchLogs = async (query: string = lastQuery.value, range: any = lastRange.value, append: boolean = false) => {
  loading.value = true;
  lastQuery.value = query;
  lastRange.value = range;
  
  try {
    const params = new URLSearchParams();
    // Moved q append to listParams section below to avoid leaking into Histogram
    
    let start: number, end: number;
    const { interval, duration } = getHistogramParams(range);

    if (typeof range === 'object' && range.start && range.end) {
      start = range.start;
      end = range.end;
      
      // If appending, use the last log's timestamp - 1ns as the new 'end'
      if (append && logs.value.length > 0) {
        end = logs.value[logs.value.length - 1].timestamp - 1;
      }
    } else {
      const now = Date.now() * 1000000;
      end = now;
      
      if (append && logs.value.length > 0) {
        end = logs.value[logs.value.length - 1].timestamp - 1;
      }
      
      start = (typeof range === 'object' && range.start) ? range.start : (now - (duration * 1000000000));
      
      if (range === 'today') {
        const d = new Date(); d.setHours(0,0,0,0);
        start = d.getTime() * 1000000;
      } else if (range === 'yesterday') {
        const d = new Date(); d.setDate(d.getDate() - 1); d.setHours(0,0,0,0);
        start = d.getTime() * 1000000;
        if (!append) {
          const e = new Date(); e.setDate(e.getDate() - 1); e.setHours(23,59,59,999);
          end = e.getTime() * 1000000;
        }
      }
    }

    params.append('start', start.toString());
    params.append('end', end.toString());
    params.append('limit', '100');
    
    const listParams = new URLSearchParams(params);
    const histParams = new URLSearchParams(params);
    
    // List params keep the query
    if (query) listParams.append('q', query);
    // Explicitly ensure Histogram ignores the query
    histParams.delete('q');

    const commonListParams = listParams.toString();
    const commonHistParams = histParams.toString();
    
    const [logsData, hist] = await Promise.all([
      api.searchLogs(commonListParams),
      // Only fetch histogram on initial load, not on append
      append ? Promise.resolve(null) : api.getHistogram(`${commonHistParams}&interval=${interval}`)
    ]);

    if (append) {
      logs.value = [...logs.value, ...(logsData || [])];
      if (!logsData || logsData.length === 0) {
        store.addToast('No more logs found in this range', 'info');
      }
    } else {
      logs.value = logsData || [];
      if (hist) {
        histogramData.value = hist || [];
        histogramTotal.value = histogramData.value.reduce((acc: number, curr: any) => acc + curr.count, 0);
      }
    }
  } catch (e: any) {
    console.error('Fetch error:', e);
    store.addToast(e.message || 'Failed to fetch logs', 'error');
  } finally {
    loading.value = false;
  }
};

const handleLoadMore = () => {
  fetchLogs(lastQuery.value, lastRange.value, true);
};

const handleAutoRefresh = (rate: string) => {
  if (autoRefreshTimer.value) clearInterval(autoRefreshTimer.value);
  if (rate === 'off') return;

  const msMap: Record<string, number> = {
    '5s': 5000,
    '15s': 15000,
    '1m': 60000,
    '5m': 300000,
    '10m': 600000,
    '30m': 1800000,
  };

  const ms = msMap[rate] || 0;

  if (ms > 0) {
    autoRefreshTimer.value = setInterval(() => {
      // Only refresh if not already loading
      if (!loading.value) fetchLogs();
    }, ms);
  }
};

const formatNumber = (num: number) => new Intl.NumberFormat().format(num);
const currentHistogramParams = computed(() => getHistogramParams(lastRange.value));

onMounted(() => fetchLogs());
onUnmounted(() => {
  if (autoRefreshTimer.value) clearInterval(autoRefreshTimer.value);
});
</script>

<template>
  <div class="flex-1 flex flex-col overflow-hidden bg-gray-900">
    <SearchBar @search="fetchLogs" @refresh="fetchLogs" @auto-refresh="handleAutoRefresh" />
    
    <!-- Engine Mode Banner -->
    <div v-if="store.nodeRole === 'engine'"
        class="bg-cyan-500/10 border-b border-cyan-500/20 px-8 py-2 flex items-center space-x-3">
        <Shield class="w-4 h-4 text-cyan-400" />
        <p class="text-xs text-cyan-400 font-medium tracking-wide">{{ store.t('auth.engine_mode_desc') }}</p>
    </div>
    
    <div class="h-48 bg-gray-950 border-b border-gray-800 p-4 relative">
        <div class="absolute top-3 left-4 text-xs font-medium text-gray-500 z-10 pointer-events-none">
            {{ store.t('search.total_logs') }} <span class="text-gray-200 font-bold ml-1 text-sm">{{ formatNumber(histogramTotal) }}</span>
        </div>
        <Histogram :data="histogramData" :interval="currentHistogramParams.interval" :duration="currentHistogramParams.duration" :loading="loading" />
    </div>

    <LogTable :logs="logs" :loading="loading" :search-query="lastQuery" @view-context="handleViewContext" @load-more="handleLoadMore" />
    
    <ContextModal 
      :show="contextModal.show"
      :loading="contextModal.loading"
      :anchor="contextModal.anchor"
      :pre="contextModal.pre"
      :post="contextModal.post"
      @close="closeContextModal"
    />
  </div>
</template>
