<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useAppStore } from '@/store';
import { api } from '@/api';
import { ChevronDown, X } from 'lucide-vue-next';

const store = useAppStore();
const isOpen = ref(false);
const selectedService = ref('');
const services = ref<{ name: string; count: number }[]>([]);
const loading = ref(false);
const searchQuery = ref('');
const dropdownRef = ref<HTMLElement | null>(null);

const emit = defineEmits(['select']);

const filteredServices = computed(() => {
  if (!searchQuery.value) return services.value;
  return services.value.filter(s => 
    s.name.toLowerCase().includes(searchQuery.value.toLowerCase())
  );
});

const fetchServices = async () => {
  loading.value = true;
  try {
    const stats = await api.getStats();
    if (stats && stats.top_services) {
      const sortedServices = Object.entries(stats.top_services)
        .map(([name, count]) => ({ name, count: count as number }))
        .sort((a, b) => b.count - a.count);
      services.value = sortedServices;
    }
  } catch (e) {
    console.error('Failed to fetch services:', e);
  } finally {
    loading.value = false;
  }
};

const handleSelect = (service: string) => {
  selectedService.value = service;
  isOpen.value = false;
  emit('select', service);
};

const handleClear = () => {
  selectedService.value = '';
  isOpen.value = false;
  emit('select', '');
};

const handleClickOutside = (event: MouseEvent) => {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    isOpen.value = false;
  }
};

const formatCount = (count: number) => {
  if (count >= 1000000) return `${(count / 1000000).toFixed(1)}M`;
  if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
  return count.toString();
};

const displayLabel = computed(() => {
  if (!selectedService.value) {
    return store.t('search.all_services');
  }
  return selectedService.value;
});

onMounted(() => {
  fetchServices();
  document.addEventListener('click', handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
});
</script>

<template>
  <div ref="dropdownRef" class="relative">
    <button 
      @click.stop="isOpen = !isOpen"
      :class="[
        'flex items-center space-x-2 px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm transition-all',
        selectedService ? 'text-cyan-400 border-cyan-500/50' : 'text-gray-400 hover:text-gray-200 hover:border-gray-600'
      ]"
    >
      <span class="font-medium">{{ displayLabel }}</span>
      <ChevronDown class="w-4 h-4 transition-transform" :class="isOpen ? 'rotate-180' : ''" />
    </button>

    <!-- Dropdown -->
    <div v-if="isOpen"
      class="absolute top-full left-0 mt-2 w-64 bg-gray-900/95 backdrop-blur-xl border border-gray-700 rounded-xl shadow-2xl z-50 overflow-hidden">
      <!-- Search Input -->
      <div class="p-2 border-b border-gray-800">
        <input 
          type="text"
          v-model="searchQuery"
          placeholder="Filter services..."
          class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
          @click.stop
        />
      </div>

      <!-- Service List -->
      <div class="max-h-64 overflow-y-auto">
        <!-- All Services Option -->
        <button 
          @click.stop="handleClear()"
          :class="[
            'w-full flex items-center justify-between px-4 py-3 text-sm transition-all',
            !selectedService ? 'text-cyan-400 bg-cyan-500/10' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
          ]"
        >
          <span>{{ store.t('search.all_services') }}</span>
          <X v-if="selectedService" class="w-4 h-4" />
        </button>

        <!-- Loading State -->
        <div v-if="loading" class="px-4 py-3 text-sm text-gray-500 text-center">
          Loading...
        </div>

        <!-- Services -->
        <button v-else
          v-for="svc in filteredServices"
          :key="svc.name"
          @click.stop="handleSelect(svc.name)"
          :class="[
            'w-full flex items-center justify-between px-4 py-3 text-sm transition-all',
            selectedService === svc.name ? 'text-cyan-400 bg-cyan-500/10' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
          ]"
        >
          <span class="truncate">{{ svc.name }}</span>
          <span class="text-xs text-gray-500 font-mono">{{ formatCount(svc.count) }}</span>
        </button>

        <!-- Empty State -->
        <div v-if="!loading && filteredServices.length === 0" class="px-4 py-3 text-sm text-gray-500 text-center">
          No services found
        </div>
      </div>
    </div>
  </div>
</template>