import { useAppStore } from '../store';

async function apiFetch(url: string, options: RequestInit = {}) {
  const store = useAppStore();
  const headers = {
    ...options.headers,
    'Authorization': `Bearer ${store.authToken}`
  } as any;

  const response = await fetch(url, { ...options, headers });

  if (response.status === 401) {
    store.logout();
    throw new Error('Unauthorized');
  }

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(errorText || `Request failed with status ${response.status}`);
  }

  return response;
}

export const api = {
  async login(payload: any) {
    const response = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  },

  async getStats() {
    const res = await apiFetch('/api/stats');
    return res.json();
  },

  async searchLogs(params: string) {
    const res = await apiFetch(`/api/search?${params}`);
    if (res.status === 400) throw new Error(await res.text());
    return res.json();
  },

  async getHistogram(params: string) {
    const res = await apiFetch(`/api/histogram?${params}`);
    return res.json();
  },

  async getContext(ts: number, service: string, limit: number = 10) {
    const res = await apiFetch(`/api/context?ts=${ts}&service=${encodeURIComponent(service)}&limit=${limit}`);
    return res.json();
  },

  async getSystemStatus() {
    const res = await fetch('/api/system/status');
    return res.json();
  },

  async getInstances() {
    const res = await apiFetch('/api/registry/instances');
    return res.json();
  },
  
  async getDevices() {
    const res = await apiFetch('/api/registry/devices');
    return res.json();
  },

  async deleteDevice(id: string) {
    const res = await apiFetch(`/api/registry/devices/${id}`, {
      method: 'DELETE'
    });
    if (!res.ok) throw new Error(await res.text());
    return res;
  },

  async getUsers() {
    const res = await apiFetch('/api/users');
    return res.json();
  },

  async getTokens() {
    const res = await apiFetch('/api/tokens');
    return res.json();
  },

  async getConfig() {
    const res = await apiFetch('/api/system/config');
    return res.json();
  },

  async updateConfig(payload: any) {
    const res = await apiFetch('/api/system/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!res.ok) throw new Error(await res.text());
    return res;
  },

  async addUser(payload: any) {
    const res = await apiFetch('/api/users', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!res.ok) throw new Error(await res.text());
    return res;
  },

  async deleteUser(username: string) {
    const res = await apiFetch(`/api/users/${username}`, {
      method: 'DELETE'
    });
    if (!res.ok) throw new Error(await res.text());
    return res;
  },

  async resetUserPassword(username: string, password: string) {
    const res = await apiFetch(`/api/users/${username}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password })
    });
    if (!res.ok) throw new Error(await res.text());
    return res;
  },

  async generateToken(payload: any) {
    const res = await apiFetch('/api/tokens', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  async revokeToken(id: string) {
    const res = await apiFetch(`/api/tokens/${id}`, {
      method: 'DELETE'
    });
    if (!res.ok) throw new Error(await res.text());
    return res;
  },
};
