<script setup lang="ts">
import { ref } from 'vue';
import { useAppStore } from '@/store';
import { Search, HelpCircle, RefreshCw } from 'lucide-vue-next';
import TimeRangeSelector from './TimeRangeSelector.vue';

const store = useAppStore();
const searchQuery = ref('');
const loading = ref(false);

const emit = defineEmits(['search', 'refresh', 'auto-refresh']);

const currentRange = ref('15m');

const handleSearch = () => {
  emit('search', searchQuery.value, currentRange.value);
};

const handleRefresh = () => {
  emit('refresh', searchQuery.value, currentRange.value);
};

const handleRangeUpdate = (range: string) => {
  currentRange.value = range;
  emit('search', searchQuery.value, range);
};
</script>

<template>
  <header class="h-16 bg-gray-900 border-b border-gray-800 flex items-center px-8 shrink-0 relative z-40">
    <div class="flex items-center space-x-3 flex-1">
      <TimeRangeSelector @update="handleRangeUpdate" @auto-refresh="$emit('auto-refresh', $event)" />

      <div class="relative flex-1 group">
        <input 
          type="text" 
          v-model="searchQuery" 
          @keyup.enter="handleSearch"
          :placeholder="store.t('search.placeholder')"
          class="w-full bg-gray-800 border border-gray-700 rounded-lg pl-10 pr-12 py-2 text-sm text-gray-200 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500 transition-all group-hover:border-gray-600"
        />
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 group-focus-within:text-cyan-400 transition-colors" />
        
        <!-- NanoQL Help Trigger -->
        <div class="absolute right-3 top-1/2 -translate-y-1/2 group/help">
          <HelpCircle class="w-4 h-4 text-gray-500 hover:text-cyan-400 cursor-help transition-colors" />
          <!-- Tooltip (simplified for now) -->
          <div class="invisible group-hover/help:visible absolute right-0 top-8 w-72 bg-gray-900/95 backdrop-blur-xl border border-gray-700 rounded-xl p-4 shadow-2xl z-50 text-xs text-gray-400">
             <h4 class="font-bold text-white mb-2">NanoQL Syntax</h4>
             <p>service:order, level:ERROR, "timeout", AND/OR/NOT, ( ... )</p>
          </div>
        </div>
      </div>
    </div>

    <div class="flex items-center space-x-6 ml-6">
      <button @click="handleRefresh" class="p-2 hover:bg-gray-800 rounded-lg text-gray-400 hover:text-white transition-all">
        <RefreshCw class="w-5 h-5" :class="{'animate-spin': loading}" />
      </button>
    </div>
  </header>
</template>
