<script setup lang="ts">
import { ref, watch, nextTick } from 'vue';
import { X, Clock, ShieldCheck, Loader2 } from 'lucide-vue-next';
import { LogItem } from '@/types';
import { useAppStore } from '@/store';

const store = useAppStore();

const props = defineProps<{
  show: boolean;
  loading: boolean;
  anchor: LogItem | null;
  pre: LogItem[];
  post: LogItem[];
}>();

const emit = defineEmits(['close']);

const scrollContainer = ref<HTMLElement | null>(null);
const anchorRef = ref<HTMLElement | null>(null);

const pad = (n: number) => String(n).padStart(2, '0');

const formatTimestamp = (ts: number) => {
  const ms = Math.floor(ts / 1000000);
  const d = new Date(ms);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${String(ts % 1000000000).padStart(9, '0').slice(0, 3)}`;
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

// Auto-scroll to anchor when shown or data loaded
watch(() => [props.show, props.loading], async ([show, loading]) => {
  if (show && !loading) {
    await nextTick();
    if (anchorRef.value) {
      anchorRef.value.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }
});
</script>

<template>
  <Transition name="fade">
    <div v-if="show" class="fixed inset-0 z-[100] flex items-center justify-center p-4 lg:p-12">
      <!-- Backdrop -->
      <div class="absolute inset-0 bg-gray-950/80 backdrop-blur-sm" @click="emit('close')"></div>
      
      <!-- Modal Content -->
      <div class="relative w-full max-w-5xl bg-gray-900/90 border border-gray-800 rounded-3xl shadow-2xl flex flex-col overflow-hidden animate-in zoom-in-95 duration-200 h-[80vh]">
        <!-- Header -->
        <div class="px-8 py-6 border-b border-gray-800 flex items-center justify-between bg-gray-900/50 backdrop-blur-md sticky top-0 z-10">
          <div class="flex items-center space-x-4">
            <div class="w-12 h-12 bg-cyan-500/20 rounded-2xl flex items-center justify-center">
              <ShieldCheck class="w-6 h-6 text-cyan-400" />
            </div>
            <div>
              <h3 class="text-xl font-bold text-white tracking-tight">Log Context</h3>
              <p class="text-xs text-gray-500 font-medium">Viewing vicinity logs for <span class="text-cyan-500 font-mono">{{ anchor?.service }}</span></p>
            </div>
          </div>
          <button @click="emit('close')" class="p-2 hover:bg-gray-800 rounded-xl text-gray-500 hover:text-white transition-all">
            <X class="w-6 h-6" />
          </button>
        </div>
        
        <!-- Log Body -->
        <div ref="scrollContainer" class="flex-1 overflow-auto p-8 custom-scrollbar bg-gray-950/30">
          <div v-if="loading" class="h-full flex flex-col items-center justify-center space-y-4">
             <Loader2 class="w-10 h-10 text-cyan-500 animate-spin" />
             <p class="text-gray-500 text-sm font-medium">Fetching adjacent logs...</p>
          </div>
          
          <div v-else class="space-y-1">
            <!-- Pre logs -->
            <div v-for="log in pre" :key="'pre-'+log.timestamp+log.message" 
               class="px-4 py-2 hover:bg-white/[0.03] rounded-lg text-xs font-mono text-gray-500 flex gap-4 transition-colors">
              <span class="text-gray-600 shrink-0 w-32 tabular-nums">{{ formatTimestamp(log.timestamp) }}</span>
              <span :class="getLevelClass(log.level)" class="px-1.5 py-0.5 rounded text-[9px] font-bold shrink-0 h-fit mt-0.5">{{ getLevelText(log.level) }}</span>
              <span class="break-all whitespace-pre-wrap leading-relaxed">{{ log.message }}</span>
            </div>
            
            <!-- Anchor -->
            <div v-if="anchor" ref="anchorRef" class="px-4 py-4 bg-cyan-700/20 border border-cyan-500/40 rounded-xl text-xs font-mono text-gray-100 flex gap-4 ring-8 ring-cyan-500/5 my-4 relative group shadow-lg shadow-cyan-900/20">
              <div class="absolute left-0 top-0 bottom-0 w-1.5 bg-cyan-500 rounded-full"></div>
              <span class="text-cyan-400 shrink-0 w-32 tabular-nums font-bold">{{ formatTimestamp(anchor.timestamp) }}</span>
              <span :class="getLevelClass(anchor.level)" class="px-1.5 py-0.5 rounded text-[9px] font-bold shrink-0 h-fit mt-0.5">{{ getLevelText(anchor.level) }}</span>
              <span class="flex-1 break-all whitespace-pre-wrap leading-relaxed font-bold">{{ anchor.message }}</span>
            </div>
            
            <!-- Post logs -->
            <div v-for="log in post" :key="'post-'+log.timestamp+log.message" 
               class="px-4 py-2 hover:bg-white/[0.03] rounded-lg text-xs font-mono text-gray-500 flex gap-4 transition-colors">
              <span class="text-gray-600 shrink-0 w-32 tabular-nums">{{ formatTimestamp(log.timestamp) }}</span>
              <span :class="getLevelClass(log.level)" class="px-1.5 py-0.5 rounded text-[9px] font-bold shrink-0 h-fit mt-0.5">{{ getLevelText(log.level) }}</span>
              <span class="break-all whitespace-pre-wrap leading-relaxed">{{ log.message }}</span>
            </div>
            
            <div v-if="!pre.length && !post.length && !anchor" class="h-64 flex flex-col items-center justify-center opacity-40">
               <ShieldCheck class="w-16 h-16 text-gray-800 mb-4" />
               <p class="text-gray-600 font-medium">No adjacent logs found in this neighborhood</p>
            </div>
          </div>
        </div>
        
        <!-- Footer -->
        <div class="px-8 py-4 bg-gray-900 border-t border-gray-800 flex justify-end">
          <button @click="emit('close')" class="px-6 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 text-xs font-bold rounded-xl transition-all">
            Close View
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}

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
