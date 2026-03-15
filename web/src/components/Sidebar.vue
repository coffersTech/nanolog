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
  Monitor
} from 'lucide-vue-next';

const store = useAppStore();
const showUserMenu = ref(false);

const navItems = [
  { view: 'discover', label: 'nav.discover', icon: Search, role: '!admin' },
  { view: 'dashboard', label: 'nav.dashboard', icon: LayoutDashboard, role: '!admin' },
  { view: 'instances', label: 'nav.instances', icon: Server, role: '!admin' },
  { view: 'settings', label: 'nav.settings', icon: Settings, role: '!engine' },
];

defineEmits(['switch-view', 'logout', 'change-password']);
</script>

<template>
  <aside class="w-64 bg-black flex flex-col border-r border-gray-800 shrink-0">
    <!-- Brand -->
    <div class="h-20 flex items-center justify-between px-6 border-b border-gray-800/50">
      <div class="flex items-center space-x-3">
        <div class="w-10 h-10 flex items-center justify-center">
          <img src="/logo.png" alt="NanoLog Logo" class="w-full h-full object-contain" />
        </div>
        <div>
          <h1 class="text-sm font-bold text-white tracking-wide uppercase">NanoLog</h1>
          <p class="text-xs text-gray-500 font-medium">Observability Hub</p>
        </div>
      </div>
      <button 
        @click="store.setLang(store.currentLang === 'en' ? 'zh' : 'en')"
        class="px-2 py-1 text-xs font-bold text-gray-400 hover:text-white border border-gray-700 hover:border-gray-600 rounded transition-all bg-gray-800/50"
      >
        {{ store.currentLang === 'en' ? 'EN' : '中文' }}
      </button>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 px-4 py-6 space-y-8 overflow-y-auto">
      <!-- Group: MONITOR -->
      <div>
        <h3 class="px-3 text-xs font-bold text-gray-600 uppercase tracking-[0.2em] mb-4">
          {{ store.t('nav.header_monitor') }}
        </h3>
        <div class="space-y-1">
          <a v-if="store.userRole !== 'admin'" href="#" @click.prevent="$emit('switch-view', 'discover')"
            class="group flex items-center space-x-3 px-3 py-2 rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50">
            <Search class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span class="text-sm font-medium">{{ store.t('nav.discover') }}</span>
          </a>
          <a v-if="store.userRole !== 'admin'" href="#" @click.prevent="$emit('switch-view', 'dashboard')"
            class="group flex items-center space-x-3 px-3 py-2 rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50">
            <LayoutDashboard class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span class="text-sm font-medium">{{ store.t('nav.dashboard') }}</span>
          </a>
          <a v-if="store.userRole !== 'admin'" href="#" @click.prevent="$emit('switch-view', 'instances')"
            class="group flex items-center space-x-3 px-3 py-2 rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50">
            <Server class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span class="text-sm font-medium">{{ store.t('nav.instances') }}</span>
          </a>
          <a v-if="store.userRole !== 'admin'" href="#" @click.prevent="$emit('switch-view', 'devices')"
            class="group flex items-center space-x-3 px-3 py-2 rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50">
            <Monitor class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span class="text-sm font-medium">{{ store.t('nav.devices') }}</span>
          </a>
        </div>
      </div>

      <!-- Group: SYSTEM -->
      <div>
        <h3 class="px-3 text-xs font-bold text-gray-600 uppercase tracking-[0.2em] mb-4">
          {{ store.t('nav.header_system') }}
        </h3>
        <div class="space-y-1">
          <a v-if="store.userRole !== 'engine'" href="#" @click.prevent="$emit('switch-view', 'settings')"
            class="group flex items-center space-x-3 px-3 py-2 rounded-lg transition-all text-gray-400 hover:text-gray-200 hover:bg-gray-800/50">
            <Settings class="w-5 h-5 flex-shrink-0 text-gray-500 group-hover:text-gray-300" />
            <span class="text-sm font-medium">{{ store.t('nav.settings') }}</span>
          </a>
        </div>
      </div>
    </nav>

    <!-- Footer Status -->
    <div class="border-t border-gray-800 bg-gray-950/20 p-3">
      <!-- Current User Info with Dropdown -->
      <div class="relative">
        <button @click="showUserMenu = !showUserMenu"
          class="w-full flex items-center space-x-3 px-2 py-2 rounded-lg hover:bg-gray-800/50 transition-all cursor-pointer">
          <div class="w-8 h-8 bg-cyan-500/20 rounded-lg flex items-center justify-center">
            <UserIcon class="w-4 h-4 text-cyan-400" />
          </div>
          <div class="flex-1 min-w-0 text-left">
            <p class="text-sm font-medium text-white truncate">{{ store.currentUser }}</p>
            <span :class="store.userRole === 'super_admin' ? 'text-purple-400' : 'text-cyan-400'"
              class="text-xs font-bold uppercase">{{ store.userRole }}</span>
          </div>
          <ChevronUp class="w-4 h-4 text-gray-500 transition-transform" :class="showUserMenu ? 'rotate-180' : ''" />
        </button>

        <!-- Dropdown Menu -->
        <div v-if="showUserMenu"
          class="absolute bottom-full left-0 right-0 mb-2 bg-gray-900 border border-gray-800 rounded-xl shadow-xl overflow-hidden z-50">
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

      <!-- Status Info -->
      <div class="flex items-center space-x-3 px-2 py-2 border-t border-gray-800/50 pt-3 mt-1">
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
