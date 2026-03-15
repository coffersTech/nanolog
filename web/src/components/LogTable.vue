<script setup lang="ts">
import { ref, computed } from 'vue';
import { LogItem } from '@/types';
import { useAppStore } from '@/store';
import { ListFilter, Loader2, ChevronDown } from 'lucide-vue-next';

const store = useAppStore();
const props = defineProps<{
  logs: LogItem[];
  loading: boolean;
  searchQuery?: string;
}>();

const emit = defineEmits(['view-context', 'load-more']);

const expandedIndex = ref(-1);

const pad = (n: number) => String(n).padStart(2, '0');

const formatTimestamp = (ts: number) => {
  const ms = Math.floor(ts / 1000000);
  const d = new Date(ms);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
};

const getLevelClass = (l: number) => {
  const map: Record<number, string> = {
    0: 'bg-gray-800 text-gray-400',
    1: 'bg-green-500/10 text-green-500',
    2: 'bg-yellow-500/10 text-yellow-500',
    3: 'bg-red-500/10 text-red-500',
    4: 'bg-purple-500/10 text-purple-500'
  };
  return map[l] || 'bg-gray-800 text-gray-400';
};

const getLevelText = (l: number) => {
  return { 0: 'DEBUG', 1: 'INFO', 2: 'WARN', 3: 'ERROR', 4: 'FATAL' }[l] || 'UNKNOWN';
};

const toggleRow = (idx: number) => {
  expandedIndex.value = expandedIndex.value === idx ? -1 : idx;
};

// Search Highlighting Logic
const getSearchTerms = () => {
  const q = props.searchQuery;
  if (!q) return [];
  const terms: string[] = [];
  
  // Extract quoted strings
  const quotedMatches = q.match(/"([^"]+)"/g);
  if (quotedMatches) {
    quotedMatches.forEach(m => terms.push(m.replace(/"/g, '')));
  }
  
  // Extract value parts from key:value patterns
  const kvMatches = q.match(/\w+:([^\s"]+|"[^"]+")/g);
  if (kvMatches) {
    kvMatches.forEach(m => {
      const val = m.split(':')[1]?.replace(/"/g, '');
      if (val && !['AND', 'OR', 'NOT'].includes(val.toUpperCase())) {
        terms.push(val);
      }
    });
  }
  return [...new Set(terms)];
};

const highlightText = (text: string) => {
  const terms = getSearchTerms();
  if (!terms.length || !text) return text;
  
  let result = text;
  terms.forEach(term => {
    const escapedTerm = term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(`(${escapedTerm})`, 'gi');
    result = result.replace(regex, '<span class="bg-yellow-600/40 text-yellow-100 px-0.5 rounded border-b border-yellow-500/50">$1</span>');
  });
  return result;
};

// JSON Formatting
const isJson = (str: string) => {
  if (!str || !str.startsWith('{')) return false;
  try {
    JSON.parse(str);
    return true;
  } catch (e) {
    return false;
  }
};

const formatJson = (str: string) => {
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch (e) {
    return str;
  }
};
</script>

<template>
  <div class="flex-1 overflow-auto bg-gray-950/20 relative custom-scrollbar">
    <table class="w-full text-left border-collapse min-w-[800px]">
      <thead class="sticky top-0 bg-gray-900/95 backdrop-blur-md shadow-sm z-20">
        <tr>
          <th class="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-widest border-b border-gray-800">{{ store.t('table.timestamp') }}</th>
          <th class="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-widest border-b border-gray-800">{{ store.t('table.level') }}</th>
          <th class="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-widest border-b border-gray-800">{{ store.t('table.service') }}</th>
          <th class="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-widest border-b border-gray-800">{{ store.t('table.host') }}</th>
          <th class="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-widest border-b border-gray-800">{{ store.t('table.message') }}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-800/50 transition-opacity duration-300" :class="{'opacity-20 blur-[0.5px]': loading}">
        <template v-for="(log, idx) in logs" :key="idx">
          <tr @click="toggleRow(idx)" 
              class="group hover:bg-white/[0.02] transition-colors cursor-pointer border-l-2 border-transparent"
              :class="expandedIndex === idx ? 'border-cyan-500 bg-cyan-500/[0.03]' : ''">
            <td class="px-6 py-4 whitespace-nowrap text-xs font-mono text-gray-400 group-hover:text-gray-300">
              {{ formatTimestamp(log.timestamp) }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span :class="getLevelClass(log.level)" class="px-2.5 py-1 rounded text-[10px] font-bold tracking-tight uppercase shadow-sm">
                {{ getLevelText(log.level) }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-400 font-medium">
              {{ log.service }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-400 font-medium">
              {{ log.host }}
            </td>
            <td class="px-6 py-4 text-sm text-gray-300 leading-relaxed font-mono break-all pr-12">
              <div class="truncate max-w-xl" v-html="highlightText(log.message)"></div>
            </td>
          </tr>
          
          <!-- Detail Row -->
          <tr v-if="expandedIndex === idx" class="bg-gray-900 border-b border-gray-800">
            <td colspan="5" class="px-6 py-8">
              <div class="flex flex-col lg:flex-row gap-8">
                <!-- Metadata Panel -->
                <div class="lg:w-1/3 space-y-4">
                  <h4 class="text-[10px] font-bold text-gray-500 uppercase tracking-[0.2em] mb-4">Meta Information</h4>
                  <div class="space-y-3">
                    <div class="flex justify-between text-xs border-b border-gray-800 pb-2">
                      <span class="text-gray-500">Service</span>
                      <span class="text-gray-200 font-bold">{{ log.service }}</span>
                    </div>
                    <div class="flex justify-between text-xs border-b border-gray-800 pb-2">
                      <span class="text-gray-500">Host</span>
                      <span class="text-gray-200 font-bold">{{ log.host }}</span>
                    </div>
                    <div class="flex justify-between text-xs border-b border-gray-800 pb-2">
                      <span class="text-gray-500">Level</span>
                      <span :class="getLevelClass(log.level)" class="px-2 py-0.5 rounded text-[10px] font-bold">{{ getLevelText(log.level) }}</span>
                    </div>
                    <div class="flex justify-between text-xs border-b border-gray-800 pb-2">
                      <span class="text-gray-500">Unix Nanos</span>
                      <span class="text-gray-400 font-mono">{{ log.timestamp }}</span>
                    </div>
                  </div>
                  
                  <button @click.stop="$emit('view-context', log)"
                    class="w-full mt-6 px-4 py-2.5 bg-cyan-600/10 hover:bg-cyan-600/20 text-cyan-400 text-xs font-bold rounded-xl border border-cyan-600/30 transition-all flex items-center justify-center gap-2 group/btn">
                    <ListFilter class="w-4 h-4 group-hover/btn:scale-110 transition-transform" />
                    View Context
                  </button>
                </div>
                
                <!-- Payload Panel -->
                <div class="flex-1 min-w-0">
                  <h4 class="text-[10px] font-bold text-gray-500 uppercase tracking-[0.2em] mb-4">Log Payload</h4>
                  <div class="bg-gray-950/80 rounded-2xl p-5 border border-gray-800/50 shadow-inner">
                    <pre v-if="isJson(log.message)" 
                      class="text-xs text-green-400/90 whitespace-pre-wrap break-all leading-relaxed font-mono custom-scrollbar max-h-[400px] overflow-auto">{{ formatJson(log.message) }}</pre>
                    <pre v-else 
                      class="text-xs text-gray-300 whitespace-pre-wrap break-all leading-relaxed font-mono">{{ log.message }}</pre>
                  </div>
                </div>
              </div>
            </td>
          </tr>
        </template>
        
        <tr v-if="logs.length === 0 && !loading">
            <td colspan="5" class="px-6 py-24 text-center">
                <p class="text-gray-600 font-medium">{{ store.t('empty.no_logs_cluster') }}</p>
            </td>
        </tr>
      </tbody>
    </table>

    <!-- Loading Overlay -->
    <div v-if="loading" class="absolute inset-0 z-30 flex items-center justify-center bg-gray-950/20 backdrop-blur-[2px] animate-in fade-in duration-300">
        <div class="flex flex-col items-center space-y-3 p-8 bg-gray-900/40 rounded-3xl border border-gray-800/50 shadow-2xl">
            <Loader2 class="w-8 h-8 text-cyan-500 animate-spin" />
            <p class="text-xs font-bold text-gray-500 uppercase tracking-[0.2em] animate-pulse">Synchronizing Logs...</p>
        </div>
    </div>

    <!-- Bottom Actions / Load More -->
    <div v-if="logs.length > 0 && !loading" class="p-8 flex justify-center border-t border-gray-800/50 bg-gray-950/10">
      <button @click="$emit('load-more')"
        class="px-8 py-3 bg-gray-900 hover:bg-gray-800 text-gray-400 hover:text-cyan-400 text-xs font-bold rounded-2xl border border-gray-800 hover:border-cyan-500/30 transition-all shadow-lg active:scale-95 flex items-center space-x-2 group/more">
        <span>Load More Logs</span>
        <ChevronDown class="w-4 h-4 group-hover/more:translate-y-0.5 transition-transform" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #1f2937;
  border-radius: 3px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #374151;
}
</style>
