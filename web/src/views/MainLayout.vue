<script setup lang="ts">
import { ref, onMounted } from 'vue';
import Sidebar from '@/components/Sidebar.vue';
import Discover from '@/views/Discover.vue';
import Dashboard from '@/views/Dashboard.vue';
import Instances from '@/views/Instances.vue';
import Devices from '@/views/Devices.vue';
import Settings from '@/views/Settings.vue';
import { useAppStore } from '@/store';
import { api } from '@/api';

const store = useAppStore();
const currentView = ref('discover');

const fetchSystemStatus = async () => {
    try {
        const data = await api.getSystemStatus();
        store.setNodeRole(data.role);
    } catch (e) {
        console.error('Failed to fetch system status:', e);
    }
};

onMounted(() => {
    fetchSystemStatus();
});

const handleSwitchView = (view: string) => { currentView.value = view; };

const handleLogout = () => {
    store.logout();
    window.location.reload();
};
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-gray-900 text-gray-100">
    <!-- Sidebar -->
    <Sidebar @switch-view="handleSwitchView" @logout="handleLogout" />
    
    <!-- Main Content -->
    <div class="flex-1 flex flex-col overflow-hidden">
        <Discover v-if="currentView === 'discover'" />
        <Dashboard v-else-if="currentView === 'dashboard'" />
        <Instances v-else-if="currentView === 'instances'" />
        <Devices v-else-if="currentView === 'devices'" />
        <Settings v-else-if="currentView === 'settings'" />
    </div>
  </div>
</template>
