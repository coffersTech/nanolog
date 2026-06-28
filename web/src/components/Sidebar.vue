<script setup lang="ts">
import { ref } from 'vue';
import { useAppStore } from '@/store';
import { 
  LayoutDashboard, 
  Search, 
  Server, 
  Settings, 
  User as UserIcon, 
  Key, 
  LogOut,
  ChevronUp,
  ChevronLeft,
  ChevronRight,
  Monitor
} from 'lucide-vue-next';

const store = useAppStore();
const showUserMenu = ref(false);
const isCollapsed = ref(false);

const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value;
};

const navItems = [
  { view: 'discover', label: 'nav.discover', icon: Search, role: '!admin', shortLabel: '搜索' },
  { view: 'dashboard', label: 'nav.dashboard', icon: LayoutDashboard, role: '!admin', shortLabel: '面板' },
  { view: 'instances', label: 'nav.instances', icon: Server, role: '!admin', shortLabel: '实例' },
  { view: 'settings', label: 'nav.settings', icon: Settings, role: '!engine', shortLabel: '设置' },
];

defineEmits(['switch-view', 'logout', 'change-password']);
</script>

<template>
  <aside 
    :class="[
      'bg-black flex flex-col border-r border-gray-800 shrink-0 transition-all duration-300 ease-in-out',
      isCollapsed ? 'w-18' : 'w-64'
    ]"
  >
    <!-- Brand -->
    <div :class="[
      'h-20 border-b border-gray-800/50 flex items-center',
      isCollapsed ? 'justify-center px-2' : 'px-6'
    ]">
      <div :class="isCollapsed ? '' : 'flex items-center space-x-3'">
        <div class="w-10 h-10 flex items-center justify-center">
          <img src="/logo.png" alt="NanoLog Logo" class="w-full h-full object-contain" />
        </div>
        <div v-if="!isCollapsed">
          <h1 class="text-sm font-bold text-white tracking-wide uppercase">NanoLog</h1>
          <p class="text-xs text-gray-500 font-medium">Observability Hub</p>
        </div>
      </div>
    </div>

    <!-- Navigation -->
    <nav :class="[
      'flex-1 overflow-y-auto transition-all duration-300',
      isCollapsed ? 'px-2 py-4' : 'px-4 py-6'
    ]">
      <!-- Group: MONITOR -->
      <div>
        <h3 v-if="!isCollapsed" class="px-3 text-xs font-bold text-gray-600 uppercase tracking-[0.2em] mb-4">
          {{ store.t('nav.header_monitor') }}
        </h3>
        <div :class="isCollapsed ? 'space-y-2' : 'space-y-1'">
          <a v-if="store.userRole !== 'admin'" href="#" @click.prevent="$emit('switch-view', 'discover')"
            :class="[
              'group rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50',
              isCollapsed ? 'flex flex-col items-center justify-center px-1 py-2' : 'flex items-center px-3 py-2 space-x-3'
            ]">
            <Search class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span :class="[
              'transition-all',
              isCollapsed ? 'text-[9px] text-center truncate max-w-full mt-1' : 'text-sm font-medium'
            ]">{{ isCollapsed ? '搜索' : store.t('nav.discover') }}</span>
          </a>
          <a v-if="store.userRole !== 'admin'" href="#" @click.prevent="$emit('switch-view', 'dashboard')"
            :class="[
              'group rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50',
              isCollapsed ? 'flex flex-col items-center justify-center px-1 py-2' : 'flex items-center px-3 py-2 space-x-3'
            ]">
            <LayoutDashboard class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span :class="[
              'transition-all',
              isCollapsed ? 'text-[9px] text-center truncate max-w-full mt-1' : 'text-sm font-medium'
            ]">{{ isCollapsed ? '面板' : store.t('nav.dashboard') }}</span>
          </a>
          <a v-if="store.userRole !== 'admin'" href="#" @click.prevent="$emit('switch-view', 'instances')"
            :class="[
              'group rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50',
              isCollapsed ? 'flex flex-col items-center justify-center px-1 py-2' : 'flex items-center px-3 py-2 space-x-3'
            ]">
            <Server class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span :class="[
              'transition-all',
              isCollapsed ? 'text-[9px] text-center truncate max-w-full mt-1' : 'text-sm font-medium'
            ]">{{ isCollapsed ? '实例' : store.t('nav.instances') }}</span>
          </a>
          <a v-if="store.userRole !== 'admin'" href="#" @click.prevent="$emit('switch-view', 'devices')"
            :class="[
              'group rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50',
              isCollapsed ? 'flex flex-col items-center justify-center px-1 py-2' : 'flex items-center px-3 py-2 space-x-3'
            ]">
            <Monitor class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span :class="[
              'transition-all',
              isCollapsed ? 'text-[9px] text-center truncate max-w-full mt-1' : 'text-sm font-medium'
            ]">{{ isCollapsed ? '设备' : store.t('nav.devices') }}</span>
          </a>
        </div>
      </div>

      <!-- Group: SYSTEM -->
      <div :class="isCollapsed ? 'mt-6' : ''">
        <h3 v-if="!isCollapsed" class="px-3 text-xs font-bold text-gray-600 uppercase tracking-[0.2em] mb-4">
          {{ store.t('nav.header_system') }}
        </h3>
        <div :class="isCollapsed ? 'space-y-2' : 'space-y-1'">
          <a v-if="store.userRole !== 'engine'" href="#" @click.prevent="$emit('switch-view', 'settings')"
            :class="[
              'group rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50',
              isCollapsed ? 'flex flex-col items-center justify-center px-1 py-2' : 'flex items-center px-3 py-2 space-x-3'
            ]">
            <Settings class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span :class="[
              'transition-all',
              isCollapsed ? 'text-[9px] text-center truncate max-w-full mt-1' : 'text-sm font-medium'
            ]">{{ isCollapsed ? '设置' : store.t('nav.settings') }}</span>
          </a>
        </div>
      </div>
    </nav>

    <!-- Footer -->
    <div class="border-t border-gray-800 bg-gray-950/20">
      <!-- Current User Info with Dropdown -->
      <div class="relative">
        <button @click="showUserMenu = !showUserMenu"
          :class="[
            'w-full flex items-center rounded-lg hover:bg-gray-800/50 transition-all cursor-pointer',
            isCollapsed ? 'justify-center px-2 py-2' : 'space-x-3 px-3 py-2'
          ]">
          <div class="w-8 h-8 bg-cyan-500/20 rounded-lg flex items-center justify-center">
            <UserIcon class="w-4 h-4 text-cyan-400" />
          </div>
          <div v-if="!isCollapsed" class="flex-1 min-w-0 text-left">
            <p class="text-sm font-medium text-white truncate">{{ store.currentUser }}</p>
            <span :class="store.userRole === 'super_admin' ? 'text-purple-400' : 'text-cyan-400'"
              class="text-xs font-bold uppercase">{{ store.userRole }}</span>
          </div>
          <ChevronUp v-if="!isCollapsed" class="w-4 h-4 text-gray-500 transition-transform" :class="showUserMenu ? 'rotate-180' : ''" />
        </button>

        <!-- Dropdown Menu -->
        <div v-if="showUserMenu"
          :class="[
            'bg-gray-900 border border-gray-800 rounded-xl shadow-xl overflow-hidden z-50',
            isCollapsed ? 'absolute bottom-full left-1/2 -translate-x-1/2 mb-2 w-48' : 'absolute bottom-full left-0 right-0 mb-2'
          ]">
          <button @click="$emit('change-password'); showUserMenu = false"
            class="w-full flex items-center space-x-3 px-4 py-3 text-gray-400 hover:text-cyan-400 hover:bg-gray-800/50 transition-all">
            <Key class="w-4 h-4" />
            <span class="text-sm font-medium">{{ store.t('nav.change_password') }}</span>
          </button>
          <div class="border-t border-gray-800"></div>
          <button @click="$emit('logout'); showUserMenu = false"
            class="w-full flex items-center space-x-3 px-4 py-3 text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-all">
            <LogOut class="w-4 h-4" />
            <span class="text-sm font-medium">{{ store.t('nav.logout') }}</span>
          </button>
        </div>
      </div>

      <!-- Toolbar: Language Switch + Collapse Toggle -->
      <div :class="[
        'flex items-center justify-between px-3 py-3 border-t border-gray-800/50',
        isCollapsed ? 'justify-center' : ''
      ]">
        <button v-if="!isCollapsed"
          @click="store.setLang(store.currentLang === 'en' ? 'zh' : 'en')"
          class="px-3 py-1.5 text-xs font-bold text-gray-400 hover:text-white border border-gray-700 hover:border-gray-600 rounded transition-all bg-gray-800/50"
        >
          {{ store.currentLang === 'en' ? 'EN' : '中文' }}
        </button>
        <button 
          @click="toggleCollapse"
          class="p-2 text-gray-500 hover:text-white border border-gray-800 hover:border-gray-700 rounded-lg transition-all hover:bg-gray-800/50"
          :title="isCollapsed ? (store.currentLang === 'en' ? 'Expand' : '展开') : (store.currentLang === 'en' ? 'Collapse' : '收缩')"
        >
          <ChevronRight v-if="isCollapsed" class="w-5 h-5" />
          <ChevronLeft v-else class="w-5 h-5" />
        </button>
      </div>

      <!-- Status Info -->
      <div v-if="!isCollapsed" class="flex items-center space-x-3 px-3 py-3 border-t border-gray-800/50">
          <div class="relative">
              <div class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
              <div class="absolute inset-0 w-2 h-2 bg-green-500 rounded-full animate-ping opacity-75"></div>
          </div>
          <div class="flex-1 min-w-0 text-left">
              <p class="text-[10px] font-bold text-gray-400 uppercase tracking-tighter">{{ store.t('common.system_status') }}</p>
              <p class="text-[10px] text-green-500 font-medium truncate">{{ store.t('common.cluster_online') }}</p>
          </div>
          <span class="text-[10px] font-mono text-gray-600">{{ store.systemVersion }}</span>
      </div>
    </div>
  </aside>
</template>
