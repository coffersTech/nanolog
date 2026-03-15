<script setup lang="ts">
import { useAppStore } from '@/store';
import Login from '@/views/Login.vue';
import MainLayout from '@/views/MainLayout.vue';
import ToastContainer from '@/components/Toast.vue';

const store = useAppStore();
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
