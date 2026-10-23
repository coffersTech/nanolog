<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useAppStore } from '@/store';
import { ChevronDown, X } from 'lucide-vue-next';

const store = useAppStore();
const isOpen = ref(false);
const selectedLevel = ref('');
const dropdownRef = ref<HTMLElement | null>(null);

const emit = defineEmits(['select']);

interface LevelOption {
  key: string;
  label: string;
  colorClass: string;
  badgeBg: string;
  textClass: string;
  borderClass: string;
}

const levels: LevelOption[] = [
  { key: 'DEBUG', label: 'DEBUG', colorClass: 'bg-gray-400', badgeBg: 'bg-gray-500/10', textClass: 'text-gray-300', borderClass: 'border-gray-500/50' },
  { key: 'INFO', label: 'INFO', colorClass: 'bg-blue-400', badgeBg: 'bg-blue-500/10', textClass: 'text-blue-400', borderClass: 'border-blue-500/50' },
  { key: 'WARN', label: 'WARN', colorClass: 'bg-amber-400', badgeBg: 'bg-amber-500/10', textClass: 'text-amber-400', borderClass: 'border-amber-500/50' },
  { key: 'ERROR', label: 'ERROR', colorClass: 'bg-rose-500', badgeBg: 'bg-rose-500/10', textClass: 'text-rose-400', borderClass: 'border-rose-500/50' },
  { key: 'FATAL', label: 'FATAL', colorClass: 'bg-purple-500', badgeBg: 'bg-purple-500/10', textClass: 'text-purple-400', borderClass: 'border-purple-500/50' },
];

const currentLevelOption = computed(() => {
  return levels.find(l => l.key === selectedLevel.value) || null;
});

const handleSelect = (level: string) => {
  selectedLevel.value = level;
  isOpen.value = false;
  emit('select', level);
};

const handleClear = () => {
  selectedLevel.value = '';
  isOpen.value = false;
  emit('select', '');
};

const handleClickOutside = (event: MouseEvent) => {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    isOpen.value = false;
  }
};

onMounted(() => {
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
        'flex items-center space-x-2 px-3 py-2 bg-gray-800 border rounded-lg text-sm transition-all',
        currentLevelOption 
          ? `${currentLevelOption.textClass} ${currentLevelOption.borderClass}` 
          : 'text-gray-400 border-gray-700 hover:text-gray-200 hover:border-gray-600'
      ]"
    >
      <span v-if="currentLevelOption" class="w-2 h-2 rounded-full shrink-0" :class="currentLevelOption.colorClass"></span>
      <span class="font-medium">
        {{ currentLevelOption ? currentLevelOption.label : store.t('search.all_levels') }}
      </span>
      <ChevronDown class="w-4 h-4 transition-transform" :class="isOpen ? 'rotate-180' : ''" />
    </button>

    <!-- Dropdown -->
    <div 
      v-if="isOpen"
      class="absolute top-full left-0 mt-2 w-48 bg-gray-900/95 backdrop-blur-xl border border-gray-700 rounded-xl shadow-2xl z-50 overflow-hidden"
    >
      <!-- All Levels Option -->
      <button 
        @click.stop="handleClear()"
        :class="[
          'w-full flex items-center justify-between px-4 py-3 text-sm transition-all border-b border-gray-800',
          !selectedLevel ? 'text-cyan-400 bg-cyan-500/10' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
        ]"
      >
        <span>{{ store.t('search.all_levels') }}</span>
        <X v-if="selectedLevel" class="w-4 h-4 text-gray-400 hover:text-white" />
      </button>

      <!-- Level List -->
      <div class="py-1">
        <button 
          v-for="lvl in levels"
          :key="lvl.key"
          @click.stop="handleSelect(lvl.key)"
          :class="[
            'w-full flex items-center space-x-2.5 px-4 py-2.5 text-sm transition-all',
            selectedLevel === lvl.key ? `${lvl.textClass} ${lvl.badgeBg}` : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
          ]"
        >
          <span class="w-2 h-2 rounded-full shrink-0" :class="lvl.colorClass"></span>
          <span class="font-medium tracking-wide">{{ lvl.label }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
