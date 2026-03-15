<script setup lang="ts">
import { useToastStore } from '@/store/toast';

const toastStore = useToastStore();
</script>

<template>
  <div class="fixed bottom-6 right-6 z-[9999] flex flex-col space-y-3 pointer-events-none">
    <TransitionGroup name="toast">
      <div v-for="toast in toastStore.toasts" :key="toast.id" 
        class="min-w-[300px] max-w-md bg-gray-900 border border-gray-800 rounded-xl p-4 shadow-2xl flex items-center space-x-3 pointer-events-auto overflow-hidden relative">
        <div :class="{
          'bg-green-500': toast.type === 'success',
          'bg-red-500': toast.type === 'error',
          'bg-cyan-500': toast.type === 'info',
          'bg-yellow-500': toast.type === 'warning'
        }" class="w-1 h-full absolute left-0 top-0"></div>
        <p class="text-sm font-medium text-gray-200">{{ toast.message }}</p>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: all 0.3s ease; }
.toast-enter-from { opacity: 0; transform: translateX(30px); }
.toast-leave-to { opacity: 0; transform: scale(0.9); }
</style>
