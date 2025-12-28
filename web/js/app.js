const { createApp, ref, computed, onMounted, onUnmounted, watch } = Vue;

createApp({
    setup() {
        const logs = ref([]);
        const loading = ref(true);
        const error = ref(null);
        const searchQuery = ref('');
        const autoRefresh = ref(false);
        const expandedIndex = ref(-1);
        const currentView = ref('discover');
        const stats = ref({ ingestion_rate: 0, disk_usage: 0, total_logs: 0 });
        let refreshInterval = null;
        let statsInterval = null;

        const parseSearchQuery = (q) => {
            const params = new URLSearchParams();
            params.append('limit', '100');
            if (!q) return params.toString();

            const parts = q.split(/\s+/);
            let textSearch = [];

            parts.forEach(part => {
                if (part.includes('=')) {
                    const [key, val] = part.split('=');
                    const k = key.toLowerCase();
                    if (k === 'level') {
                        const lvls = { 'DEBUG': 0, 'INFO': 1, 'WARN': 2, 'ERROR': 3, 'FATAL': 4 };
                        const l = lvls[val.toUpperCase()];
                        params.append('level', l !== undefined ? l : val);
                    } else if (k === 'service' || k === 'svc') {
                        params.append('service', val);
                    } else if (k === 'host' || k === 'ip' || k === 'hostname') {
                        params.append('host', val);
                    } else if (k === 'start' || k === 'min_ts') {
                        params.append('start', val);
                    } else if (k === 'end' || k === 'max_ts') {
                        params.append('end', val);
                    } else {
                        textSearch.push(part);
                    }
                } else {
                    textSearch.push(part);
                }
            });

            if (textSearch.length > 0) {
                params.append('q', textSearch.join(' '));
            }
            return params.toString();
        };

        const fetchLogs = async () => {
            if (loading.value && logs.value.length > 0) return;
            loading.value = true;
            error.value = null;
            try {
                const queryString = parseSearchQuery(searchQuery.value);
                const response = await fetch(`/api/search?${queryString}`);
                if (!response.ok) throw new Error(`HTTP ${response.status}`);
                const data = await response.json();
                logs.value = data || [];
            } catch (e) {
                error.value = `Failed to fetch: ${e.message}`;
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
            const ctx = document.getElementById('logHistogram').getContext('2d');
            chartInstance = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Log Volume',
                        data: [],
                        backgroundColor: '#06b6d4', // cyan-500
                        borderRadius: 2,
                        borderSkipped: false
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        x: {
                            ticks: { color: '#6b7280', maxTicksLimit: 10 },
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
                            mode: 'index',
                            intersect: false,
                            callbacks: {
                                title: (items) => {
                                    if (items.length > 0) {
                                        const d = new Date(parseInt(items[0].label));
                                        return d.toLocaleTimeString();
                                    }
                                }
                            }
                        }
                    }
                }
            });
        };

        const fetchHistogram = async () => {
            try {
                let qs = parseSearchQuery(searchQuery.value);
                // Ensure interval is set (default 60s if not)
                // This logic could be improved to dynamic interval based on time range
                if (!qs.includes('interval=')) qs += '&interval=60';

                const res = await fetch(`/api/histogram?${qs}`);
                if (!res.ok) return;
                const data = await res.json();

                if (chartInstance) {
                    if (data && data.length > 0) {
                        chartInstance.data.labels = data.map(p => {
                            const d = new Date(p.time / 1000000);
                            return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
                        });
                        chartInstance.data.datasets[0].data = data.map(p => p.count);
                    } else {
                        chartInstance.data.labels = [];
                        chartInstance.data.datasets[0].data = [];
                    }
                    chartInstance.update();
                }
            } catch (e) {
                console.error("Histogram fetch error", e);
            }
        };

        watch(autoRefresh, (v) => {
            if (v) refreshInterval = setInterval(() => { fetchLogs(); fetchHistogram(); }, 2000); // Update both
            else if (refreshInterval) { clearInterval(refreshInterval); refreshInterval = null; }
        });

        onMounted(() => {
            initChart();
            fetchLogs();
            fetchHistogram(); // Initial load
        });

        const fetchAll = () => {
            fetchLogs();
            fetchHistogram();
        };

        const formatBytes = (bytes, decimals = 2) => {
            if (!+bytes) return '0 Bytes';
            const k = 1024, dm = decimals < 0 ? 0 : decimals, sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
        };
        const formatNumber = (num) => new Intl.NumberFormat().format(num);

        // Dashboard Logic
        let pieChart, barChart;


        const fetchStats = async () => {
            if (currentView.value !== 'dashboard') return;
            try {
                const res = await fetch('/api/stats');
                if (!res.ok) return;
                const data = await res.json();
                stats.value = data;

                // Update Charts
                if (pieChart && data.level_dist) {
                    const keys = Object.keys(data.level_dist);
                    if (keys.length > 0) {
                        pieChart.data.labels = keys;
                        pieChart.data.datasets[0].data = Object.values(data.level_dist);
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
            } catch (e) { console.error(e); }
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

        const switchView = (view) => {
            currentView.value = view;
            if (view === 'dashboard') {
                setTimeout(initDashboard, 100);
            } else {
                if (statsInterval) clearInterval(statsInterval);
                if (pieChart) pieChart.destroy();
                if (barChart) barChart.destroy();
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
            fetchLogs: fetchAll, formatTimestamp, getLevelText, getLevelClass
        };
    }
}).mount('#app');
