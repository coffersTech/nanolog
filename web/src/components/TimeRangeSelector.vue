<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue';
import { useAppStore } from '@/store';
import { Clock, ChevronDown, MoveRight, Check } from 'lucide-vue-next';
import flatpickr from 'flatpickr';
import 'flatpickr/dist/flatpickr.min.css';
import { Mandarin } from 'flatpickr/dist/l10n/zh.js';

const store = useAppStore();
const isOpen = ref(false);
const selectedLabel = ref('search.15m');
const selectedValue = ref('15m');

const startTime = ref('');
const endTime = ref('');
const autoRefreshInterval = ref('off');
const showRecentHistory = ref(false);

const recentCustomRanges = ref<any[]>(JSON.parse(localStorage.getItem('nanolog_recent_ranges') || '[]'));

let startPicker: any = null;
let endPicker: any = null;

const pad = (n: number) => String(n).padStart(2, '0');

const ranges = [
  { value: '5m', label: 'search.5m' },
  { value: '15m', label: 'search.15m' },
  { value: '30m', label: 'search.30m' },
  { value: '1h', label: 'search.1h' },
  { value: '6h', label: 'search.6h' },
  { value: '1d', label: 'search.1d' },
  { value: '3d', label: 'search.3d' },
  { value: '7d', label: 'search.7d' },
  { value: '30d', label: 'search.30d' },
  { value: '90d', label: 'search.90d' },
  { value: 'today', label: 'search.today' },
  { value: 'yesterday', label: 'search.yesterday' },
  { value: 'day_before_yesterday', label: 'search.day_before_yesterday' },
  { value: 'this_week', label: 'search.this_week' },
  { value: 'last_week', label: 'search.last_week' },
  { value: 'this_month', label: 'search.this_month' },
  { value: 'last_month', label: 'search.last_month' },
];

const refreshRates = [
  { value: 'off', label: 'search.refresh_off' },
  { value: '5s', label: 'search.refresh_5s' },
  { value: '15s', label: 'search.refresh_15s' },
  { value: '1m', label: 'search.refresh_1m' },
  { value: '5m', label: 'search.refresh_5m' },
  { value: '10m', label: 'search.refresh_10m' },
  { value: '30m', label: 'search.refresh_30m' },
];

const emit = defineEmits(['update', 'auto-refresh']);

const formatDefaultTimes = () => {
  const now = new Date();
  const fifteenMinAgo = new Date(now.getTime() - 15 * 60 * 1000);
  
  const fmt = (d: Date) => {
    const year = d.getFullYear();
    const month = pad(d.getMonth() + 1);
    const day = pad(d.getDate());
    const hours = pad(d.getHours());
    const minutes = pad(d.getMinutes());
    return `${year}-${month}-${day} ${hours}:${minutes}`;
  };
  
  startTime.value = fmt(fifteenMinAgo);
  endTime.value = fmt(now);
};

const selectRange = (range: any) => {
  selectedLabel.value = range.label;
  selectedValue.value = range.value;
  emit('update', range.value);
  isOpen.value = false;
};

const applyCustomRange = () => {
  // Convert milliseconds from Date.getTime() to nanoseconds for the backend/Discover logic
  const start = new Date(startTime.value).getTime() * 1000000;
  const end = new Date(endTime.value).getTime() * 1000000;
  
  if (isNaN(start) || isNaN(end) || start >= end) {
    store.addToast(store.t('alerts.invalid_time_range'), 'error');
    return;
  }
  
  const range = { start, end };
  selectedLabel.value = 'search.custom';
  selectedValue.value = 'custom';
  
  // Update history
  const historyItem = { start: startTime.value, end: endTime.value };
  recentCustomRanges.value = [historyItem, ...recentCustomRanges.value.filter(r => r.start !== historyItem.start || r.end !== historyItem.end)].slice(0, 3);
  localStorage.setItem('nanolog_recent_ranges', JSON.stringify(recentCustomRanges.value));
  
  emit('update', range);
  isOpen.value = false;
};

const selectRecentCustom = (range: any) => {
  startTime.value = range.start;
  endTime.value = range.end;
  applyCustomRange();
};

watch(autoRefreshInterval, (val) => {
  emit('auto-refresh', val);
});

onMounted(() => {
  formatDefaultTimes();
  
  const pickerConfig: any = {
    enableTime: true,
    time_24hr: true,
    dateFormat: "Y-m-d H:i",
    disableMobile: "true",
    locale: store.currentLang === 'zh' ? Mandarin : undefined,
    onClose: (selectedDates: Date[], dateStr: string, instance: any) => {
        if (instance.element.id === 'start-picker') startTime.value = dateStr;
        else endTime.value = dateStr;
    }
  };

  startPicker = flatpickr("#start-picker", pickerConfig);
  endPicker = flatpickr("#end-picker", pickerConfig);
});

const getLocalTimezone = () => {
  const offset = new Date().getTimezoneOffset();
  const absOffset = Math.abs(offset);
  const hours = pad(Math.floor(absOffset / 60));
  const minutes = pad(absOffset % 60);
  const sign = offset <= 0 ? '+' : '-';
  return `UTC${sign}${hours}:${minutes}`;
};

const timezoneLabel = getLocalTimezone();

const containerRef = ref<HTMLElement | null>(null);

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement;
  
  // Don't close if clicking inside the container
  if (containerRef.value && containerRef.value.contains(target)) {
    return;
  }

  // Don't close if clicking inside a flatpickr calendar (it's often appended to body)
  if (target.closest('.flatpickr-calendar')) {
    return;
  }

  isOpen.value = false;
};

onMounted(() => {
  window.addEventListener('click', handleClickOutside);
  formatDefaultTimes();
  
  const pickerConfig: any = {
    enableTime: true,
    time_24hr: true,
    dateFormat: "Y-m-d H:i",
    disableMobile: "true",
    locale: store.currentLang === 'zh' ? Mandarin : undefined,
    onClose: (selectedDates: Date[], dateStr: string, instance: any) => {
        if (instance.element.id === 'start-picker') startTime.value = dateStr;
        else endTime.value = dateStr;
    }
  };

  startPicker = flatpickr("#start-picker", pickerConfig);
  endPicker = flatpickr("#end-picker", pickerConfig);
});

onUnmounted(() => {
    window.removeEventListener('click', handleClickOutside);
    if (startPicker) startPicker.destroy();
    if (endPicker) endPicker.destroy();
});
</script>

<template>
  <div class="relative" ref="containerRef">
    <button @click="isOpen = !isOpen"
      class="flex items-center space-x-2 px-3 py-2 bg-gray-900/50 border border-gray-800 rounded-lg text-sm text-gray-300 hover:border-gray-700 transition-all hover:bg-gray-800/80">
      <Clock class="w-4 h-4 text-gray-400" />
      <span class="font-medium whitespace-nowrap">{{ store.t(selectedLabel) }}</span>
      <ChevronDown class="w-3.5 h-3.5 text-gray-500 transition-transform duration-300" :class="isOpen ? 'rotate-180' : ''" />
    </button>

    <!-- Dropdown Content -->
    <div v-show="isOpen"
      class="absolute top-full left-0 mt-3 w-[640px] bg-gray-900 border border-gray-800 rounded-2xl shadow-[0_20px_50px_rgba(0,0,0,0.5)] z-[100] p-6 backdrop-blur-xl bg-opacity-95">
      
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-white font-bold text-base">{{ store.t('search.time_range') }}</h3>
        <span class="text-[10px] text-gray-500 font-mono bg-gray-800/50 px-2.5 py-1 rounded-full border border-gray-700/50 uppercase tracking-wider">
          {{ store.t('search.tz_local') }} ({{ timezoneLabel }})
        </span>
      </div>

      <!-- Presets Grid -->
      <div class="grid grid-cols-5 gap-2 mb-8">
        <button v-for="range in ranges" :key="range.value"
          @click="selectRange(range)"
          class="px-2 py-2 rounded-lg text-[11px] font-bold transition-all duration-200 text-center"
          :class="selectedValue === range.value 
            ? 'bg-cyan-600 text-white shadow-[0_0_20px_rgba(6,182,212,0.3)] border border-cyan-500' 
            : 'bg-gray-800/40 text-gray-400 hover:text-gray-200 hover:bg-gray-800 border border-gray-700/50 hover:border-gray-600'">
          {{ store.t(range.label) }}
        </button>
      </div>

      <!-- Custom Range Divider -->
      <div class="h-px bg-gray-800/80 mb-8"></div>

      <!-- Custom Input Section -->
      <div class="flex items-end space-x-4 mb-8">
        <div class="flex-1 space-y-2">
          <label class="text-[10px] font-bold text-gray-500 uppercase tracking-widest px-1">{{ store.t('search.start_time') }}</label>
          <input type="text" v-model="startTime" id="start-picker"
            class="w-full bg-gray-950 border border-gray-800 rounded-xl px-3 py-2.5 text-xs text-gray-200 font-mono focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/20 transition-all" />
        </div>
        <div class="pb-3.5">
          <MoveRight class="w-5 h-5 text-gray-700" />
        </div>
        <div class="flex-1 space-y-2">
          <label class="text-[10px] font-bold text-gray-500 uppercase tracking-widest px-1">{{ store.t('search.end_time') }}</label>
          <input type="text" v-model="endTime" id="end-picker"
            class="w-full bg-gray-950 border border-gray-800 rounded-xl px-3 py-2.5 text-xs text-gray-200 font-mono focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/20 transition-all" />
        </div>
        <button @click="applyCustomRange"
          class="h-[38px] px-6 rounded-xl bg-cyan-600 text-white text-xs font-black hover:bg-cyan-500 transition-all active:scale-95 shadow-lg shadow-cyan-600/20">
          {{ store.t('common.confirm') }}
        </button>
      </div>

      <!-- Footer Section (Recent & Refresh) -->
      <div class="space-y-4 pt-6 border-t border-gray-800/80">
        <div class="flex items-center justify-between">
          <button @click="showRecentHistory = !showRecentHistory"
            class="flex items-center space-x-2 text-xs font-medium text-gray-400 hover:text-cyan-400 transition-colors group">
            <span>{{ store.t('search.recent_custom') }}</span>
            <ChevronDown class="w-3.5 h-3.5 transition-transform" :class="showRecentHistory ? 'rotate-180' : ''" />
          </button>

          <div class="flex items-center space-x-4">
            <span class="text-xs font-medium text-gray-500 italic">{{ store.t('search.page_auto_refresh') }}</span>
            <select v-model="autoRefreshInterval"
              class="bg-gray-950 border border-gray-800 rounded-lg px-3 py-1.5 text-xs text-gray-300 font-medium focus:outline-none focus:border-cyan-500/50">
              <option v-for="rate in refreshRates" :key="rate.value" :value="rate.value">
                {{ store.t(rate.label) }}
              </option>
            </select>
          </div>
        </div>

        <!-- Recent History List -->
        <div v-if="showRecentHistory" class="space-y-1.5 animate-in slide-in-from-top-2 duration-200">
            <div v-if="recentCustomRanges.length === 0" class="text-[10px] text-gray-600 italic px-2 py-2">
                No recent custom ranges found.
            </div>
            <button v-for="(r, idx) in recentCustomRanges" :key="idx"
                @click="selectRecentCustom(r)"
                class="w-full text-left px-3 py-2 rounded-lg bg-gray-950/30 border border-gray-800/50 hover:border-cyan-500/30 hover:bg-cyan-500/5 text-[11px] text-gray-400 hover:text-gray-200 transition-all font-mono group flex items-center justify-between">
                <span>{{ r.start }} ~ {{ r.end }}</span>
                <Check class="w-3 h-3 text-cyan-500 opacity-0 group-hover:opacity-100 transition-opacity" />
            </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Glassmorphism effect for the dropdown */
.backdrop-blur-xl {
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
}

/* Flatpickr Dark Theme Customization */
:deep(.flatpickr-calendar) {
    background: #0f172a !important; /* gray-900 */
    border: 1px solid rgba(255, 255, 255, 0.1) !important;
    box-shadow: 0 10px 30px -5px rgba(0, 0, 0, 0.5) !important;
    border-radius: 12px !important;
}

:deep(.flatpickr-day) {
    color: #f1f5f9 !important; /* slate-100 */
}

:deep(.flatpickr-day.flatpickr-disabled),
:deep(.flatpickr-day.flatpickr-disabled:hover),
:deep(.flatpickr-day.prevMonthDay),
:deep(.flatpickr-day.nextMonthDay) {
    color: #475569 !important; /* slate-500 */
}

:deep(.flatpickr-day.selected) {
    background: #0891b2 !important; /* cyan-600 */
    border-color: #0891b2 !important;
    color: #fff !important;
}

:deep(.flatpickr-day:hover) {
    background: rgba(8, 145, 178, 0.2) !important;
}

:deep(.flatpickr-weekday) {
    color: #94a3b8 !important; /* slate-400 */
    font-weight: 600 !important;
}

:deep(.flatpickr-time input),
:deep(.flatpickr-time .flatpickr-am-pm) {
    color: #fff !important;
}

:deep(.flatpickr-time .flatpickr-time-separator) {
    color: #94a3b8 !important;
}

:deep(.flatpickr-calendar.hasTime .flatpickr-time) {
    height: 44px !important;
    border-top: 1px solid rgba(255, 255, 255, 0.1) !important;
}

:deep(.flatpickr-months .flatpickr-month),
:deep(.flatpickr-current-month .flatpickr-monthDropdown-months),
:deep(.numInput.cur-year) {
    color: #fff !important;
    fill: #fff !important;
    font-weight: 700 !important;
}

:deep(.flatpickr-months .flatpickr-prev-month),
:deep(.flatpickr-months .flatpickr-next-month) {
    color: #fff !important;
    fill: #fff !important;
}

:deep(.flatpickr-time input:hover),
:deep(.flatpickr-time .flatpickr-am-pm:hover),
:deep(.flatpickr-time input:focus),
:deep(.flatpickr-time .flatpickr-am-pm:focus) {
    background: rgba(255, 255, 255, 0.1) !important;
}
</style>
