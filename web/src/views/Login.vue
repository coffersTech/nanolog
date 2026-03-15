<script setup lang="ts">
import { ref, computed } from 'vue';
import { useAppStore, useToastStore } from '@/store';
import { api } from '@/api';
import { Shield, Lock, User as UserIcon, Loader2, Languages, ArrowRight, Eye, EyeOff } from 'lucide-vue-next';

const store = useAppStore();
const toastStore = useToastStore();
const loginForm = ref({ username: store.lastUsername, password: '', remember: true });
const showPassword = ref(false);
const loading = ref(false);
const error = ref('');

// Better reactive translation bridge
const t = computed(() => store.t);
const currentLang = computed(() => store.currentLang);

const toggleLang = () => {
    const nextLang = store.currentLang === 'zh' ? 'en' : 'zh';
    store.setLang(nextLang);
};

const handleLogin = async () => {
    if (loading.value) return;
    loading.value = true;
    error.value = '';
    try {
        const data = await api.login({ username: loginForm.value.username, password: loginForm.value.password });
        store.setAuth(data.token, data.username, data.role, loginForm.value.remember);
        toastStore.showToast(store.t('common.success'), 'success');
    } catch (e: any) {
        const msg = e.message || store.t('alerts.login_failed');
        error.value = msg;
        toastStore.showToast(msg, 'error');
    } finally {
        loading.value = false;
    }
};
</script>

<template>
  <div class="min-h-screen w-full flex flex-col lg:flex-row bg-[#020617] text-slate-200 selection:bg-cyan-500/30 overflow-x-hidden">
    <!-- Fixed Noise Texture Overlay -->
    <div class="fixed inset-0 pointer-events-none z-[100] opacity-[0.03] mix-blend-overlay bg-[url('https://grainy-gradients.vercel.app/noise.svg')]"></div>
    
    <!-- Language Switcher - Increased prominence and better accessibility -->
    <button 
        @click.stop="toggleLang" 
        type="button"
        class="fixed top-6 right-6 z-[110] flex items-center space-x-2 px-5 py-2.5 bg-slate-900/60 hover:bg-slate-800/80 backdrop-blur-xl border border-slate-700/50 rounded-full transition-all duration-300 group shadow-2xl active:scale-95"
    >
        <Languages class="w-4 h-4 text-cyan-400 group-hover:rotate-12 transition-transform duration-500" />
        <span class="text-[11px] font-black uppercase tracking-[0.2em] text-slate-300 group-hover:text-white">
            {{ currentLang === 'zh' ? 'En' : '中文' }}
        </span>
    </button>

    <!-- Left Side: Branding & Immersive Experience -->
    <div class="relative w-full lg:w-[60%] xl:w-[65%] flex flex-col justify-start p-8 lg:p-24 pt-20 lg:pt-32 border-b lg:border-b-0 lg:border-r border-slate-800/20 min-h-[60vh] lg:min-h-screen">
      <!-- Animated Mesh Gradient Background -->
      <div class="absolute inset-0 overflow-hidden pointer-events-none">
        <div class="absolute top-[-10%] left-[-5%] w-[60%] h-[60%] bg-blue-600/10 rounded-full blur-[120px] animate-pulse"></div>
        <div class="absolute bottom-[-10%] right-[-5%] w-[50%] h-[50%] bg-cyan-500/10 rounded-full blur-[100px] animate-pulse" style="animation-delay: 2s;"></div>
      </div>

      <!-- Content -->
      <div class="relative z-10 max-w-2xl animate-in fade-in slide-in-from-left-8 duration-1000">
        <div class="flex items-center space-x-5 mb-10">
            <div class="relative group cursor-pointer">
                <div class="absolute -inset-2 bg-gradient-to-tr from-cyan-500/20 to-blue-600/20 rounded-2xl blur-lg group-hover:opacity-100 transition duration-700 opacity-50"></div>
            <div class="relative w-16 h-16 flex items-center justify-center overflow-hidden rounded-2xl border border-slate-800/10">
                <img src="/logo.png" alt="NanoLog" class="w-full h-full object-contain transform group-hover:scale-110 transition-transform duration-500" />
            </div>
            </div>
            <div class="h-10 w-px bg-slate-800/50"></div>
            <span class="text-3xl font-black tracking-tighter text-white uppercase italic">Nano<span class="text-cyan-500">Log</span></span>
        </div>

        <h1 class="text-5xl lg:text-7xl font-black text-white leading-[1.1] mb-6 tracking-tight">
          <span class="relative inline-block mt-4">
            <span class="bg-clip-text text-transparent bg-gradient-to-r from-cyan-400 via-blue-500 to-indigo-500 italic whitespace-nowrap">{{ t('auth.adv_intel') }}</span>
            <div class="absolute -bottom-2 left-0 w-3/4 h-1.5 bg-cyan-500/30 rounded-full blur-[2px]"></div>
          </span>
        </h1>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mt-12">
            <div class="group p-8 rounded-3xl bg-slate-900/20 border border-slate-800/30 hover:border-cyan-500/30 transition-all duration-500 backdrop-blur-sm">
                <div class="w-12 h-12 rounded-2xl bg-cyan-500/10 flex items-center justify-center mb-6 group-hover:bg-cyan-500/20 transition-colors">
                    <Shield class="w-6 h-6 text-cyan-400" />
                </div>
                <h3 class="text-xl font-bold text-white mb-3">{{ t('auth.feature_security_title') }}</h3>
                <p class="text-slate-500 text-sm leading-relaxed font-medium">{{ t('auth.feature_security_desc') }}</p>
            </div>

            <div class="group p-8 rounded-3xl bg-slate-900/20 border border-slate-800/30 hover:border-blue-500/30 transition-all duration-500 backdrop-blur-sm">
                <div class="w-12 h-12 rounded-2xl bg-blue-500/10 flex items-center justify-center mb-6 group-hover:bg-blue-500/20 transition-colors">
                    <Loader2 class="w-6 h-6 text-blue-400" />
                </div>
                <h3 class="text-xl font-bold text-white mb-3 italic">{{ t('auth.feature_perf_title') }}</h3>
                <p class="text-slate-500 text-sm leading-relaxed font-medium">{{ t('auth.feature_perf_desc') }}</p>
            </div>
        </div>

        <div class="mt-12 flex items-center space-x-12 opacity-40">
            <div class="flex flex-col">
                <span class="text-3xl font-black text-white italic tracking-tighter">{{ store.systemVersion }}</span>
                <span class="text-[10px] uppercase tracking-[0.2em] font-black text-slate-500">{{ t('auth.release_status') }}</span>
            </div>
            <div class="h-10 w-px bg-slate-800"></div>
            <div class="flex flex-col">
                <span class="text-3xl font-black text-white italic tracking-tighter">99.9%</span>
                <span class="text-[10px] uppercase tracking-[0.2em] font-black text-slate-500">{{ t('auth.uptime_rel') }}</span>
            </div>
        </div>
      </div>
    </div>

    <!-- Right Side: Elevated Login UI -->
    <div class="relative w-full lg:w-[40%] xl:w-[35%] flex flex-col justify-start p-8 lg:p-16 pt-12 lg:pt-10 bg-[#020617] lg:bg-transparent min-h-screen">
        <!-- Animated vertical separator -->
        <div class="hidden lg:block absolute left-0 top-0 bottom-0 w-px bg-gradient-to-b from-transparent via-slate-800/50 to-transparent overflow-hidden">
            <div class="absolute top-0 left-0 w-full h-32 bg-gradient-to-b from-transparent via-cyan-500/50 to-transparent blur-[1px] animate-glow-scan"></div>
        </div>
        
        <div class="w-full max-w-md mx-auto animate-in fade-in slide-in-from-right-12 duration-1000">
            <div class="mb-10">
                <div class="inline-flex items-center space-x-2 px-3 py-1 rounded-full bg-cyan-500/10 border border-cyan-500/20 text-cyan-400 text-[10px] font-black uppercase tracking-[0.2em] mb-8">
                    <div class="w-1.5 h-1.5 rounded-full bg-cyan-500 animate-pulse"></div>
                    <span>{{ t('auth.system_online') }}</span>
                </div>
                <h2 class="text-4xl lg:text-5xl font-black text-white tracking-tight mb-4 italic">{{ t('auth.welcome_back') }}</h2>
                <p class="text-slate-500 font-medium text-base tracking-wide">{{ t('auth.login_subtitle') }}</p>
            </div>

            <!-- Form Section -->
            <form @submit.prevent="handleLogin" class="space-y-6">
                <Transition
                    enter-active-class="transition duration-500 cubic-bezier(0.16, 1, 0.3, 1)"
                    enter-from-class="transform -translate-y-4 opacity-0"
                    enter-to-class="transform translate-y-0 opacity-100"
                    leave-active-class="transition duration-300 ease-in"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0"
                >
                    <div v-if="error" class="bg-red-500/10 border border-red-500/20 rounded-2xl p-5 text-xs text-red-400 font-bold flex items-center space-x-4">
                        <div class="w-2 h-2 rounded-full bg-red-500 animate-pulse"></div>
                        <span class="tracking-wide">{{ error }}</span>
                    </div>
                </Transition>

                <!-- Input Groups with Correct Height and Styles -->
                <div class="space-y-4 group">
                    <label class="text-[11px] uppercase tracking-[0.3em] font-black text-slate-500 group-focus-within:text-cyan-500 transition-colors block ml-1">
                        {{ t('auth.username') }}
                    </label>
                    <div class="relative">
                        <input 
                            type="text" 
                            v-model="loginForm.username" 
                            required
                            placeholder="..."
                            class="w-full h-16 bg-slate-900/40 border border-slate-800/60 rounded-xl px-6 text-base text-white placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-cyan-500/20 focus:border-cyan-500/50 transition-all hover:bg-slate-900/60" 
                        />
                        <div class="absolute right-6 top-1/2 -translate-y-1/2">
                            <UserIcon class="w-5 h-5 text-slate-600 group-focus-within:text-cyan-500 transition-colors" />
                        </div>
                    </div>
                </div>

                <div class="space-y-4 group">
                    <label class="text-[11px] uppercase tracking-[0.3em] font-black text-slate-500 group-focus-within:text-cyan-500 transition-colors block ml-1">
                        {{ t('auth.password') }}
                    </label>
                    <div class="relative">
                        <input 
                            :type="showPassword ? 'text' : 'password'" 
                            v-model="loginForm.password" 
                            required
                            placeholder="••••••••"
                            class="w-full h-16 bg-slate-900/40 border border-slate-800/60 rounded-xl px-6 text-base text-white placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-cyan-500/20 focus:border-cyan-500/50 transition-all hover:bg-slate-900/60 font-mono tracking-widest" 
                        />
                        <div class="absolute right-6 top-1/2 -translate-y-1/2 flex items-center space-x-4">
                            <button @click="showPassword = !showPassword" type="button" class="text-slate-600 hover:text-cyan-400 transition-colors focus:outline-none p-1">
                                <Eye v-if="showPassword" class="w-5 h-5" />
                                <EyeOff v-else class="w-5 h-5" />
                            </button>
                            <div class="h-6 w-px bg-slate-800"></div>
                            <Lock class="w-5 h-5 text-slate-600 group-focus-within:text-cyan-500 transition-colors" />
                        </div>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2">
                    <label class="flex items-center space-x-3 cursor-pointer group">
                        <div class="relative flex items-center justify-center">
                            <input type="checkbox" v-model="loginForm.remember" class="peer h-5 w-5 cursor-pointer appearance-none rounded-lg border border-slate-800 bg-slate-900 transition-all checked:border-cyan-500 checked:bg-cyan-500/10 focus:outline-none">
                            <svg class="pointer-events-none absolute h-3.5 w-3.5 opacity-0 peer-checked:opacity-100 text-cyan-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="4"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"></path></svg>
                        </div>
                        <span class="text-[11px] font-black text-slate-500 group-hover:text-slate-300 transition-colors uppercase tracking-[0.2em]">{{ t('auth.remember_me') }}</span>
                    </label>
                    <button type="button" class="text-[11px] font-black text-slate-600 hover:text-cyan-500 transition-colors uppercase tracking-[0.2em]">{{ t('auth.contact_it') }}</button>
                </div>

                <!-- Action Button with High Visibility -->
                <button 
                    type="submit" 
                    :disabled="loading"
                    class="w-full h-16 relative group overflow-hidden bg-white text-black font-black rounded-2xl shadow-[0_20px_40px_-15px_rgba(6,182,212,0.25)] transition-all transform active:scale-[0.98] disabled:opacity-50 flex items-center justify-center space-x-4"
                >
                    <div class="absolute inset-0 bg-cyan-500 transition-all duration-300 ease-out translate-y-16 group-hover:translate-y-0 opacity-10"></div>
                    <Loader2 v-if="loading" class="w-5 h-5 animate-spin" />
                    <span class="relative uppercase tracking-[0.4em] text-[13px] indent-[0.4em]">{{ loading ? t('auth.authenticating') : t('auth.unlock_dashboard') }}</span>
                    <ArrowRight v-if="!loading" class="w-5 h-5 group-hover:translate-x-2 transition-transform duration-500" />
                </button>
            </form>

            <footer class="mt-5 flex flex-col items-center">
                <div class="h-px w-12 bg-gradient-to-r from-transparent via-slate-800 to-transparent mb-10"></div>
                <p class="text-[10px] text-slate-700 font-bold uppercase tracking-[0.5em] mb-4">{{ t('auth.footer_note') }}</p>
                <div class="flex space-x-8">
                    <a href="https://github.com/coffersTech/nanolog" target="_blank" class="w-10 h-10 rounded-2xl bg-slate-950 border border-slate-900 flex items-center justify-center hover:border-slate-700 hover:bg-slate-900 transition-all group">
                        <svg class="w-5 h-5 text-slate-600 group-hover:text-white transition-colors" fill="currentColor" viewBox="0 0 24 24"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.041-1.416-4.041-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg></a>
                </div>
            </footer>
        </div>
    </div>
  </div>
</template>

<style scoped>
/* Ensure smooth transitions for all elements */
* {
    transition-property: background-color, border-color, color, fill, stroke, opacity, box-shadow, transform;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
    transition-duration: 150ms;
}

/* Custom focus ring for accessibility without ugly outlines */
input:focus {
    box-shadow: 0 0 0 4px rgba(6, 182, 212, 0.1);
}

@keyframes glow-scan {
    0% { transform: translateY(-128px); opacity: 0; }
    10% { opacity: 1; }
    90% { opacity: 1; }
    100% { transform: translateY(100vh); opacity: 0; }
}

.animate-glow-scan {
    animation: glow-scan 4s cubic-bezier(0.4, 0, 0.2, 1) infinite;
}
</style>
