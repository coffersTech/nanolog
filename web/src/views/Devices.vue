<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useAppStore } from '@/store';
import { api } from '@/api';
import { Instance } from '@/types';
import { Search, Trash2, Monitor, Globe, Clock, ShieldCheck } from 'lucide-vue-next';

const store = useAppStore();
const devices = ref<Instance[]>([]);
const loading = ref(false);
const searchQuery = ref('');

const fetchDevices = async () => {
  loading.value = true;
  try {
    devices.value = await api.getDevices();
  } catch (e) {
    store.addToast(store.t('alerts.error'), 'error');
    console.error(e);
  } finally {
    loading.value = false;
  }
};

const deleteDevice = async (id: string) => {
    if (!confirm(store.t('devices.unbind_confirm'))) return;
    try {
        await api.deleteDevice(id);
        devices.value = devices.value.filter(d => d.instance_id !== id);
        store.addToast(store.t('alerts.success'), 'success');
    } catch (e) {
        store.addToast(store.t('alerts.error'), 'error');
    }
};

const isOnline = (lastSeen: number) => {
  return (Date.now() / 1000 - lastSeen) < 60;
};

const formatDate = (ts: number) => {
    if (!ts || ts <= 0) return store.t('common.never');
    return new Date(ts * 1000).toLocaleString();
};

const filteredDevices = computed(() => {
    let result = devices.value.filter(d => {
        const query = searchQuery.value.toLowerCase();
        return (d.service_name?.toLowerCase() || '').includes(query) || 
               (d.hostname?.toLowerCase() || '').includes(query) ||
               (d.ip?.toLowerCase() || '').includes(query);
    });
    return result;
});

onMounted(fetchDevices);
</script>

<template>
  <main class="flex-1 flex flex-col overflow-auto bg-gray-900 p-8">
    <div class="flex justify-between items-center mb-6">
      <div class="flex items-baseline gap-4">
        <h2 class="text-2xl font-bold text-white">{{ store.t('nav.devices') }}</h2>
        <span class="text-xs text-gray-500 font-mono tracking-tight uppercase">
            {{ filteredDevices.length }} {{ store.t('devices.total_assets') }}
        </span>
      </div>
      <div class="flex items-center gap-3">
        <div class="relative group">
          <Search class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 group-focus-within:text-blue-400 transition-colors" />
          <input 
            v-model="searchQuery"
            type="text" 
            :placeholder="store.t('devices.search_placeholder')"
            class="pl-9 pr-4 py-1.5 bg-gray-800/50 border border-gray-700 rounded-lg text-sm text-gray-200 focus:outline-none focus:border-blue-500/50 focus:bg-gray-800 transition-all w-64"
          />
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 gap-4">
        <div v-for="device in filteredDevices" :key="device.instance_id" 
             class="bg-gray-800/40 border border-gray-800 rounded-xl p-5 hover:border-gray-700 transition-all group relative overflow-hidden">
            
            <!-- Background Decoration -->
            <div class="absolute -right-4 -bottom-4 opacity-[0.03] group-hover:opacity-[0.07] transition-opacity">
                <Monitor class="w-24 h-24" />
            </div>

            <div class="flex justify-between items-start mb-4">
                <div class="flex items-center gap-3">
                    <div class="p-2 bg-blue-500/10 rounded-lg text-blue-400 group-hover:bg-blue-500/20 transition-colors">
                        <Monitor class="w-5 h-5" />
                    </div>
                    <div>
                        <h3 class="text-sm font-bold text-white group-hover:text-blue-400 transition-colors truncate max-w-[150px]">
                            {{ device.hostname }}
                        </h3>
                        <p class="text-[10px] text-gray-500 uppercase tracking-wider font-bold">{{ device.service_name }}</p>
                    </div>
                </div>
                <div class="flex items-center gap-2">
                    <div class="flex items-center gap-1.5">
                        <span :class="isOnline(device.last_seen_at) ? 'bg-green-500' : 'bg-gray-600'" class="h-1.5 w-1.5 rounded-full"></span>
                        <span class="text-[10px] font-bold uppercase tracking-widest text-gray-500">
                             {{ isOnline(device.last_seen_at) ? store.t('common.online') : store.t('common.offline') }}
                        </span>
                    </div>
                    <button @click="deleteDevice(device.instance_id)" class="text-gray-600 hover:text-red-400 transition-colors p-1 origin-center hover:scale-110">
                        <Trash2 class="w-4 h-4" />
                    </button>
                </div>
            </div>

            <div class="space-y-3">
                <div class="flex items-center justify-between text-xs">
                    <div class="flex items-center gap-2 text-gray-500">
                        <Globe class="w-3.5 h-3.5" />
                        <span>{{ store.t('devices.ip_address') }}</span>
                    </div>
                    <span class="text-gray-300 font-mono">{{ device.ip }}</span>
                </div>
                <div class="flex items-center justify-between text-xs">
                    <div class="flex items-center gap-2 text-gray-500">
                        <ShieldCheck class="w-3.5 h-3.5" />
                        <span>{{ store.t('devices.sdk_version') }}</span>
                    </div>
                    <span class="text-gray-400 font-mono text-[10px]">{{ device.sdk_version }}</span>
                </div>
                <div class="flex items-center justify-between text-xs">
                    <div class="flex items-center gap-2 text-gray-500">
                        <Clock class="w-3.5 h-3.5" />
                        <span>{{ store.t('devices.first_seen') }}</span>
                    </div>
                    <span class="text-gray-400 text-[10px]">{{ formatDate(device.registered_at) }}</span>
                </div>
            </div>

            <div class="mt-4 pt-4 border-t border-gray-800/50 flex items-center justify-between">
                <span class="text-[10px] text-gray-600 font-bold uppercase tracking-widest">{{ store.t('devices.last_online') }}</span>
                <span class="text-[10px] text-gray-500">{{ formatDate(device.last_seen_at) }}</span>
            </div>
        </div>

        <div v-if="filteredDevices.length === 0" class="col-span-full py-20 bg-gray-800/20 rounded-2xl border border-dashed border-gray-800 flex flex-col items-center justify-center gap-3 text-gray-600">
            <Monitor class="w-12 h-12 opacity-20" />
            <p>{{ store.t('devices.no_managed_devices') }}</p>
        </div>
    </div>
  </main>
</template>
