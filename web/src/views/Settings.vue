<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useAppStore } from '@/store';
import { api } from '@/api';
import { User, ApiKey, Stats } from '@/types';
import { Plus, Trash2, Key, Shield } from 'lucide-vue-next';

const store = useAppStore();
const activeTab = ref('tokens');
const users = ref<User[]>([]);
const tokens = ref<ApiKey[]>([]);
const retention = ref('');
const stats = ref<Stats | null>(null);

// Modal states
const showAddUserModal = ref(false);
const showAddTokenModal = ref(false);
const showResetPasswordModal = ref(false);
const showTokenCreatedModal = ref(false);
const createdToken = ref('');

// Confirm Modal state
const showConfirmModal = ref(false);
const confirmTitle = ref('');
const confirmMsg = ref('');
const confirmAction = ref<(() => void) | null>(null);

// Form states
const newUser = ref({ username: '', password: '', role: 'viewer' });
const resetPasswordForm = ref({ username: '', password: '' });
const newToken = ref({ name: '', type: 'write' });

const formatBytes = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const fetchData = async () => {
    try {
        if (store.userRole === 'super_admin' || store.userRole === 'admin') {
            users.value = await api.getUsers();
        }
        tokens.value = await api.getTokens();
        const config = await api.getConfig();
        retention.value = config.retention;
        stats.value = await api.getStats();
    } catch (e) { console.error(e); }
};

// User Actions
const addUser = async () => {
    try {
        await api.addUser(newUser.value);
        showAddUserModal.value = false;
        fetchData();
        store.addToast(store.t('common.success'), 'success');
        newUser.value = { username: '', password: '', role: 'viewer' };
    } catch (e: any) {
        store.addToast(e.message, 'error');
    }
};

const deleteUser = async (username: string) => {
    confirmTitle.value = store.t('modals.delete_user_title');
    confirmMsg.value = store.t('modals.delete_user_msg').replace('{name}', username);
    confirmAction.value = async () => {
        try {
            await api.deleteUser(username);
            fetchData();
            store.addToast(store.t('common.success'), 'success');
        } catch (e: any) { store.addToast(e.message, 'error'); }
        showConfirmModal.value = false;
    };
    showConfirmModal.value = true;
};

const openResetPassword = (username: string) => {
    resetPasswordForm.value = { username, password: '' };
    showResetPasswordModal.value = true;
};

const resetPassword = async () => {
    try {
        await api.resetUserPassword(resetPasswordForm.value.username, resetPasswordForm.value.password);
        showResetPasswordModal.value = false;
        store.addToast(store.t('alerts.pwd_updated'), 'success');
    } catch (e: any) { store.addToast(e.message, 'error'); }
};

// Token Actions
const generateToken = async () => {
    try {
        const res = await api.generateToken(newToken.value);
        createdToken.value = res.token;
        showAddTokenModal.value = false;
        showTokenCreatedModal.value = true;
        fetchData();
    } catch (e: any) { store.addToast(e.message, 'error'); }
};

const revokeToken = async (id: string) => {
    if (!id) {
        store.addToast('Invalid Token ID', 'error');
        return;
    }
    confirmTitle.value = store.t('modals.revoke_key_title');
    confirmMsg.value = store.t('modals.revoke_key_msg');
    confirmAction.value = async () => {
        try {
            await api.revokeToken(id);
            await fetchData();
            store.addToast(store.t('alerts.revoke_success'), 'success');
        } catch (e: any) { 
            console.error('Revoke error:', e);
            store.addToast(e.message || store.t('alerts.revoke_failed'), 'error'); 
        }
        showConfirmModal.value = false;
    };
    showConfirmModal.value = true;
};

const copyToken = (val?: string) => {
    const textToCopy = val || createdToken.value;
    if (!textToCopy) return;
    navigator.clipboard.writeText(textToCopy);
    store.addToast(store.t('alerts.token_copied'), 'success');
};

const updateRetention = async () => {
    try {
        await api.updateConfig({ retention: retention.value });
        store.addToast(store.t('alerts.retention_updated'), 'success');
    } catch (e: any) { store.addToast(e.message, 'error'); }
};

onMounted(fetchData);
</script>

<template>
  <main class="flex-1 flex flex-col overflow-hidden bg-gray-900">
    <!-- Admin Mode Banner -->
    <div v-if="store.nodeRole === 'admin' || store.nodeRole === 'console'"
        class="bg-purple-500/10 border-b border-purple-500/20 px-8 py-2 flex items-center space-x-3">
        <Shield class="w-4 h-4 text-purple-400" />
        <p class="text-xs text-purple-400 font-medium tracking-wide">{{ store.t('auth.admin_mode_desc') }}</p>
    </div>

    <!-- Header & Tabs -->
    <header class="h-16 bg-gray-900 border-b border-gray-800 flex items-center px-8 shrink-0">
        <h2 class="text-xl font-bold text-white">{{ store.t('common.admin_console') }}</h2>
        <nav class="ml-12 flex space-x-8">
            <button v-if="store.userRole === 'super_admin' || store.userRole === 'admin'" @click="activeTab = 'users'"
                :class="activeTab === 'users' ? 'text-cyan-400 border-b-2 border-cyan-400' : 'text-gray-400 hover:text-gray-200'"
                class="h-16 px-1 text-sm font-medium transition-all">{{ store.t('settings.tab_users') }}</button>
            <button @click="activeTab = 'tokens'"
                :class="activeTab === 'tokens' ? 'text-cyan-400 border-b-2 border-cyan-400' : 'text-gray-400 hover:text-gray-200'"
                class="h-16 px-1 text-sm font-medium transition-all">{{ store.t('settings.tab_keys') }}</button>
            <button @click="activeTab = 'system'"
                :class="activeTab === 'system' ? 'text-cyan-400 border-b-2 border-cyan-400' : 'text-gray-400 hover:text-gray-200'"
                class="h-16 px-1 text-sm font-medium transition-all">{{ store.t('settings.tab_system') }}</button>
        </nav>
    </header>

    <div class="flex-1 overflow-auto p-8">
        <!-- Tab: Users -->
        <div v-if="activeTab === 'users'" class="max-w-4xl space-y-6">
            <div class="flex items-center justify-between">
                <div>
                    <h3 class="text-lg font-bold text-white">{{ store.t('settings.users_title') }}</h3>
                    <p class="text-sm text-gray-500 mt-1">{{ store.t('settings.users_desc') }}</p>
                </div>
                <button v-if="store.userRole === 'super_admin' || store.userRole === 'admin'"
                    @click="showAddUserModal = true"
                    class="bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-bold px-4 py-2 rounded-lg transition-all shadow-lg shadow-cyan-500/10 flex items-center space-x-2">
                    <Plus class="w-4 h-4" />
                    <span>{{ store.t('settings.add_user') }}</span>
                </button>
            </div>

            <div class="bg-gray-800/50 rounded-xl border border-gray-800 overflow-hidden">
                <table class="w-full text-left">
                    <thead class="bg-black/20 text-[10px] uppercase font-bold text-gray-500 tracking-widest">
                        <tr>
                            <th class="px-6 py-4">{{ store.t('table.username') }}</th>
                            <th class="px-6 py-4">{{ store.t('table.role') }}</th>
                            <th class="px-6 py-4">{{ store.t('table.created_at') }}</th>
                            <th class="px-6 py-4 text-right">{{ store.t('common.actions') }}</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-800">
                        <tr v-for="user in users" :key="user.username" class="text-sm text-gray-300">
                            <td class="px-6 py-4 font-medium text-white">{{ user.username }}</td>
                            <td class="px-6 py-4">
                                <span
                                    :class="user.role === 'super_admin' ? 'bg-purple-500/10 text-purple-400 border-purple-500/20' : 'bg-blue-500/10 text-blue-400 border-blue-500/20'"
                                    class="px-2 py-0.5 rounded border text-xs font-bold uppercase">{{ store.t('roles.' + user.role) }}</span>
                            </td>
                            <td class="px-6 py-4 text-xs font-mono text-gray-500">{{ user.created_at }}</td>
                            <td class="px-6 py-4 text-right space-x-2">
                                <button @click="openResetPassword(user.username)" class="p-1.5 text-gray-500 hover:text-cyan-400 transition-colors">
                                    <Key class="w-4 h-4" />
                                </button>
                                <button @click="deleteUser(user.username)" class="p-1.5 text-gray-500 hover:text-red-500 transition-colors">
                                    <Trash2 class="w-4 h-4" />
                                </button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Tab: Tokens -->
        <div v-if="activeTab === 'tokens'" class="max-w-4xl space-y-6">
            <div class="flex items-center justify-between">
                <div>
                    <h3 class="text-lg font-bold text-white">{{ store.t('settings.api_keys_title') }}</h3>
                    <p class="text-sm text-gray-500 mt-1">{{ store.t('settings.api_keys_desc') }}</p>
                </div>
                <button @click="showAddTokenModal = true" class="bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-bold px-4 py-2 rounded-lg transition-all shadow-lg shadow-cyan-500/10 flex items-center space-x-2">
                    <Plus class="w-4 h-4" />
                    <span>{{ store.t('settings.add_key') }}</span>
                </button>
            </div>

            <div class="bg-gray-800/50 rounded-xl border border-gray-800 overflow-hidden">
                <table class="w-full text-left">
                    <thead class="bg-black/20 text-[10px] uppercase font-bold text-gray-500 tracking-widest">
                        <tr>
                            <th class="px-6 py-4">{{ store.t('table.name') }}</th>
                            <th class="px-6 py-4">{{ store.t('table.prefix') }}</th>
                            <th class="px-6 py-4">{{ store.t('table.type') }}</th>
                            <th class="px-6 py-4">{{ store.t('table.created_by') }}</th>
                            <th class="px-6 py-4 text-right">{{ store.t('common.actions') }}</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-800">
                        <tr v-for="token in tokens" :key="token.id" class="text-sm text-gray-300">
                            <td class="px-6 py-4 font-medium text-white">{{ token.name }}</td>
                            <td class="px-6 py-4 text-xs font-mono text-gray-500">
                                {{ token.token ? token.token.substring(0, 8) : (token.prefix ? 'sk-' + token.prefix : 'sk-••••') }}...
                            </td>
                            <td class="px-6 py-4">
                                <span class="bg-gray-700/50 text-gray-400 px-2 py-0.5 rounded border border-gray-700 text-xs font-bold uppercase">{{ token.type }}</span>
                            </td>
                            <td class="px-6 py-4 text-xs text-gray-400">admin</td>
                            <td class="px-6 py-4 text-right space-x-2">
                                <button v-if="token.token" @click="copyToken(token.token)" class="p-1.5 text-gray-500 hover:text-cyan-400 transition-colors" title="Copy Full Token">
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"></path>
                                    </svg>
                                </button>
                                <button @click="revokeToken(token.id)" class="p-1.5 text-gray-500 hover:text-red-500 transition-colors">
                                    <Trash2 class="w-4 h-4" />
                                </button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Tab: System -->
        <div v-if="activeTab === 'system'" class="max-w-2xl space-y-8">
            <section class="space-y-4">
                <h3 class="text-lg font-bold text-white">{{ store.t('settings.general_info') }}</h3>
                <div class="grid grid-cols-2 gap-4">
                    <div class="bg-black/20 p-4 rounded-lg border border-gray-800">
                        <p class="text-xs uppercase font-bold text-gray-500">{{ store.t('settings.version') }}</p>
                        <p class="text-xl font-bold text-white mt-1">{{ store.t('common.version') }}</p>
                    </div>
                    <div class="bg-black/20 p-4 rounded-lg border border-gray-800">
                        <p class="text-xs uppercase font-bold text-gray-500">{{ store.t('settings.storage') }}</p>
                        <p class="text-xl font-bold text-white mt-1">{{ stats ? formatBytes(stats.disk_usage) : '---' }}</p>
                    </div>
                </div>
            </section>

            <section class="bg-gray-800/50 border border-gray-800 rounded-xl p-6">
                <div class="flex items-center space-x-3 mb-6">
                    <Shield class="w-5 h-5 text-purple-400" />
                    <h3 class="text-lg font-bold text-white">{{ store.t('settings.system_config_title') }}</h3>
                </div>
                <div class="space-y-4">
                    <div>
                        <label class="block text-xs font-bold text-gray-500 uppercase mb-2">{{ store.t('settings.retention_policy') }}</label>
                        <div class="flex gap-4">
                            <input type="text" v-model="retention" class="flex-1 bg-black/50 border border-gray-800 rounded-xl px-4 py-3 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50" />
                            <button @click="updateRetention" class="px-6 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-sm font-bold transition-all transform active:scale-95">
                                {{ store.t('settings.update_policy') }}
                            </button>
                        </div>
                        <p class="text-xs text-gray-500 mt-2">{{ store.t('settings.retention_desc') }}</p>
                    </div>
                </div>
            </section>
        </div>
    </div>

    <!-- Modals -->
    <!-- Add User Modal -->
    <div v-if="showAddUserModal" class="fixed inset-0 z-[110] bg-black/60 backdrop-blur-sm flex items-center justify-center p-6">
        <div class="w-full max-w-md bg-gray-900 border border-gray-800 rounded-2xl p-8 space-y-6 shadow-2xl">
            <h3 class="text-xl font-bold text-white">{{ store.t('modals.add_user_title') }}</h3>
            <div class="space-y-4">
                <div class="space-y-1">
                    <label class="text-xs uppercase font-bold text-gray-500 px-1">{{ store.t('table.username') }}</label>
                    <input type="text" v-model="newUser.username" class="w-full bg-black/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50">
                </div>
                <div class="space-y-1">
                    <label class="text-xs uppercase font-bold text-gray-500 px-1">{{ store.t('auth.password') }}</label>
                    <input type="password" v-model="newUser.password" class="w-full bg-black/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50">
                </div>
                <div class="space-y-1">
                    <label class="text-xs uppercase font-bold text-gray-500 px-1">{{ store.t('modals.role') }}</label>
                    <select v-model="newUser.role" class="w-full bg-black/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 appearance-none">
                        <option value="admin">{{ store.t('roles.admin') }}</option>
                        <option value="viewer">{{ store.t('roles.viewer') }}</option>
                        <option value="super_admin">{{ store.t('roles.super_admin') }}</option>
                    </select>
                </div>
            </div>
            <div class="flex space-x-3 pt-4">
                <button @click="showAddUserModal = false" class="flex-1 bg-gray-800 hover:bg-gray-700 text-gray-400 font-bold py-2.5 rounded-xl transition-all">{{ store.t('common.cancel') }}</button>
                <button @click="addUser" class="flex-1 bg-cyan-600 hover:bg-cyan-500 text-white font-bold py-2.5 rounded-xl transition-all">{{ store.t('common.save') }}</button>
            </div>
        </div>
    </div>

    <!-- Reset Password Modal -->
    <div v-if="showResetPasswordModal" class="fixed inset-0 z-[110] bg-black/60 backdrop-blur-sm flex items-center justify-center p-6">
        <div class="w-full max-w-md bg-gray-900 border border-gray-800 rounded-2xl p-8 space-y-6 shadow-2xl">
            <h3 class="text-xl font-bold text-white">{{ store.t('modals.reset_pwd_title') }}</h3>
            <p class="text-sm text-gray-500 italic">{{ store.t('auth.username') }}: {{ resetPasswordForm.username }}</p>
            <div class="space-y-1">
                <label class="text-xs uppercase font-bold text-gray-500 px-1">{{ store.t('auth.password') }}</label>
                <input type="password" v-model="resetPasswordForm.password" class="w-full bg-black/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50">
            </div>
            <div class="flex space-x-3 pt-4">
                <button @click="showResetPasswordModal = false" class="flex-1 bg-gray-800 hover:bg-gray-700 text-gray-400 font-bold py-2.5 rounded-xl transition-all">{{ store.t('common.cancel') }}</button>
                <button @click="resetPassword" class="flex-1 bg-cyan-600 hover:bg-cyan-500 text-white font-bold py-2.5 rounded-xl transition-all">{{ store.t('common.confirm') }}</button>
            </div>
        </div>
    </div>

    <!-- Add Token Modal -->
    <div v-if="showAddTokenModal" class="fixed inset-0 z-[110] bg-black/60 backdrop-blur-sm flex items-center justify-center p-6">
        <div class="w-full max-w-md bg-gray-900 border border-gray-800 rounded-2xl p-8 space-y-6 shadow-2xl">
            <h3 class="text-xl font-bold text-white">{{ store.t('modals.add_token_title') }}</h3>
            <div class="space-y-4">
                <div class="space-y-1">
                    <label class="text-xs uppercase font-bold text-gray-500 px-1">{{ store.t('modals.token_name') }}</label>
                    <input type="text" v-model="newToken.name" placeholder="e.g. My-Microservice-SDK" class="w-full bg-black/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50">
                </div>
                <div class="space-y-1">
                    <label class="text-xs uppercase font-bold text-gray-500 px-1">{{ store.t('modals.token_type') }}</label>
                    <div class="grid grid-cols-2 gap-3">
                        <button @click="newToken.type = 'write'" :class="newToken.type === 'write' ? 'border-cyan-500 bg-cyan-500/10 text-cyan-400' : 'border-gray-800 text-gray-500 hover:border-gray-700'" class="px-4 py-3 rounded-xl border text-center transition-all">
                            <div class="text-sm font-bold">{{ store.t('modals.permissions.write') || 'Write' }}</div>
                            <div class="text-xs opacity-70">SDK Ingest</div>
                        </button>
                        <button @click="newToken.type = 'read'" :class="newToken.type === 'read' ? 'border-cyan-500 bg-cyan-500/10 text-cyan-400' : 'border-gray-800 text-gray-500 hover:border-gray-700'" class="px-4 py-3 rounded-xl border text-center transition-all">
                            <div class="text-sm font-bold">{{ store.t('modals.permissions.read') }}</div>
                            <div class="text-xs opacity-70">Analytics/Grafana</div>
                        </button>
                    </div>
                </div>
            </div>
            <div class="flex space-x-3 pt-4">
                <button @click="showAddTokenModal = false" class="flex-1 bg-gray-800 hover:bg-gray-700 text-gray-400 font-bold py-2.5 rounded-xl transition-all">{{ store.t('common.cancel') }}</button>
                <button @click="generateToken" class="flex-1 bg-cyan-600 hover:bg-cyan-500 text-white font-bold py-2.5 rounded-xl transition-all">{{ store.t('settings.add_key') }}</button>
            </div>
        </div>
    </div>

    <!-- Token Created Modal -->
    <div v-if="showTokenCreatedModal" class="fixed inset-0 z-[120] bg-black/80 backdrop-blur-md flex items-center justify-center p-6">
        <div class="w-full max-w-lg bg-gray-900 border border-cyan-500/30 rounded-3xl p-10 space-y-8 shadow-2xl relative overflow-hidden">
            <div class="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-cyan-500 to-blue-600"></div>
            <div class="text-center">
                <div class="w-16 h-16 bg-cyan-500/20 rounded-2xl flex items-center justify-center mx-auto mb-6">
                    <Key class="w-8 h-8 text-cyan-400" />
                </div>
                <h3 class="text-2xl font-bold text-white">{{ store.t('modals.token_created_title') }}</h3>
                <p class="text-gray-400 mt-2">{{ store.t('modals.token_created_desc') }}</p>
            </div>
            
            <div class="bg-black/50 border border-gray-800 rounded-2xl p-6 relative group">
                <p class="text-xs font-bold text-gray-500 uppercase mb-3 px-1">Secrect API Key</p>
                <div class="flex items-center space-x-4">
                    <code class="flex-1 text-cyan-400 font-mono text-sm break-all">{{ createdToken }}</code>
                    <button @click="copyToken" class="p-3 bg-gray-800 hover:bg-gray-700 rounded-xl transition-all">
                        <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"></path>
                        </svg>
                    </button>
                </div>
            </div>

            <button @click="showTokenCreatedModal = false" class="w-full bg-cyan-600 hover:bg-cyan-500 text-white font-bold py-4 rounded-2xl transition-all shadow-lg shadow-cyan-500/20">
                {{ store.t('common.confirm') }}
            </button>
        </div>
    </div>

    <!-- Generic Confirmation Modal -->
    <div v-if="showConfirmModal" class="fixed inset-0 z-[150] bg-black/60 backdrop-blur-sm flex items-center justify-center p-6">
        <div class="w-full max-w-sm bg-gray-900 border border-gray-800 rounded-2xl p-8 space-y-6 shadow-2xl">
            <div class="text-center">
                <div class="w-12 h-12 bg-red-500/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                    <Trash2 class="w-6 h-6 text-red-500" />
                </div>
                <h3 class="text-lg font-bold text-white">{{ confirmTitle }}</h3>
                <p class="text-sm text-gray-500 mt-2">{{ confirmMsg }}</p>
            </div>
            <div class="flex space-x-3 pt-4">
                <button @click="showConfirmModal = false" class="flex-1 bg-gray-800 hover:bg-gray-700 text-gray-400 font-bold py-2.5 rounded-xl transition-all">
                    {{ store.t('common.cancel') }}
                </button>
                <button @click="confirmAction" class="flex-1 bg-red-600 hover:bg-red-500 text-white font-bold py-2.5 rounded-xl transition-all shadow-lg shadow-red-600/20">
                    {{ store.t('common.confirm') }}
                </button>
            </div>
        </div>
    </div>
  </main>
</template>
