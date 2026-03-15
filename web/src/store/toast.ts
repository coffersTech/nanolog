import { defineStore } from 'pinia';
import { ref } from 'vue';

export interface Toast {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info' | 'warning';
}

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([]);
  let nextId = 0;

  const showToast = (message: string, type: Toast['type'] = 'info') => {
    const id = nextId++;
    toasts.value.push({ id, message, type });
    setTimeout(() => {
      toasts.value = toasts.value.filter(t => t.id !== id);
    }, 3000);
  };

  return {
    toasts,
    showToast,
  };
});
