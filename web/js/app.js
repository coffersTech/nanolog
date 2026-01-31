const { createApp, ref, computed, onMounted, onUnmounted, watch } = Vue;

const STORAGE_KEY = 'nanolog_session';

createApp({
    setup() {
        const logs = ref([]);
        const loading = ref(false);
        const error = ref(null);
        const searchQuery = ref('');
        const autoRefresh = ref(0);
        const showRecentHistory = ref(false);
        const expandedIndex = ref(-1);
        const currentView = ref('discover');
        const stats = ref({ ingestion_rate: 0, disk_usage: 0, total_logs: 0 });
        let refreshInterval = null;
        let statsInterval = null;
        let startPicker = null;
        let endPicker = null;

        // I18n State
        const currentLang = ref(localStorage.getItem('nanolog_lang') || 'zh');
        console.log('NanoLog App Logic v1.5 Loaded');
        const t = (path, params = {}) => {
            const keys = path.split('.');
            let obj = messages[currentLang.value];
            for (const k of keys) {
                if (!obj) return path;
                obj = obj[k];
            }
            let str = obj || path;
            if (typeof str === 'string') {
                Object.keys(params).forEach(key => {
                    str = str.replace(`{${key}}`, params[key]);
                });
            }
            return str;
        };
        const setLang = (lang) => {
            currentLang.value = lang;
            localStorage.setItem('nanolog_lang', lang);
        };
        const pendingLang = ref('');
        const showLangConfirm = ref(false);

        const confirmSwitchLanguage = () => {
            const target = currentLang.value === 'en' ? 'zh' : 'en';
            pendingLang.value = target;
            showLangConfirm.value = true;
        };

        const executeSwitchLanguage = () => {
            setLang(pendingLang.value);
            showLangConfirm.value = false;
            window.location.reload(); // Reload to ensure full re-render
        };

        const authToken = ref('');
        const isAuthenticated = ref(false);
        const loginForm = ref({ username: '', password: '', remember: true });
        const userRole = ref('');
        const currentUser = ref('');
        const systemInitialized = ref(true);
        const nodeRole = ref('all');

        // Management State
        const settingsTab = ref('tokens');
        const users = ref([]);
        const tokens = ref([]);
        const showAddUserModal = ref(false);
        const newUser = ref({ username: '', password: '', role: 'admin' });
        const showAddTokenModal = ref(false);
        const newToken = ref({ name: '', type: 'write' });
        const generatedToken = ref(null);
        const initForm = ref({ username: '', password: '' });
        const retentionInput = ref('');
        const showLogoutConfirm = ref(false);
        // Instances State
        const instances = ref([]);
        const instancesInterval = ref(null);

        const showResetPasswordModal = ref(false);
        const resetPasswordForm = ref({ username: '', password: '' });
        const showChangePasswordModal = ref(false);
        const changePasswordForm = ref({ currentPassword: '', newPassword: '', confirmPassword: '' });
        const showUserMenu = ref(false);
        const selectedTimeRange = ref('15m');
        const showTimeRangeDropdown = ref(false);
        const showCustomTimeModal = ref(false);
        const customTimeRange = ref({ start: '', end: '' });
        const recentCustomRanges = ref(JSON.parse(localStorage.getItem('recentCustomRanges') || '[]'));
        const timeParams = ref({ start: null, end: null });
        const timeRangeLabel = ref('');
        const histogramTotal = ref(0);
        const searchError = ref(''); // NanoQL syntax error message

        // Context Modal
        const contextModal = ref({ show: false, loading: false, pre: [], anchor: null, post: [], service: '', limit: 10 });
        const openContextModal = async (logItem) => {
            contextModal.value = { show: true, loading: true, pre: [], anchor: null, post: [], service: logItem.service, limit: 10 };
            try {
                const response = await apiFetch(`/api/context?ts=${logItem.timestamp}&service=${encodeURIComponent(logItem.service)}&limit=10`);
                const data = await response.json();
                contextModal.value.pre = data.pre || [];
                contextModal.value.anchor = data.anchor;
                contextModal.value.post = data.post || [];
            } catch (e) {
                console.error('Context fetch error:', e);
            } finally {
                contextModal.value.loading = false;
            }
        };

        // Confirmation Modal
        const confirmModal = ref({ show: false, title: '', message: '', action: null });
        const triggerConfirm = (title, message, action) => {
            confirmModal.value = { show: true, title, message, action };
        };

        // Toast notification
        const toast = ref({ show: false, message: '', type: 'success' });
        const showToast = (message, type = 'success') => {
            toast.value = { show: true, message, type };
            setTimeout(() => {
                toast.value.show = false;
            }, 3000);
        };

        watch(showTimeRangeDropdown, (open) => {
            if (open && timeParams.value.start && timeParams.value.end) {
                const formatDate = (ts) => {
                    const d = new Date(ts / 1000000);
                    const pad = (n) => String(n).padStart(2, '0');
                    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
                };
                const s = formatDate(timeParams.value.start);
                const e = formatDate(timeParams.value.end);
                customTimeRange.value.start = s;
                customTimeRange.value.end = e;

                // Sync Flatpickr instances
                if (startPicker) startPicker.setDate(s, false);
                if (endPicker) endPicker.setDate(e, false);
            }
        });

        const apiFetch = async (url, options = {}) => {
            const headers = {
                ...options.headers,
                'Authorization': `Bearer ${authToken.value}`
            };

            const response = await fetch(url, { ...options, headers });

            if (response.status === 401) {
                isAuthenticated.value = false;
                authToken.value = '';
                localStorage.removeItem(STORAGE_KEY);
                throw new Error('Unauthorized');
            }

            return response;
        };

        const login = async () => {
            if (!loginForm.value.username || !loginForm.value.password) return;
            loading.value = true;
            error.value = null;
            try {
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        username: loginForm.value.username,
                        password: loginForm.value.password
                    })
                });

                if (!response.ok) {
                    const txt = await response.text();
                    throw new Error(txt || t('alerts.invalid_credentials'));
                }

                const data = await response.json();
                authToken.value = data.token;
                userRole.value = data.role;
                currentUser.value = data.username;

                if (loginForm.value.remember) {
                    localStorage.setItem(STORAGE_KEY, authToken.value);
                    localStorage.setItem('nanolog_role', userRole.value);
                    localStorage.setItem('nanolog_user', currentUser.value);
                } else {
                    localStorage.removeItem(STORAGE_KEY);
                }

                isAuthenticated.value = true;
                fetchAll();
                if (currentView.value === 'dashboard') initDashboard();
            } catch (e) {
                error.value = e.message || t('alerts.login_failed');
                authToken.value = '';
            } finally {
                loading.value = false;
            }
        };

        const logout = () => {
            showLogoutConfirm.value = true;
        };

        const confirmLogout = () => {
            isAuthenticated.value = false;
            authToken.value = '';
            userRole.value = '';
            currentUser.value = '';
            localStorage.removeItem(STORAGE_KEY);
            localStorage.removeItem('nanolog_role');
            localStorage.removeItem('nanolog_user');
            // Clear current data
            logs.value = [];
            stats.value = { ingestion_rate: 0, disk_usage: 0, total_logs: 0 };
            showLogoutConfirm.value = false;
        };

        const cancelLogout = () => {
            showLogoutConfirm.value = false;
        };

        const parseSearchQuery = (q) => {
            const params = new URLSearchParams();
            params.append('limit', '100');

            // Pass entire query string as NanoQL (q parameter)
            if (q && q.trim()) {
                params.append('q', q.trim());
            }

            // Always append time params if available
            if (timeParams.value.start) params.append('start', timeParams.value.start);
            if (timeParams.value.end) params.append('end', timeParams.value.end);

            return params.toString();
        };

        const updateTimeRange = (range) => {
            selectedTimeRange.value = range;
            const now = new Date();
            const nowTs = now.getTime();
            let start = null;
            let end = nowTs * 1000000;

            const t_key = range.startsWith('last_') ? range : (range === 'custom' ? 'custom' : 'search.' + range);

            // Helper to get start of day
            const getStartOfDay = (d) => {
                const day = new Date(d);
                day.setHours(0, 0, 0, 0);
                return day;
            };

            if (range === '5m') start = (nowTs - 5 * 60 * 1000) * 1000000;
            else if (range === '15m') start = (nowTs - 15 * 60 * 1000) * 1000000;
            else if (range === '30m') start = (nowTs - 30 * 60 * 1000) * 1000000;
            else if (range === '1h') start = (nowTs - 60 * 60 * 1000) * 1000000;
            else if (range === '6h') start = (nowTs - 6 * 60 * 60 * 1000) * 1000000;
            else if (range === '24h' || range === '1d') start = (nowTs - 24 * 60 * 60 * 1000) * 1000000;
            else if (range === '3d') start = (nowTs - 3 * 24 * 60 * 60 * 1000) * 1000000;
            else if (range === '7d') start = (nowTs - 7 * 24 * 60 * 60 * 1000) * 1000000;
            else if (range === '30d') start = (nowTs - 30 * 24 * 60 * 60 * 1000) * 1000000;
            else if (range === '90d') start = (nowTs - 90 * 24 * 60 * 60 * 1000) * 1000000;
            else if (range === 'today') {
                start = getStartOfDay(now).getTime() * 1000000;
            } else if (range === 'yesterday') {
                const y = getStartOfDay(now);
                y.setDate(y.getDate() - 1);
                start = y.getTime() * 1000000;
                const ye = new Date(y);
                ye.setHours(23, 59, 59, 999);
                end = ye.getTime() * 1000000;
            } else if (range === 'day_before_yesterday') {
                const d = getStartOfDay(now);
                d.setDate(d.getDate() - 2);
                start = d.getTime() * 1000000;
                const de = new Date(d);
                de.setHours(23, 59, 59, 999);
                end = de.getTime() * 1000000;
            } else if (range === 'this_week') {
                const d = getStartOfDay(now);
                const day = d.getDay() || 7; // 1-7, 7 is Sunday
                d.setDate(d.getDate() - day + 1);
                start = d.getTime() * 1000000;
            } else if (range === 'last_week') {
                const d = getStartOfDay(now);
                const day = d.getDay() || 7;
                d.setDate(d.getDate() - day - 6); // Previous Monday
                start = d.getTime() * 1000000;
                const de = new Date(d);
                de.setDate(de.getDate() + 6);
                de.setHours(23, 59, 59, 999);
                end = de.getTime() * 1000000;
            } else if (range === 'this_month') {
                const d = getStartOfDay(now);
                d.setDate(1);
                start = d.getTime() * 1000000;
            } else if (range === 'last_month') {
                const d = getStartOfDay(now);
                d.setMonth(d.getMonth() - 1);
                d.setDate(1);
                start = d.getTime() * 1000000;
                const de = new Date(d);
                de.setMonth(de.getMonth() + 1);
                de.setDate(0); // Last day of previous month
                de.setHours(23, 59, 59, 999);
                end = de.getTime() * 1000000;
            } else if (range === 'custom') {
                // For custom, we don't fetch yet, just show inputs
                return;
            }

            timeParams.value = { start, end };
            if (range === 'custom') {
                timeRangeLabel.value = t('search.custom');
            } else {
                timeRangeLabel.value = t('search.' + range) || range;
            }
            showTimeRangeDropdown.value = false; // Close for presets
            fetchAll();
        };

        const applyCustomTimeRange = () => {
            if (!customTimeRange.value.start || !customTimeRange.value.end) return;
            const startStr = customTimeRange.value.start;
            const endStr = customTimeRange.value.end;
            const start = new Date(startStr).getTime() * 1000000;
            const end = new Date(endStr).getTime() * 1000000;

            if (end <= start) {
                showToast(t('alerts.invalid_time_range'), 'error');
                return;
            }

            timeParams.value = { start, end };
            selectedTimeRange.value = 'custom';

            // Format label as date range
            const d1 = new Date(start / 1000000);
            const d2 = new Date(end / 1000000);
            const pad = (n) => String(n).padStart(2, '0');
            const fmtStr = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
            timeRangeLabel.value = `${fmtStr(d1)} ~ ${fmtStr(d2)}`;
            console.log('Applied Custom Range Label:', timeRangeLabel.value);

            // Save to recent
            const newRange = { start: startStr, end: endStr, label: timeRangeLabel.value };
            recentCustomRanges.value = [newRange, ...recentCustomRanges.value.filter(r => r.start !== newRange.start || r.end !== newRange.end)].slice(0, 3);
            localStorage.setItem('recentCustomRanges', JSON.stringify(recentCustomRanges.value));

            showTimeRangeDropdown.value = false;
            fetchAll();
        };

        const selectRecentCustom = (recent) => {
            customTimeRange.value.start = recent.start;
            customTimeRange.value.end = recent.end;
            applyCustomTimeRange();
        };


        // Initialize default time range
        setTimeout(() => updateTimeRange('15m'), 100);

        const fetchLogs = async () => {
            if (!isAuthenticated.value) return;
            if (loading.value && logs.value.length > 0) return;
            loading.value = true;
            error.value = null;
            searchError.value = ''; // Clear previous search error
            try {
                const queryString = parseSearchQuery(searchQuery.value);
                const response = await apiFetch(`/api/search?${queryString}`);

                // Handle syntax errors (400 Bad Request)
                if (response.status === 400) {
                    const errorText = await response.text();
                    searchError.value = errorText || 'Invalid query syntax';
                    logs.value = [];
                    return;
                }

                const data = await response.json();
                logs.value = data || [];
            } catch (e) {
                if (e.message !== 'Unauthorized') {
                    error.value = `Failed to fetch logs: ${e.message}`;
                }
            } finally {
                loading.value = false;
            }
        };

        const filteredLogs = computed(() => logs.value);

        const formatTimestamp = (ts) => {
            const ms = Math.floor(ts / 1000000);
            const d = new Date(ms);
            return d.getFullYear() + "-" +
                String(d.getMonth() + 1).padStart(2, '0') + "-" +
                String(d.getDate()).padStart(2, '0') + " " +
                String(d.getHours()).padStart(2, '0') + ":" +
                String(d.getMinutes()).padStart(2, '0') + ":" +
                String(d.getSeconds()).padStart(2, '0');
        };

        const getLevelText = (l) => ({ 0: 'DEBUG', 1: 'INFO', 2: 'WARN', 3: 'ERROR', 4: 'FATAL' }[l] || 'UNKNOWN');

        // Toggle Row
        const toggleRow = (index) => {
            expandedIndex.value = expandedIndex.value === index ? -1 : index;
        };

        // JSON Formatting helpers
        const isJson = (str) => {
            try {
                const o = JSON.parse(str);
                return o && typeof o === 'object';
            } catch (e) { return false; }
        };

        const formatJson = (str) => {
            try {
                return JSON.stringify(JSON.parse(str), null, 2);
            } catch (e) { return str; }
        };

        // Extract search terms from NanoQL query for highlighting
        const getSearchTerms = () => {
            const q = searchQuery.value;
            if (!q) return [];
            const terms = [];
            // Extract quoted strings
            const quotedMatches = q.match(/"([^"]+)"/g);
            if (quotedMatches) {
                quotedMatches.forEach(m => terms.push(m.replace(/"/g, '')));
            }
            // Extract value parts from key:value patterns
            const kvMatches = q.match(/\w+:([^\s"]+|"[^"]+")/g);
            if (kvMatches) {
                kvMatches.forEach(m => {
                    const val = m.split(':')[1]?.replace(/"/g, '');
                    if (val && !['AND', 'OR', 'NOT'].includes(val.toUpperCase())) {
                        terms.push(val);
                    }
                });
            }
            return [...new Set(terms)];
        };

        // Highlight matching text in a string
        const highlightText = (text) => {
            const terms = getSearchTerms();
            if (!terms.length || !text) return text;
            let result = text;
            terms.forEach(term => {
                const regex = new RegExp(`(${term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
                result = result.replace(regex, '<span class="bg-yellow-900 text-yellow-100 px-0.5 rounded">$1</span>');
            });
            return result;
        };

        const getLevelClass = (l) => ({
            0: 'bg-gray-800 text-gray-400',
            1: 'bg-green-500/10 text-green-500',
            2: 'bg-yellow-500/10 text-yellow-500',
            3: 'bg-red-500/10 text-red-500',
            4: 'bg-purple-500/10 text-purple-500'
        }[l] || 'bg-gray-800 text-gray-400');



        // Chart Logic
        let chartInstance = null;
        const initChart = () => {
            const el = document.getElementById('logHistogram');
            if (!el) return;
            const ctx = el.getContext('2d');
            chartInstance = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Log Volume',
                        data: [],
                        backgroundColor: '#06b6d4', // cyan-500
                        borderRadius: 2,
                        borderSkipped: false,
                        hoverBackgroundColor: '#0891b2'
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    layout: {
                        padding: {
                            top: 25
                        }
                    },
                    interaction: {
                        mode: 'index',
                        intersect: false,
                    },
                    scales: {
                        x: {
                            ticks: {
                                color: '#6b7280',
                                maxTicksLimit: 10,
                                callback: function (value) {
                                    const ts = this.getLabelForValue(value);
                                    if (!ts) return '';
                                    const d = new Date(ts / 1000000);
                                    const pad = (n) => String(n).padStart(2, '0');
                                    const durationSec = chartInstance.config.customDuration || 0;

                                    if (durationSec > 30 * 86400) return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
                                    if (durationSec > 86400) return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;

                                    // Default Time
                                    if ((chartInstance.config.customInterval || 60) < 60) {
                                        return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
                                    }
                                    return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
                                }
                            },
                            grid: { display: false }
                        },
                        y: {
                            ticks: { color: '#6b7280' },
                            grid: { color: '#1f2937' } // gray-800
                        }
                    },
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            backgroundColor: 'rgba(255, 255, 255, 0.95)',
                            titleColor: '#1f2937', // gray-800
                            bodyColor: '#374151', // gray-700
                            borderColor: '#e5e7eb', // gray-200
                            borderWidth: 1,
                            padding: 12,
                            boxPadding: 4,
                            usePointStyle: true,
                            callbacks: {
                                title: (items) => {
                                    if (items.length > 0) {
                                        const startTs = parseInt(items[0].label);
                                        const intervalSec = chartInstance.config.customInterval || 0;
                                        const startD = new Date(startTs / 1000000);
                                        const endD = new Date((startTs / 1000000) + (intervalSec * 1000));

                                        const pad = (n) => String(n).padStart(2, '0');
                                        const fmtTime = (d) => `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
                                        const fmtDate = (d) => `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

                                        // Include date if range > 24h
                                        if ((chartInstance.config.customDuration || 0) > 86400) {
                                            return `${fmtDate(startD)} ${fmtTime(startD)} - ${fmtTime(endD)}`;
                                        }
                                        return `${fmtTime(startD)} - ${fmtTime(endD)}`;
                                    }
                                    return '';
                                },
                                label: (context) => {
                                    const count = context.raw;
                                    const interval = chartInstance.config.customInterval || 0;
                                    let intervalStr = interval + '秒';
                                    if (interval >= 3600) intervalStr = Math.round(interval / 3600 * 10) / 10 + '小时';
                                    else if (interval >= 60) intervalStr = Math.round(interval / 60 * 10) / 10 + '分钟';

                                    return [
                                        ` 日志总条数    ${count}`,
                                        ` 统计间隔        ${intervalStr}`
                                    ];
                                },
                                labelColor: function (context) {
                                    return {
                                        borderColor: '#06b6d4',
                                        backgroundColor: '#06b6d4',
                                        borderWidth: 1,
                                        borderRadius: 0,
                                    };
                                }
                            }
                        }
                    }
                }
            });
        };

        const fetchHistogram = async () => {
            if (!isAuthenticated.value) return;
            try {
                // 1. Calculate Dynamic Interval
                let durationSec = 15 * 60; // Default 15m
                let startNs = timeParams.value.start;
                let endNs = timeParams.value.end;

                if (startNs && endNs) {
                    durationSec = (endNs - startNs) / 1000000000;
                } else {
                    // Fallback if params missing (shouldn't happen with fix)
                    endNs = Date.now() * 1000000;
                    startNs = endNs - (durationSec * 1000000000);
                }

                // Target ~50 bars
                const rawInterval = durationSec / 50;

                // Snap to grid (seconds)
                const ranges = [
                    10, 20, 30,
                    60, 300, 600, 1800, // 1m, 5m, 10m, 30m
                    3600, 7200, 14400, 21600, 43200, // 1h, 2h, 4h, 6h, 12h
                    86400, 172800 // 1d, 2d
                ];

                // Find closest standard interval (rounding up)
                let interval = ranges.find(r => r >= rawInterval) || ranges[ranges.length - 1];
                if (rawInterval > ranges[ranges.length - 1]) {
                    // For very long ranges, just use raw calc rounded to hours
                    interval = Math.ceil(rawInterval / 3600) * 3600;
                }

                let qs = parseSearchQuery(searchQuery.value);
                if (!qs.includes('interval=')) {
                    qs += `&interval=${interval}`;
                }

                // 2. Fetch Data
                const res = await apiFetch(`/api/histogram?${qs}`);
                const data = await res.json(); // [{time: ns, count: int}, ...]

                // 3. Zero-Fill Logic (Generate Buckets)
                const buckets = [];
                // Align start time to interval boundary for clean charts
                const intervalNs = interval * 1000000000;
                let current = Math.floor(startNs / intervalNs) * intervalNs;
                const endBoundary = Math.ceil(endNs / intervalNs) * intervalNs;

                // Create Map for O(1) lookup
                const dataMap = new Map();
                if (data) {
                    data.forEach(p => {
                        // Align data timestamp to bucket start
                        const bucketTime = Math.floor(p.time / intervalNs) * intervalNs;
                        dataMap.set(bucketTime, (dataMap.get(bucketTime) || 0) + p.count);
                    });
                }

                while (current <= endBoundary) {
                    buckets.push({
                        time: current,
                        count: dataMap.get(current) || 0
                    });
                    current += intervalNs;
                }

                if (chartInstance) {
                    // Update chart data with raw timestamps
                    chartInstance.data.labels = buckets.map(p => p.time);
                    chartInstance.data.datasets[0].data = buckets.map(p => p.count);

                    // Store metadata for tooltip and formatting
                    chartInstance.config.customInterval = interval; // in seconds
                    chartInstance.config.customDuration = durationSec;

                    chartInstance.update();
                }

                // Update total count
                histogramTotal.value = buckets.reduce((acc, curr) => acc + curr.count, 0);

            } catch (e) {
                if (e.message !== 'Unauthorized') console.error("Histogram fetch error", e);
            }
        };

        watch(autoRefresh, (v) => {
            if (refreshInterval) { clearInterval(refreshInterval); refreshInterval = null; }
            if (v > 0 && isAuthenticated.value) {
                refreshInterval = setInterval(() => {
                    // 动态更新时间窗口 (Rolling Window)
                    const r = selectedTimeRange.value;
                    const now = Date.now();
                    const ms = (m) => m * 60 * 1000;

                    const durationMap = {
                        '5m': ms(5), '15m': ms(15), '30m': ms(30),
                        '1h': ms(60), '6h': ms(360),
                        '24h': ms(1440), '1d': ms(1440),
                        '3d': ms(4320), '7d': ms(10080),
                        '30d': ms(43200), '90d': ms(129600)
                    };

                    if (durationMap[r]) {
                        timeParams.value.end = now * 1000000;
                        timeParams.value.start = (now - durationMap[r]) * 1000000;
                    } else if (['today', 'this_week', 'this_month'].includes(r)) {
                        // 仅更新结束时间为当前时间
                        timeParams.value.end = now * 1000000;
                    }

                    fetchLogs();
                    fetchHistogram();
                }, v * 1000);
            }
        });

        const fetchAll = () => {
            if (!isAuthenticated.value) return;
            fetchLogs();
            fetchHistogram();
        };

        const fetchConfig = async () => {
            try {
                const res = await apiFetch('/api/system/config');
                const data = await res.json();
                retentionInput.value = data.retention;
            } catch (e) { }
        };

        const updateRetention = async () => {
            try {
                const res = await apiFetch('/api/system/config', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ retention: retentionInput.value })
                });
                if (res.ok) alert(t('alerts.retention_updated'));
            } catch (e) { alert(e.message); }
        };

        const fetchUsers = async () => {
            if (userRole.value !== 'super_admin') return;
            try {
                const res = await apiFetch('/api/users');
                users.value = await res.json();
            } catch (e) { }
        };

        const addUser = async () => {
            try {
                const res = await apiFetch('/api/users', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(newUser.value)
                });
                if (res.ok) {
                    showAddUserModal.value = false;
                    newUser.value = { username: '', password: '', role: 'admin' };
                    fetchUsers();
                }
            } catch (e) { alert(e.message); }
        };

        const deleteUser = async (username) => {
            triggerConfirm(
                t('modals.delete_user_title'),
                t('modals.delete_user_msg', { name: username }),
                async () => {
                    try {
                        const res = await apiFetch(`/api/users/${username}`, { method: 'DELETE' });
                        if (res.ok) {
                            showToast(t('alerts.user_deleted', { username }), 'success');
                            fetchUsers();
                        } else {
                            const txt = await res.text();
                            showToast(t('common.error') + ': ' + txt, 'error');
                        }
                    } catch (e) {
                        showToast(t('common.error') + ': ' + e.message, 'error');
                    }
                }
            );
        };

        const openResetPassword = (username) => {
            resetPasswordForm.value = { username, password: '' };
            showResetPasswordModal.value = true;
        };

        const resetPassword = async () => {
            if (!resetPasswordForm.value.password) {
                showToast(t('alerts.enter_new_pwd'), 'error');
                return;
            }
            try {
                const res = await apiFetch(`/api/users/${resetPasswordForm.value.username}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ password: resetPasswordForm.value.password })
                });
                if (res.ok) {
                    showToast(t('alerts.pwd_updated'), 'success');
                    showResetPasswordModal.value = false;
                    resetPasswordForm.value = { username: '', password: '' };
                } else {
                    const txt = await res.text();
                    showToast(t('common.error') + ': ' + txt, 'error');
                }
            } catch (e) {
                showToast(t('common.error') + ': ' + e.message, 'error');
            }
        };

        const changePassword = async () => {
            if (!changePasswordForm.value.currentPassword || !changePasswordForm.value.newPassword) {
                showToast(t('alerts.fill_all_fields'), 'error');
                return;
            }
            if (changePasswordForm.value.newPassword !== changePasswordForm.value.confirmPassword) {
                showToast(t('alerts.pwd_mismatch'), 'error');
                return;
            }
            try {
                const res = await apiFetch('/api/profile/password', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        current_password: changePasswordForm.value.currentPassword,
                        new_password: changePasswordForm.value.newPassword
                    })
                });
                if (res.ok) {
                    showToast(t('alerts.pwd_updated'), 'success');
                    showChangePasswordModal.value = false;
                    changePasswordForm.value = { currentPassword: '', newPassword: '', confirmPassword: '' };
                } else {
                    const txt = await res.text();
                    showToast(txt || t('common.error'), 'error');
                }
            } catch (e) {
                showToast(t('common.error') + ': ' + e.message, 'error');
            }
        };

        const fetchTokens = async () => {
            try {
                const res = await apiFetch('/api/tokens');
                tokens.value = await res.json();
            } catch (e) { }
        };

        const generateToken = async () => {
            try {
                const res = await apiFetch('/api/tokens', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(newToken.value)
                });
                const data = await res.json();
                generatedToken.value = data.token;
                showAddTokenModal.value = false;
                newToken.value = { name: '', type: 'write' };
                fetchTokens();
            } catch (e) { showToast(e.message, 'error'); }
        };

        const revokeToken = async (id) => {
            triggerConfirm(
                t('modals.revoke_key_title'),
                t('modals.revoke_key_msg'),
                async () => {
                    try {
                        await apiFetch(`/api/tokens/${id}`, { method: 'DELETE' });
                        showToast(t('alerts.revoke_success'), 'success');
                        fetchTokens();
                    } catch (e) { showToast(t('alerts.revoke_failed'), 'error'); }
                }
            );
        };

        const copyGeneratedToken = () => {
            navigator.clipboard.writeText(generatedToken.value);
            showToast(t('alerts.token_copied'), 'success');
        };

        const copyToken = (token) => {
            navigator.clipboard.writeText(token);
            showToast(t('alerts.token_copied'), 'success');
        };

        const checkSystemStatus = async () => {
            try {
                const res = await fetch('/api/system/status');
                const data = await res.json();
                systemInitialized.value = data.initialized;
                nodeRole.value = data.node_role || 'all';

                // Initial view adjustment
                if (nodeRole.value === 'admin' && currentView.value !== 'settings') {
                    currentView.value = 'settings';
                } else if (nodeRole.value === 'engine' && currentView.value === 'settings') {
                    currentView.value = 'discover';
                }
            } catch (e) {
                console.error("Failed to check system status", e);
            }
        };

        const initializeSystem = async () => {
            loading.value = true;
            error.value = null;
            try {
                const res = await fetch('/api/system/init', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(initForm.value)
                });
                if (!res.ok) throw new Error(await res.text());
                const data = await res.json();
                authToken.value = data.token;
                userRole.value = data.role;
                currentUser.value = data.username;
                isAuthenticated.value = true;
                systemInitialized.value = true;
                nodeRole.value = data.node_role || nodeRole.value;
                localStorage.setItem(STORAGE_KEY, data.token);
                localStorage.setItem('nanolog_role', data.role);
                localStorage.setItem('nanolog_user', data.username);
                fetchAll();
            } catch (e) { error.value = e.message; }
            finally { loading.value = false; }
        };

        onMounted(async () => {
            // Smoothly remove preloader
            const loader = document.getElementById('app-loader');
            if (loader) {
                loader.style.opacity = '0';
                setTimeout(() => {
                    loader.style.visibility = 'hidden';
                }, 500);
            }

            await checkSystemStatus();
            const savedToken = localStorage.getItem(STORAGE_KEY);
            if (savedToken) {
                authToken.value = savedToken;
                userRole.value = localStorage.getItem('nanolog_role') || '';
                currentUser.value = localStorage.getItem('nanolog_user') || '';
                // Verify token
                try {
                    const res = await apiFetch('/api/stats');
                    if (res.ok) {
                        isAuthenticated.value = true;
                        initChart();
                        fetchAll();
                    } else {
                        localStorage.removeItem(STORAGE_KEY);
                    }
                } catch (e) {
                    localStorage.removeItem(STORAGE_KEY);
                }
            }
            loading.value = false;

            // Initialize Flatpickr
            const pickerConfig = {
                enableTime: true,
                time_24hr: true,
                dateFormat: "Y-m-d H:i",
                disableMobile: "true",
                locale: currentLang.value === 'zh' ? flatpickr.l10ns.zh : flatpickr.l10ns.default,
                onChange: (selectedDates, dateStr, instance) => {
                    if (instance.element.id === 'start-time-picker') {
                        customTimeRange.value.start = dateStr;
                    } else {
                        customTimeRange.value.end = dateStr;
                    }
                }
            };

            startPicker = flatpickr("#start-time-picker", pickerConfig);
            endPicker = flatpickr("#end-time-picker", pickerConfig);
        });

        const formatBytes = (bytes) => {
            if (!+bytes) return '0.00 MB';
            const gb = 1024 * 1024 * 1024;
            const mb = 1024 * 1024;
            if (bytes >= gb) {
                return (bytes / gb).toFixed(2) + ' GB';
            }
            return (bytes / mb).toFixed(2) + ' MB';
        };
        const formatNumber = (num) => new Intl.NumberFormat().format(num);

        // Dashboard Logic
        let pieChart, barChart;


        const fetchStats = async () => {
            if (!isAuthenticated.value || currentView.value !== 'dashboard') return;
            try {
                const res = await apiFetch('/api/stats');
                const data = await res.json();
                stats.value = data;

                // Update Charts
                if (pieChart && data.level_dist) {
                    const keys = Object.keys(data.level_dist);
                    if (keys.length > 0) {
                        const colors = {
                            'INFO': '#10b981',
                            'WARN': '#f59e0b',
                            'ERROR': '#ef4444',
                            'FATAL': '#7c3aed',
                            'DEBUG': '#6b7280',
                            'UNKNOWN': '#374151'
                        };
                        pieChart.data.labels = keys;
                        pieChart.data.datasets[0].data = Object.values(data.level_dist);
                        pieChart.data.datasets[0].backgroundColor = keys.map(k => colors[k] || '#374151');
                        pieChart.update();
                    }
                }
                if (barChart && data.top_services) {
                    const entries = Object.entries(data.top_services).map(([name, count]) => ({ name, count }));
                    if (entries.length > 0) {
                        entries.sort((a, b) => b.count - a.count);
                        const top5 = entries.slice(0, 5);
                        barChart.data.labels = top5.map(s => s.name);
                        barChart.data.datasets[0].data = top5.map(s => s.count);
                        barChart.update();
                    }
                }
            } catch (e) { if (e.message !== 'Unauthorized') console.error(e); }
        };

        const initDashboard = () => {
            if (pieChart) pieChart.destroy();
            if (barChart) barChart.destroy();
            if (statsInterval) clearInterval(statsInterval);

            const elPie = document.getElementById('levelChart');
            const elBar = document.getElementById('serviceChart');
            if (!elPie || !elBar) return;

            const ctxPie = elPie.getContext('2d');
            pieChart = new Chart(ctxPie, {
                type: 'doughnut',
                data: {
                    labels: [],
                    datasets: [{
                        data: [],
                        backgroundColor: ['#22c55e', '#eab308', '#ef4444', '#a855f7', '#6366f1', '#ec4899'],
                        borderWidth: 0
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'right',
                            labels: { color: '#9ca3af', font: { size: 11 }, usePointStyle: true, padding: 20 }
                        },
                        tooltip: { cornerRadius: 8, padding: 12 }
                    },
                    cutout: '70%'
                }
            });

            const ctxBar = elBar.getContext('2d');
            barChart = new Chart(ctxBar, {
                type: 'bar',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Logs',
                        data: [],
                        backgroundColor: '#06b6d4',
                        borderRadius: 6,
                        barThickness: 20
                    }]
                },
                options: {
                    indexAxis: 'y', // Horizontal
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        x: {
                            grid: { color: '#1f2937', borderDash: [2, 2] },
                            ticks: { color: '#6b7280', font: { size: 10 } },
                            beginAtZero: true
                        },
                        y: {
                            grid: { display: false },
                            ticks: { color: '#9ca3af', font: { size: 11 } }
                        }
                    },
                    plugins: {
                        legend: { display: false },
                        tooltip: { cornerRadius: 8, padding: 12 }
                    }
                }
            });

            fetchStats();
            statsInterval = setInterval(fetchStats, 2000);
        };

        // Instances Logic
        const fetchInstances = async () => {
            if (!isAuthenticated.value) return;
            try {
                const res = await apiFetch('/api/registry/instances');
                instances.value = await res.json() || [];
            } catch (e) { }
        };

        const isOnline = (lastSeen) => {
            const now = Date.now() / 1000;
            return (now - lastSeen) < 60; // < 1 minute
        };

        const formatUptime = (ts) => {
            const now = Date.now() / 1000;
            const diff = now - ts;
            if (diff < 60) return Math.floor(diff) + 's';
            if (diff < 3600) return Math.floor(diff / 60) + 'm';
            if (diff < 86400) return Math.floor(diff / 3600) + 'h';
            return Math.floor(diff / 86400) + 'd';
        };

        const switchView = (view) => {
            // Cleanup previous view state
            if (statsInterval) { clearInterval(statsInterval); statsInterval = null; }
            if (pieChart) { pieChart.destroy(); pieChart = null; }
            if (barChart) { barChart.destroy(); barChart = null; }
            if (chartInstance) { chartInstance.destroy(); chartInstance = null; }
            if (instancesInterval.value) { clearInterval(instancesInterval.value); instancesInterval.value = null; }

            currentView.value = view;

            if (view === 'dashboard') {
                setTimeout(initDashboard, 100);
            } else if (view === 'settings') {
                fetchUsers();
                fetchTokens();
                fetchConfig();
            } else if (view === 'instances') {
                fetchInstances();
                instancesInterval.value = setInterval(fetchInstances, 10000);
            } else {
                // Default to discover view
                setTimeout(initChart, 100);
                setTimeout(fetchAll, 150);
            }
        };

        onUnmounted(() => {
            if (statsInterval) clearInterval(statsInterval);
            if (refreshInterval) clearInterval(refreshInterval);
            if (chartInstance) chartInstance.destroy();
        });

        return {
            logs, filteredLogs, loading, error, searchQuery, autoRefresh, expandedIndex, currentView, stats, switchView,
            toggleRow, isJson, formatJson, formatBytes, formatNumber,
            fetchLogs: fetchAll, formatTimestamp, getLevelText, getLevelClass,
            isAuthenticated, authToken, loginForm, login, logout, showLogoutConfirm, confirmLogout, cancelLogout,
            userRole, currentUser, systemInitialized, nodeRole, settingsTab,
            users, tokens, showAddUserModal, newUser, showAddTokenModal, newToken, generatedToken,
            initForm, retentionInput, toast, showToast, showUserMenu, confirmModal, triggerConfirm,
            addUser, deleteUser, openResetPassword, resetPassword, showResetPasswordModal, resetPasswordForm,
            showChangePasswordModal, changePasswordForm, changePassword,
            generateToken, revokeToken, copyGeneratedToken, copyToken, updateRetention, initializeSystem,
            currentLang, setLang, t,
            showLangConfirm, pendingLang, confirmSwitchLanguage, executeSwitchLanguage,
            selectedTimeRange, showTimeRangeDropdown, customTimeRange,
            recentCustomRanges, timeParams, timeRangeLabel,
            updateTimeRange, applyCustomTimeRange, selectRecentCustom,
            showRecentHistory, histogramTotal, searchError, highlightText,
            contextModal, openContextModal,
            instances, isOnline, formatUptime, fetchInstances
        };
    }
}).mount('#app');
