<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useAppStore } from '@/store';
import { api } from '@/api';
import { Instance } from '@/types';
import { RefreshCw, Search, ArrowUp, ArrowDown } from 'lucide-vue-next';

const store = useAppStore();
const instances = ref<Instance[]>([]);
const loading = ref(false);
const searchQuery = ref('');
const sortKey = ref<keyof Instance>('last_seen_at');
const sortOrder = ref<'asc' | 'desc'>('desc');
let refreshInterval: any = null;

const fetchInstances = async () => {
  loading.value = true;
  try {
    instances.value = await api.getInstances();
  } catch (e) {
    store.addToast(store.t('alerts.error'), 'error');
    console.error(e);
  } finally {
    loading.value = false;
  }
};

const isOnline = (lastSeen: number) => {
  return (Date.now() / 1000 - lastSeen) < 60;
};

const formatUptime = (ts: number) => {
    const diff = (Date.now() / 1000 - ts);
    if (diff < 60) return Math.floor(diff) + 's';
    if (diff < 3600) return Math.floor(diff / 60) + 'm';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h';
    return Math.floor(diff / 86400) + 'd ' + Math.floor((diff % 86400) / 3600) + 'h';
};

const filteredInstances = computed(() => {
    let result = instances.value.filter(inst => {
        const query = searchQuery.value.toLowerCase();
        return inst.service_name.toLowerCase().includes(query) || 
               inst.hostname?.toLowerCase().includes(query) ||
               inst.ip?.toLowerCase().includes(query);
    });

    result.sort((a, b) => {
        let valA = a[sortKey.value];
        let valB = b[sortKey.value];
        
        if (sortKey.value === 'last_seen_at') {
            const onlineA = isOnline(a.last_seen_at as number) ? 1 : 0;
            const onlineB = isOnline(b.last_seen_at as number) ? 1 : 0;
            return sortOrder.value === 'desc' ? onlineB - onlineA : onlineA - onlineB;
        }

        if (valA! < valB!) return sortOrder.value === 'asc' ? -1 : 1;
        if (valA! > valB!) return sortOrder.value === 'asc' ? 1 : -1;
        return 0;
    });

    return result;
});

const toggleSort = (key: keyof Instance) => {
    if (sortKey.value === key) {
        sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc';
    } else {
        sortKey.value = key;
        sortOrder.value = 'asc';
    }
};

onMounted(() => {
    fetchInstances();
    refreshInterval = setInterval(() => {
        if (!loading.value) api.getInstances().then(res => instances.value = res);
    }, 10000);
});

onUnmounted(() => {
    if (refreshInterval) clearInterval(refreshInterval);
});
</script>

<template>
  <main class="flex-1 flex flex-col overflow-auto bg-gray-900 p-8">
    <div class="flex justify-between items-center mb-6">
      <div class="flex items-baseline gap-4">
        <h2 class="text-2xl font-bold text-white">{{ store.t('nav.instances') }}</h2>
        <span class="text-xs text-gray-500 font-mono tracking-tight uppercase">
            {{ filteredInstances.length }} {{ store.t('nav.instances') }}
        </span>
      </div>
      <div class="flex items-center gap-3">
        <div class="relative group">
          <Search class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 group-focus-within:text-blue-400 transition-colors" />
          <input 
            v-model="searchQuery"
            type="text" 
            :placeholder="store.t('search.placeholder')"
            class="pl-9 pr-4 py-1.5 bg-gray-800/50 border border-gray-700 rounded-lg text-sm text-gray-200 focus:outline-none focus:border-blue-500/50 focus:bg-gray-800 transition-all w-64"
          />
        </div>
        <button @click="fetchInstances" class="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded-lg text-sm text-gray-300 flex items-center gap-2 transition-colors">
            <RefreshCw class="w-4 h-4" :class="{'animate-spin': loading}" />
            {{ store.t('search.refresh') }}
        </button>
      </div>
    </div>

    <div class="bg-gray-800/50 rounded-xl border border-gray-800 overflow-hidden shadow-2xl">
      <table class="w-full text-left">
        <thead class="bg-black/20 text-[10px] uppercase font-bold text-gray-500 tracking-widest border-b border-gray-800">
          <tr>
            <th class="px-6 py-4 cursor-pointer hover:text-gray-300 transition-colors" @click="toggleSort('last_seen_at')">
                <div class="flex items-center gap-1">
                    {{ store.t('table.status') }}
                    <ArrowUp v-if="sortKey === 'last_seen_at' && sortOrder === 'asc'" class="w-3 h-3" />
                    <ArrowDown v-if="sortKey === 'last_seen_at' && sortOrder === 'desc'" class="w-3 h-3" />
                </div>
            </th>
            <th class="px-6 py-4 cursor-pointer hover:text-gray-300 transition-colors" @click="toggleSort('service_name')">
                <div class="flex items-center gap-1">
                    {{ store.t('table.service') }}
                    <ArrowUp v-if="sortKey === 'service_name' && sortOrder === 'asc'" class="w-3 h-3" />
                    <ArrowDown v-if="sortKey === 'service_name' && sortOrder === 'desc'" class="w-3 h-3" />
                </div>
            </th>
            <th class="px-6 py-4 cursor-pointer hover:text-gray-300 transition-colors" @click="toggleSort('hostname')">
                <div class="flex items-center gap-1">
                    {{ store.t('table.host') }}
                    <ArrowUp v-if="sortKey === 'hostname' && sortOrder === 'asc'" class="w-3 h-3" />
                    <ArrowDown v-if="sortKey === 'hostname' && sortOrder === 'desc'" class="w-3 h-3" />
                </div>
            </th>
            <th class="px-6 py-4">IP</th>
            <th class="px-6 py-4">LANG</th>
            <th class="px-6 py-4">{{ store.t('table.sdk_ver') }}</th>
            <th class="px-6 py-4 cursor-pointer hover:text-gray-300 transition-colors" @click="toggleSort('registered_at')">
                <div class="flex items-center gap-1">
                    {{ store.t('table.uptime') }}
                    <ArrowUp v-if="sortKey === 'registered_at' && sortOrder === 'asc'" class="w-3 h-3" />
                    <ArrowDown v-if="sortKey === 'registered_at' && sortOrder === 'desc'" class="w-3 h-3" />
                </div>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-800">
          <tr v-for="inst in filteredInstances" :key="inst.instance_id" class="text-sm hover:bg-white/[0.02] transition-colors group">
            <td class="px-6 py-4">
              <div class="flex items-center space-x-2">
                <div class="relative flex h-2.5 w-2.5">
                  <span v-if="isOnline(inst.last_seen_at)" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                  <span :class="isOnline(inst.last_seen_at) ? 'bg-green-500' : 'bg-red-500'" class="relative inline-flex rounded-full h-2.5 w-2.5"></span>
                </div>
                <span :class="isOnline(inst.last_seen_at) ? 'text-green-400' : 'text-gray-500'" class="font-bold text-xs uppercase">
                    {{ isOnline(inst.last_seen_at) ? store.t('common.online') : store.t('common.offline') }}
                </span>
              </div>
            </td>
            <td class="px-6 py-4 font-medium text-white">{{ inst.service_name }}</td>
            <td class="px-6 py-4 text-gray-400 font-mono text-xs">{{ inst.hostname }}</td>
            <td class="px-6 py-4 text-gray-500 font-mono text-xs">{{ inst.ip }}</td>
            <td class="px-6 py-4">
                <span class="px-1.5 py-0.5 bg-gray-800 text-gray-400 rounded text-[10px] font-bold uppercase ring-1 ring-inset ring-gray-700">
                    {{ inst.language }}
                </span>
            </td>
            <td class="px-6 py-4 font-mono text-xs text-gray-500">{{ inst.sdk_version }}</td>
            <td class="px-6 py-4 text-gray-400">{{ formatUptime(inst.registered_at) }}</td>
          </tr>
          <tr v-if="filteredInstances.length === 0 && !loading">
            <td colspan="7" class="px-6 py-12 text-center text-gray-600">
                <div class="flex flex-col items-center gap-2">
                    <Search class="w-8 h-8 text-gray-800" />
                    {{ store.t('empty.no_instances') }}
                </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </main>
</template>
