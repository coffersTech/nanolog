import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { LogItem, Stats, User, ApiKey, SystemStatus } from '../types';
import { getT } from '../i18n';
import { useToastStore } from './toast';

export * from './toast';

export const useAppStore = defineStore('app', () => {
  const authToken = ref(sessionStorage.getItem('nanolog_session') || '');
  const isAuthenticated = ref(!!authToken.value);
  const currentUser = ref(sessionStorage.getItem('nanolog_user') || '');
  const userRole = ref(sessionStorage.getItem('nanolog_role') || '');
  const lastUsername = ref(localStorage.getItem('nanolog_last_user') || '');
  
  const currentLang = ref(localStorage.getItem('nanolog_lang') || 'zh');
  const nodeRole = ref('standalone'); // console, engine, standalone
  const systemVersion = ref('v0.0.0');
  const t = computed(() => getT(currentLang.value));

  const setNodeRole = (role: string, version?: string) => {
    nodeRole.value = role;
    if (version) systemVersion.value = version;
  };

  const setAuth = (token: string, user: string, role: string, remember: boolean = false) => {
    authToken.value = token;
    currentUser.value = user;
    userRole.value = role;
    isAuthenticated.value = true;
    sessionStorage.setItem('nanolog_session', token);
    sessionStorage.setItem('nanolog_user', user);
    sessionStorage.setItem('nanolog_role', role);
    
    if (remember) {
      lastUsername.value = user;
      localStorage.setItem('nanolog_last_user', user);
    }
  };

  const logout = () => {
    authToken.value = '';
    currentUser.value = '';
    userRole.value = '';
    isAuthenticated.value = false;
    sessionStorage.removeItem('nanolog_session');
    sessionStorage.removeItem('nanolog_user');
    sessionStorage.removeItem('nanolog_role');
  };

  const setLang = (lang: string) => {
    currentLang.value = lang;
    localStorage.setItem('nanolog_lang', lang);
  };

  const addToast = (message: string, type: any = 'info') => {
    const toast = useToastStore();
    toast.showToast(message, type);
  };

  return {
    authToken,
    isAuthenticated,
    currentUser,
    userRole,
    currentLang,
    lastUsername,
    nodeRole,
    systemVersion,
    t,
    setAuth,
    logout,
    setLang,
    setNodeRole,
    addToast,
  };
});
