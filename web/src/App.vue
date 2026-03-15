<script setup lang="ts">
import { onMounted } from 'vue';
import { useAppStore } from '@/store';
import { api } from '@/api';
import Login from '@/views/Login.vue';
import MainLayout from '@/views/MainLayout.vue';
import ToastContainer from '@/components/Toast.vue';

const store = useAppStore();

const fetchSystemStatus = async () => {
    try {
        const data = await api.getSystemStatus();
        store.setNodeRole(data.node_role, data.version);
    } catch (e) {
        console.error('Failed to fetch system status:', e);
    }
};

onMounted(() => {
    fetchSystemStatus();
});
</script>

<template>
  <div class="h-screen overflow-hidden bg-gray-900 text-gray-100">
    <Login v-if="!store.isAuthenticated" />
    <MainLayout v-else />
    <ToastContainer />
  </div>
</template>

<style>
/* App specific styles */
</style>
