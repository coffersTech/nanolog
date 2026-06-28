<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue';
import { LogItem } from '@/types';
import { useAppStore } from '@/store';
import { X, ListFilter } from 'lucide-vue-next';

const store = useAppStore();
const props = defineProps<{
  show: boolean;
  log: LogItem | null;
  searchQuery?: string;
}>();

const emit = defineEmits(['close', 'view-context']);

const drawerRef = ref<HTMLElement | null>(null);

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

const getSearchTerms = () => {
  const q = props.searchQuery;
  if (!q) return [];
  const terms: string[] = [];
  
  const quotedMatches = q.match(/"([^"]+)"/g);
  if (quotedMatches) {
    quotedMatches.forEach(m => terms.push(m.replace(/"/g, '')));
  }
  
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

const closeDrawer = () => {
  emit('close');
};

const handleViewContext = () => {
  if (props.log) {
    emit('view-context', props.log);
  }
};

const handleOverlayClick = (e: MouseEvent) => {
  if (e.target === e.currentTarget) {
    closeDrawer();
  }
};

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.show) {
    closeDrawer();
  }
};

watch(() => props.show, (newVal) => {
  if (newVal) {
    document.body.style.overflow = 'hidden';
  } else {
    document.body.style.overflow = '';
  }
});

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
  document.body.style.overflow = '';
});

const hasLog = computed(() => props.show && props.log);
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="show" class="fixed inset-0 z-50 flex justify-end">
        <div 
          class="absolute inset-0 bg-black/60 backdrop-blur-sm"
          @click="handleOverlayClick"
        ></div>
        
        <div 
          ref="drawerRef"
          class="relative w-full max-w-xl bg-gray-900 border-l border-gray-800 shadow-2xl overflow-hidden flex flex-col"
        >
          <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800 bg-gray-900/95 backdrop-blur-md sticky top-0 z-10">
            <div class="flex items-center gap-3">
              <span :class="getLevelClass(log?.level || 0)" class="px-2.5 py-1 rounded text-[10px] font-bold tracking-tight uppercase shadow-sm">
                {{ getLevelText(log?.level || 0) }}
              </span>
              <span class="text-sm font-bold text-gray-200">{{ store.t('drawer.log_details') }}</span>
            </div>
            <button 
              @click="closeDrawer"
              class="p-2 hover:bg-gray-800 rounded-lg text-gray-400 hover:text-white transition-colors"
            >
              <X class="w-5 h-5" />
            </button>
          </div>

          <div v-if="hasLog" class="flex-1 overflow-auto custom-scrollbar p-6">
            <div class="space-y-6">
              <div>
                <h4 class="text-[10px] font-bold text-gray-500 uppercase tracking-[0.2em] mb-4">{{ store.t('drawer.meta_info') }}</h4>
                <div class="bg-gray-950/80 rounded-2xl p-5 border border-gray-800/50 space-y-3">
                  <div class="flex justify-between text-xs border-b border-gray-800 pb-2">
                    <span class="text-gray-500">{{ store.t('drawer.service') }}</span>
                    <span class="text-gray-200 font-bold">{{ log?.service }}</span>
                  </div>
                  <div class="flex justify-between text-xs border-b border-gray-800 pb-2">
                    <span class="text-gray-500">{{ store.t('drawer.host') }}</span>
                    <span class="text-gray-200 font-bold">{{ log?.host }}</span>
                  </div>
                  <div class="flex justify-between text-xs border-b border-gray-800 pb-2">
                    <span class="text-gray-500">{{ store.t('drawer.level') }}</span>
                    <span :class="getLevelClass(log?.level || 0)" class="px-2 py-0.5 rounded text-[10px] font-bold">{{ getLevelText(log?.level || 0) }}</span>
                  </div>
                  <div class="flex justify-between text-xs border-b border-gray-800 pb-2">
                    <span class="text-gray-500">{{ store.t('drawer.timestamp') }}</span>
                    <span class="text-gray-300 font-mono">{{ formatTimestamp(log?.timestamp || 0) }}</span>
                  </div>
                  <div class="flex justify-between text-xs pb-2">
                    <span class="text-gray-500">{{ store.t('drawer.unix_nanos') }}</span>
                    <span class="text-gray-400 font-mono text-[10px]">{{ log?.timestamp }}</span>
                  </div>
                </div>
              </div>

              <div>
                <h4 class="text-[10px] font-bold text-gray-500 uppercase tracking-[0.2em] mb-4">{{ store.t('drawer.log_payload') }}</h4>
                <div class="bg-gray-950/80 rounded-2xl p-5 border border-gray-800/50 shadow-inner">
                  <pre v-if="isJson(log?.message || '')" 
                    class="text-xs text-green-400/90 whitespace-pre-wrap break-all leading-relaxed font-mono max-h-[50vh] overflow-auto">{{ formatJson(log?.message || '') }}</pre>
                  <pre v-else 
                    class="text-xs text-gray-300 whitespace-pre-wrap break-all leading-relaxed font-mono max-h-[50vh] overflow-auto" v-html="highlightText(log?.message || '')"></pre>
                </div>
              </div>
            </div>
          </div>

          <div v-if="hasLog" class="p-6 border-t border-gray-800 bg-gray-900/95 backdrop-blur-md">
            <button 
              @click="handleViewContext"
              class="w-full px-6 py-3 bg-cyan-600/10 hover:bg-cyan-600/20 text-cyan-400 text-sm font-bold rounded-xl border border-cyan-600/30 transition-all flex items-center justify-center gap-2 group"
            >
              <ListFilter class="w-5 h-5 group-hover:scale-110 transition-transform" />
              {{ store.t('drawer.view_context') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-enter-active,
.drawer-leave-active {
  transition: all 0.3s ease;
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from > div:last-child,
.drawer-leave-to > div:last-child {
  transform: translateX(100%);
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #374151;
  border-radius: 3px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #4b5563;
}
</style>
