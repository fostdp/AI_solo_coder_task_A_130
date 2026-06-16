let mainScene = null;
let miniScene = null;
let riverbedPanel = null;
let wsClient = null;
let currentTab = 'overview';
let currentSubTab = 'macha';
let charts = {};
let latestHydrologyData = {};

document.addEventListener('DOMContentLoaded', () => {
    initTabs();
    initSubTabs();
    initLayerControls();
    initViewPresets();
    initDataControls();
    initAlertFilters();
    loadInitialData();
    initWebSocket();
    initStationMap();
    initOverviewCharts();
    initDataCharts();
    
    setTimeout(() => {
        initThreeScenes();
    }, 100);
    
    setInterval(updateSystemTime, 1000);
    window.addEventListener('resize', handleResize);
});

function initTabs() {
    const tabBtns = document.querySelectorAll('.tab-btn');
    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            switchTab(tab);
        });
    });
}

function switchTab(tabName) {
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === tabName);
    });
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === tabName);
    });
    
    currentTab = tabName;
    
    if (tabName === '3d-view') {
        setTimeout(() => {
            if (mainScene) mainScene.resize();
        }, 100);
    } else if (tabName === 'repair') {
        if (currentSubTab === 'macha') {
            initMachaSimulation();
        } else if (currentSubTab === 'bamboo') {
            initBambooSimulation();
        }
    } else if (tabName === 'evolution') {
        initRiverbedPanel();
    } else if (tabName === 'data') {
        loadStationOptions();
    } else if (tabName === 'alerts') {
        loadAlerts();
    }
}

function initSubTabs() {
    const subTabBtns = document.querySelectorAll('.sub-tab-btn');
    subTabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const subTab = btn.dataset.subtab;
            switchSubTab(subTab);
        });
    });
}

function switchSubTab(subTabName) {
    document.querySelectorAll('.sub-tab-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.subtab === subTabName);
    });
    document.querySelectorAll('.sub-tab-content').forEach(content => {
        content.classList.toggle('active', content.id === subTabName);
    });
    
    currentSubTab = subTabName;
    
    if (subTabName === 'macha') {
        initMachaSimulation();
    } else if (subTabName === 'bamboo') {
        initBambooSimulation();
    } else if (subTabName === 'records') {
        loadRepairRecords();
    }
}

function initLayerControls() {
    const layerCheckboxes = [
        { id: 'layer-terrain', layer: 'terrain' },
        { id: 'layer-water', layer: 'water' },
        { id: 'layer-structures', layer: 'structures' },
        { id: 'layer-wolong', layer: 'wolongIron' },
        { id: 'layer-stations', layer: 'stations' }
    ];
    
    layerCheckboxes.forEach(({ id, layer }) => {
        const checkbox = document.getElementById(id);
        if (checkbox) {
            checkbox.addEventListener('change', (e) => {
                if (mainScene) {
                    mainScene.setLayerVisibility(layer, e.target.checked);
                }
            });
        }
    });
    
    const waterScale = document.getElementById('water-scale');
    if (waterScale) {
        waterScale.addEventListener('input', (e) => {
            if (mainScene && mainScene.waterSystem) {
                mainScene.waterSystem.setWaterScale(parseFloat(e.target.value));
            }
        });
    }
    
    const particleCount = document.getElementById('particle-count');
    if (particleCount) {
        particleCount.addEventListener('input', (e) => {
            if (mainScene && mainScene.waterSystem) {
                mainScene.waterSystem.setParticleCount(parseInt(e.target.value));
            }
        });
    }
}

function initViewPresets() {
    const presetBtns = document.querySelectorAll('.view-presets button');
    presetBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const view = btn.textContent.trim();
            if (mainScene) {
                mainScene.setViewPreset(view);
            }
        });
    });
}

function initDataControls() {
    const refreshBtn = document.getElementById('refresh-data');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadDataCharts);
    }
    
    const timeRange = document.getElementById('time-range');
    if (timeRange) {
        timeRange.addEventListener('change', loadDataCharts);
    }
    
    const dataStation = document.getElementById('data-station');
    if (dataStation) {
        dataStation.addEventListener('change', loadDataCharts);
    }
}

function initAlertFilters() {
    const filter = document.getElementById('alert-filter');
    if (filter) {
        filter.addEventListener('change', loadAlerts);
    }
}

async function loadInitialData() {
    try {
        await Promise.all([
            loadLatestHydrologyData(),
            loadWolongIronData(),
            loadStations()
        ]);
    } catch (error) {
        console.error('加载初始数据失败:', error);
        loadMockData();
    }
}

async function loadLatestHydrologyData() {
    try {
        const data = await API.hydrology.getAllLatest();
        if (Array.isArray(data)) {
            data.forEach(item => {
                latestHydrologyData[item.station_id] = item;
            });
            updateHydrologyParams();
            updateRealtimeData();
        }
    } catch (error) {
        console.error('加载水文数据失败:', error);
        throw error;
    }
}

async function loadWolongIronData() {
    try {
        const data = await API.wolongIron.getAll();
        window.wolongIronData = data;
        updateWolongChart();
    } catch (error) {
        console.error('加载卧铁数据失败:', error);
        window.wolongIronData = [
            { id: 1, name: '卧铁1', elevation: 726.24 },
            { id: 2, name: '卧铁2', elevation: 726.18 },
            { id: 3, name: '卧铁3', elevation: 726.12 },
            { id: 4, name: '卧铁4', elevation: 726.06 }
        ];
        updateWolongChart();
    }
}

async function loadStations() {
    try {
        const data = await API.hydrology.getStations();
        window.stationsData = data;
        updateStationSelects();
    } catch (error) {
        console.error('加载站点数据失败:', error);
        window.stationsData = CONFIG.STATIONS;
        updateStationSelects();
    }
}

function loadMockData() {
    const mockData = {
        'NEIJ-001': { station_id: 'NEIJ-001', water_level: 728.5, flow_rate: 350, sediment_concentration: 0.8, bed_elevation: 726.5, timestamp: new Date().toISOString() },
        'NEIJ-002': { station_id: 'NEIJ-002', water_level: 728.3, flow_rate: 320, sediment_concentration: 0.75, bed_elevation: 726.4, timestamp: new Date().toISOString() },
        'NEIJ-003': { station_id: 'NEIJ-003', water_level: 728.1, flow_rate: 280, sediment_concentration: 0.9, bed_elevation: 726.3, timestamp: new Date().toISOString() },
        'WAIJ-001': { station_id: 'WAIJ-001', water_level: 728.6, flow_rate: 450, sediment_concentration: 1.2, bed_elevation: 726.6, timestamp: new Date().toISOString() },
        'WAIJ-002': { station_id: 'WAIJ-002', water_level: 728.4, flow_rate: 420, sediment_concentration: 1.1, bed_elevation: 726.5, timestamp: new Date().toISOString() },
        'FSSY-001': { station_id: 'FSSY-001', water_level: 728.2, flow_rate: 150, sediment_concentration: 1.5, bed_elevation: 726.2, timestamp: new Date().toISOString() },
        'FSSY-002': { station_id: 'FSSY-002', water_level: 728.0, flow_rate: 140, sediment_concentration: 1.4, bed_elevation: 726.1, timestamp: new Date().toISOString() },
        'RJK-001': { station_id: 'RJK-001', water_level: 727.8, flow_rate: 100, sediment_concentration: 0.6, bed_elevation: 725.9, timestamp: new Date().toISOString() }
    };
    
    Object.assign(latestHydrologyData, mockData);
    updateHydrologyParams();
    updateRealtimeData();
    
    window.wolongIronData = [
        { id: 1, name: '卧铁1', elevation: 726.24 },
        { id: 2, name: '卧铁2', elevation: 726.18 },
        { id: 3, name: '卧铁3', elevation: 726.12 },
        { id: 4, name: '卧铁4', elevation: 726.06 }
    ];
    updateWolongChart();
    
    window.stationsData = CONFIG.STATIONS;
    updateStationSelects();
}

function updateHydrologyParams() {
    const container = document.getElementById('hydrology-params');
    if (!container) return;
    
    container.innerHTML = '';
    
    CONFIG.STATIONS.forEach(station => {
        const data = latestHydrologyData[station.id];
        if (!data) return;
        
        const card = document.createElement('div');
        card.className = 'param-card';
        card.innerHTML = `
            <div class="param-header" style="border-left: 3px solid ${station.color}">
                <span class="param-station">${station.name}</span>
            </div>
            <div class="param-values">
                <div class="param-item">
                    <span class="param-label">水位</span>
                    <span class="param-value">${data.water_level?.toFixed(2) || '--'} m</span>
                </div>
                <div class="param-item">
                    <span class="param-label">流量</span>
                    <span class="param-value">${data.flow_rate?.toFixed(1) || '--'} m³/s</span>
                </div>
                <div class="param-item">
                    <span class="param-label">含沙量</span>
                    <span class="param-value">${data.sediment_concentration?.toFixed(2) || '--'} kg/m³</span>
                </div>
                <div class="param-item">
                    <span class="param-label">河床高程</span>
                    <span class="param-value">${data.bed_elevation?.toFixed(2) || '--'} m</span>
                </div>
            </div>
        `;
        container.appendChild(card);
    });
}

function updateRealtimeData() {
    const container = document.getElementById('realtime-data');
    if (!container) return;
    
    container.innerHTML = '';
    
    CONFIG.STATIONS.slice(0, 4).forEach(station => {
        const data = latestHydrologyData[station.id];
        if (!data) return;
        
        const item = document.createElement('div');
        item.className = 'realtime-item';
        item.innerHTML = `
            <div class="realtime-station" style="color: ${station.color}">${station.name}</div>
            <div class="realtime-values">
                <span>水位: ${data.water_level?.toFixed(1) || '--'}m</span>
                <span>流量: ${data.flow_rate?.toFixed(0) || '--'}m³/s</span>
            </div>
        `;
        container.appendChild(item);
    });
}

function updateSystemTime() {
    const timeEl = document.getElementById('data-time');
    if (timeEl) {
        timeEl.textContent = formatDateTime(new Date());
    }
}

function initWebSocket() {
    try {
        wsClient = new WebSocketClient(CONFIG.WS_URL);
        
        wsClient.on('hydrology', (data) => {
            latestHydrologyData[data.station_id] = data;
            updateHydrologyParams();
            updateRealtimeData();
            updateStationMap();
            
            if (currentTab === 'data') {
                updateDataChartsWithNewData(data);
            }
            if (currentTab === 'overview') {
                updateEvolutionChartWithNewData(data);
            }
        });
        
        wsClient.on('alert', (data) => {
            showAlertToast(data);
            updateAlertCount();
            if (currentTab === 'alerts') {
                loadAlerts();
            }
        });
        
        wsClient.on('connect', () => {
            document.getElementById('system-status').textContent = '连接中';
            document.getElementById('system-status').className = 'status-value connected';
        });
        
        wsClient.on('disconnect', () => {
            document.getElementById('system-status').textContent = '断开';
            document.getElementById('system-status').className = 'status-value disconnected';
        });
    } catch (error) {
        console.error('WebSocket初始化失败:', error);
        document.getElementById('system-status').textContent = '离线模式';
        document.getElementById('system-status').className = 'status-value';
        
        setInterval(() => {
            simulateRealtimeData();
        }, 5000);
    }
}

function simulateRealtimeData() {
    Object.keys(latestHydrologyData).forEach(stationId => {
        const data = latestHydrologyData[stationId];
        if (data) {
            data.water_level += (Math.random() - 0.5) * 0.02;
            data.flow_rate += (Math.random() - 0.5) * 5;
            data.sediment_concentration += (Math.random() - 0.5) * 0.02;
            data.bed_elevation += (Math.random() - 0.5) * 0.005;
            data.timestamp = new Date().toISOString();
        }
    });
    
    updateHydrologyParams();
    updateRealtimeData();
    updateStationMap();
}

function initThreeScenes() {
    try {
        const mainContainer = document.getElementById('three-container');
        const miniContainer = document.getElementById('mini-3d-container');
        
        if (mainContainer) {
            mainScene = new Dujiangyan3D('three-container');
            mainScene.init();
        }
        
        if (miniContainer) {
            miniScene = new Dujiangyan3D('mini-3d-container');
            miniScene.init();
        }
    } catch (error) {
        console.error('Three.js场景初始化失败:', error);
    }
}

function initRiverbedPanel() {
    if (!riverbedPanel) {
        riverbedPanel = new RiverbedPanel();
        riverbedPanel.init();
    }
}

function initStationMap() {
    const canvas = document.getElementById('station-canvas');
    if (!canvas) return;
    
    const ctx = canvas.getContext('2d');
    const rect = canvas.parentElement.getBoundingClientRect();
    canvas.width = rect.width;
    canvas.height = 200;
    
    drawStationMap(ctx, canvas.width, canvas.height);
}

function drawStationMap(ctx, width, height) {
    const gradient = ctx.createLinearGradient(0, 0, width, height);
    gradient.addColorStop(0, '#1a2a4a');
    gradient.addColorStop(1, '#2d4a6a');
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, width, height);
    
    ctx.strokeStyle = 'rgba(0, 168, 255, 0.4)';
    ctx.lineWidth = 8;
    ctx.lineCap = 'round';
    ctx.beginPath();
    ctx.moveTo(0, height * 0.5);
    ctx.bezierCurveTo(width * 0.3, height * 0.3, width * 0.4, height * 0.7, width * 0.6, height * 0.5);
    ctx.bezierCurveTo(width * 0.75, height * 0.35, width * 0.85, height * 0.65, width, height * 0.5);
    ctx.stroke();
    
    ctx.strokeStyle = 'rgba(0, 168, 255, 0.3)';
    ctx.lineWidth = 5;
    ctx.beginPath();
    ctx.moveTo(width * 0.5, height * 0.5);
    ctx.quadraticCurveTo(width * 0.7, height * 0.2, width * 0.9, height * 0.15);
    ctx.stroke();
    
    const stationPositions = {
        'NEIJ-001': { x: 0.35, y: 0.45 },
        'NEIJ-002': { x: 0.55, y: 0.42 },
        'NEIJ-003': { x: 0.75, y: 0.2 },
        'WAIJ-001': { x: 0.25, y: 0.55 },
        'WAIJ-002': { x: 0.65, y: 0.6 },
        'FSSY-001': { x: 0.45, y: 0.38 },
        'FSSY-002': { x: 0.5, y: 0.58 },
        'RJK-001': { x: 0.6, y: 0.3 }
    };
    
    CONFIG.STATIONS.forEach(station => {
        const pos = stationPositions[station.id];
        if (!pos) return;
        
        const x = width * pos.x;
        const y = height * pos.y;
        const data = latestHydrologyData[station.id];
        
        ctx.fillStyle = station.color;
        ctx.strokeStyle = '#fff';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.arc(x, y, 8, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        
        if (data && data.bed_elevation > 726.2) {
            ctx.fillStyle = '#ff4444';
            ctx.beginPath();
            ctx.arc(x, y, 12, 0, Math.PI * 2);
            ctx.globalAlpha = 0.3 + Math.sin(Date.now() * 0.005) * 0.2;
            ctx.fill();
            ctx.globalAlpha = 1;
        }
        
        ctx.fillStyle = '#fff';
        ctx.font = '10px Arial';
        ctx.textAlign = 'center';
        ctx.fillText(station.name, x, y + 22);
    });
}

function updateStationMap() {
    const canvas = document.getElementById('station-canvas');
    if (!canvas) return;
    
    const ctx = canvas.getContext('2d');
    drawStationMap(ctx, canvas.width, canvas.height);
}

function initOverviewCharts() {
    initEvolutionChart();
    initWolongChart();
}

function initEvolutionChart() {
    const ctx = document.getElementById('evolution-chart');
    if (!ctx) return;
    
    const labels = [];
    const data = [];
    const now = new Date();
    
    for (let i = 30; i >= 0; i--) {
        const date = new Date(now - i * 24 * 60 * 60 * 1000);
        labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        data.push(726.4 + Math.sin(i * 0.3) * 0.1 + (Math.random() - 0.5) * 0.05);
    }
    
    charts.evolution = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: '内江中段河床高程 (m)',
                data: data,
                borderColor: '#00d4ff',
                backgroundColor: 'rgba(0, 212, 255, 0.1)',
                tension: 0.4,
                fill: true
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    min: 726.2,
                    max: 726.8,
                    grid: { color: 'rgba(255,255,255,0.1)' },
                    ticks: { color: '#aaa' }
                },
                x: {
                    grid: { color: 'rgba(255,255,255,0.1)' },
                    ticks: { color: '#aaa', maxTicksLimit: 7 }
                }
            },
            plugins: {
                legend: { display: false }
            }
        }
    });
}

function initWolongChart() {
    const ctx = document.getElementById('wolong-chart');
    if (!ctx || !window.wolongIronData) return;
    
    const labels = window.wolongIronData.map(w => w.name);
    const elevations = window.wolongIronData.map(w => w.elev);
    const bedElevations = [726.5, 726.45, 726.38, 726.35];
    
    charts.wolong = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: labels,
            datasets: [{
                label: '卧铁高程 (m)',
                data: elevations,
                backgroundColor: 'rgba(192, 192, 192, 0.8)',
                borderColor: '#c0c0c0',
                borderWidth: 1
            }, {
                label: '当前河床高程 (m)',
                data: bedElevations,
                backgroundColor: 'rgba(255, 170, 0, 0.8)',
                borderColor: '#ffaa00',
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    min: 725.8,
                    max: 726.8,
                    grid: { color: 'rgba(255,255,255,0.1)' },
                    ticks: { color: '#aaa' }
                },
                x: {
                    grid: { color: 'rgba(255,255,255,0.1)' },
                    ticks: { color: '#aaa' }
                }
            },
            plugins: {
                legend: { labels: { color: '#ccc' } }
            }
        }
    });
}

function updateWolongChart() {
    if (charts.wolong && window.wolongIronData) {
        charts.wolong.data.labels = window.wolongIronData.map(w => w.name);
        charts.wolong.data.datasets[0].data = window.wolongIronData.map(w => w.elev);
        charts.wolong.update('none');
    }
}

function updateEvolutionChartWithNewData(data) {
    if (!charts.evolution || data.station_id !== 'NEIJ-002') return;
    
    charts.evolution.data.labels.push(`${new Date().getMonth() + 1}/${new Date().getDate()}`);
    charts.evolution.data.datasets[0].data.push(data.bed_elevation);
    
    if (charts.evolution.data.labels.length > 31) {
        charts.evolution.data.labels.shift();
        charts.evolution.data.datasets[0].data.shift();
    }
    
    charts.evolution.update('none');
}

function updateStationSelects() {
    const dataSelect = document.getElementById('data-station');
    if (dataSelect && window.stationsData) {
        dataSelect.innerHTML = '';
        window.stationsData.forEach(station => {
            const option = document.createElement('option');
            option.value = station.id || station.station_id;
            option.textContent = station.name;
            dataSelect.appendChild(option);
        });
    }
}

function loadStationOptions() {
    const dataSelect = document.getElementById('data-station');
    if (dataSelect && dataSelect.options.length === 0 && window.stationsData) {
        updateStationSelects();
    }
}

function initDataCharts() {
    const params = ['water-level', 'flow-rate', 'sediment', 'bed-elevation'];
    const titles = ['水位变化 (m)', '流量变化 (m³/s)', '含沙量变化 (kg/m³)', '河床高程变化 (m)'];
    const colors = ['#00d4ff', '#00ff88', '#ffaa00', '#ff66cc'];
    
    params.forEach((param, index) => {
        const ctx = document.getElementById(`${param}-chart`);
        if (!ctx) return;
        
        charts[param] = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: titles[index],
                    data: [],
                    borderColor: colors[index],
                    backgroundColor: colors[index].replace(')', ', 0.1)').replace('rgb', 'rgba'),
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        grid: { color: 'rgba(255,255,255,0.1)' },
                        ticks: { color: '#aaa' }
                    },
                    x: {
                        grid: { color: 'rgba(255,255,255,0.1)' },
                        ticks: { color: '#aaa', maxTicksLimit: 8 }
                    }
                },
                plugins: {
                    legend: { display: false }
                }
            }
        });
    });
}

async function loadDataCharts() {
    const stationId = document.getElementById('data-station')?.value;
    const hours = parseInt(document.getElementById('time-range')?.value || '24');
    
    if (!stationId) return;
    
    try {
        const endTime = new Date();
        const startTime = new Date(endTime - hours * 60 * 60 * 1000);
        
        const data = await API.hydrology.getHistory(stationId, startTime, endTime, 200);
        
        if (Array.isArray(data)) {
            const labels = data.map(d => formatDateTime(new Date(d.timestamp)));
            const waterLevels = data.map(d => d.water_level);
            const flowRates = data.map(d => d.flow_rate);
            const sediments = data.map(d => d.sediment_concentration);
            const bedElevations = data.map(d => d.bed_elevation);
            
            updateChart('water-level', labels, waterLevels);
            updateChart('flow-rate', labels, flowRates);
            updateChart('sediment', labels, sediments);
            updateChart('bed-elevation', labels, bedElevations);
        }
    } catch (error) {
        console.error('加载数据图表失败:', error);
        loadMockDataCharts(hours);
    }
}

function loadMockDataCharts(hours) {
    const dataPoints = Math.min(hours, 100);
    const labels = [];
    const waterLevels = [];
    const flowRates = [];
    const sediments = [];
    const bedElevations = [];
    
    const now = new Date();
    for (let i = dataPoints - 1; i >= 0; i--) {
        const time = new Date(now - i * (hours / dataPoints) * 60 * 60 * 1000);
        labels.push(formatDateTime(time));
        
        const t = i / dataPoints;
        waterLevels.push(728.5 + Math.sin(t * Math.PI * 4) * 0.3 + (Math.random() - 0.5) * 0.1);
        flowRates.push(350 + Math.sin(t * Math.PI * 3) * 100 + (Math.random() - 0.5) * 20);
        sediments.push(0.8 + Math.sin(t * Math.PI * 5) * 0.4 + (Math.random() - 0.5) * 0.1);
        bedElevations.push(726.5 + Math.sin(t * Math.PI * 2) * 0.2 + t * 0.1);
    }
    
    updateChart('water-level', labels, waterLevels);
    updateChart('flow-rate', labels, flowRates);
    updateChart('sediment', labels, sediments);
    updateChart('bed-elevation', labels, bedElevations);
}

function updateChart(param, labels, data) {
    if (charts[param]) {
        charts[param].data.labels = labels;
        charts[param].data.datasets[0].data = data;
        charts[param].update('none');
    }
}

function updateDataChartsWithNewData(data) {
    const selectedStation = document.getElementById('data-station')?.value;
    if (selectedStation !== data.station_id) return;
    
    const label = formatDateTime(new Date(data.timestamp));
    
    if (charts['water-level']) {
        charts['water-level'].data.labels.push(label);
        charts['water-level'].data.datasets[0].data.push(data.water_level);
        if (charts['water-level'].data.labels.length > 100) {
            charts['water-level'].data.labels.shift();
            charts['water-level'].data.datasets[0].data.shift();
        }
        charts['water-level'].update('none');
    }
    
    if (charts['flow-rate']) {
        charts['flow-rate'].data.labels.push(label);
        charts['flow-rate'].data.datasets[0].data.push(data.flow_rate);
        if (charts['flow-rate'].data.labels.length > 100) {
            charts['flow-rate'].data.labels.shift();
            charts['flow-rate'].data.datasets[0].data.shift();
        }
        charts['flow-rate'].update('none');
    }
    
    if (charts['sediment']) {
        charts['sediment'].data.labels.push(label);
        charts['sediment'].data.datasets[0].data.push(data.sediment_concentration);
        if (charts['sediment'].data.labels.length > 100) {
            charts['sediment'].data.labels.shift();
            charts['sediment'].data.datasets[0].data.shift();
        }
        charts['sediment'].update('none');
    }
    
    if (charts['bed-elevation']) {
        charts['bed-elevation'].data.labels.push(label);
        charts['bed-elevation'].data.datasets[0].data.push(data.bed_elevation);
        if (charts['bed-elevation'].data.labels.length > 100) {
            charts['bed-elevation'].data.labels.shift();
            charts['bed-elevation'].data.datasets[0].data.shift();
        }
        charts['bed-elevation'].update('none');
    }
}

async function loadAlerts() {
    const filter = document.getElementById('alert-filter')?.value;
    const container = document.getElementById('alerts-list');
    if (!container) return;
    
    try {
        let acknowledged = null;
        let level = null;
        
        if (filter === 'unacknowledged') acknowledged = false;
        else if (filter === 'acknowledged') acknowledged = true;
        else if (filter === 'CRITICAL' || filter === 'WARNING' || filter === 'NOTICE') level = filter;
        
        let data = await API.alerts.get(acknowledged, 50);
        
        if (level) {
            data = data.filter(a => a.level === level);
        }
        
        renderAlerts(data);
        updateAlertCount(data.filter(a => !a.acknowledged).length);
    } catch (error) {
        console.error('加载告警失败:', error);
        loadMockAlerts();
    }
}

function loadMockAlerts() {
    const mockAlerts = [
        {
            id: 1,
            station_id: 'NEIJ-003',
            station_name: '宝瓶口上游',
            level: 'CRITICAL',
            message: '河床高程726.35m已超过卧铁高程726.12m，超覆0.23m',
            bed_elevation: 726.35,
            wolong_elevation: 726.12,
            elevation_diff: 0.23,
            acknowledged: false,
            created_at: new Date().toISOString()
        },
        {
            id: 2,
            station_id: 'NEIJ-002',
            station_name: '内江中段',
            level: 'WARNING',
            message: '河床高程726.45m接近卧铁高程726.18m，超覆0.27m',
            bed_elevation: 726.45,
            wolong_elevation: 726.18,
            elevation_diff: 0.27,
            acknowledged: false,
            created_at: new Date(Date.now() - 3600000).toISOString()
        },
        {
            id: 3,
            station_id: 'FSSY-001',
            station_name: '飞沙堰进口',
            level: 'NOTICE',
            message: '含沙量1.8kg/m³超过阈值1.5kg/m³',
            bed_elevation: 726.2,
            wolong_elevation: null,
            elevation_diff: null,
            acknowledged: true,
            acknowledged_by: '系统管理员',
            acknowledged_at: new Date(Date.now() - 7200000).toISOString(),
            created_at: new Date(Date.now() - 10800000).toISOString()
        }
    ];
    
    renderAlerts(mockAlerts);
    updateAlertCount(2);
}

function renderAlerts(alerts) {
    const container = document.getElementById('alerts-list');
    if (!container) return;
    
    if (alerts.length === 0) {
        container.innerHTML = '<div class="no-alerts">暂无告警信息</div>';
        return;
    }
    
    container.innerHTML = alerts.map(alert => {
        const levelClass = alert.level.toLowerCase();
        const levelText = { CRITICAL: '严重', WARNING: '警告', NOTICE: '通知' }[alert.level] || alert.level;
        
        return `
            <div class="alert-item ${levelClass} ${alert.acknowledged ? 'acknowledged' : ''}">
                <div class="alert-header">
                    <span class="alert-level ${levelClass}">${levelText}</span>
                    <span class="alert-station">${alert.station_name || alert.station_id}</span>
                    <span class="alert-time">${formatDateTime(new Date(alert.created_at))}</span>
                </div>
                <div class="alert-message">${alert.message}</div>
                ${alert.bed_elevation !== undefined ? `
                    <div class="alert-details">
                        <span>河床高程: ${alert.bed_elevation.toFixed(2)}m</span>
                        ${alert.wolong_elevation ? `<span>卧铁高程: ${alert.wolong_elevation.toFixed(2)}m</span>` : ''}
                        ${alert.elevation_diff !== null ? `<span>超覆高度: ${alert.elevation_diff.toFixed(2)}m</span>` : ''}
                    </div>
                ` : ''}
                ${alert.acknowledged ? `
                    <div class="alert-acknowledged">
                        已由 ${alert.acknowledged_by || '系统'} 于 ${formatDateTime(new Date(alert.acknowledged_at))} 确认
                    </div>
                ` : `
                    <button class="btn-acknowledge" onclick="acknowledgeAlert(${alert.id})">确认告警</button>
                `}
            </div>
        `;
    }).join('');
}

function updateAlertCount(count) {
    const countEl = document.getElementById('alert-count');
    if (countEl) {
        if (typeof count === 'number') {
            countEl.textContent = count;
        } else {
            const current = parseInt(countEl.textContent) || 0;
            countEl.textContent = current + 1;
        }
    }
}

async function acknowledgeAlert(alertId) {
    try {
        await API.alerts.acknowledge(alertId, '前端用户');
        loadAlerts();
    } catch (error) {
        console.error('确认告警失败:', error);
        loadAlerts();
    }
}

function showAlertToast(alert) {
    const toast = document.getElementById('alert-toast');
    const title = document.getElementById('toast-title');
    const message = document.getElementById('toast-message');
    
    if (!toast || !title || !message) return;
    
    const levelText = { CRITICAL: '严重告警', WARNING: '警告', NOTICE: '通知' }[alert.level] || '新告警';
    
    title.textContent = `${levelText} - ${alert.station_name || alert.station_id}`;
    message.textContent = alert.message;
    
    toast.className = `alert-toast ${alert.level.toLowerCase()}`;
    
    setTimeout(() => {
        closeToast();
    }, 5000);
}

function closeToast() {
    const toast = document.getElementById('alert-toast');
    if (toast) {
        toast.className = 'alert-toast hidden';
    }
}

async function loadRepairRecords() {
    const container = document.getElementById('records-body');
    if (!container) return;
    
    try {
        const data = await API.records.getAnnualRepair();
        renderRepairRecords(data);
    } catch (error) {
        console.error('加载岁修记录失败:', error);
        loadMockRepairRecords();
    }
}

function loadMockRepairRecords() {
    const mockRecords = [
        { year: 2024, location: '内江', repair_type: '岁修', bamboo_cages: 120, macha: 35, sediment_removed: 2500, elevation_before: 726.5, elevation_after: 726.2, notes: '常规岁修' },
        { year: 2023, location: '外江', repair_type: '岁修', bamboo_cages: 150, macha: 40, sediment_removed: 3200, elevation_before: 726.6, elevation_after: 726.15, notes: '重点加固飞沙堰' },
        { year: 2022, location: '内江', repair_type: '抢修', bamboo_cages: 80, macha: 25, sediment_removed: 1800, elevation_before: 726.45, elevation_after: 726.25, notes: '汛期后抢修' },
        { year: 2021, location: '宝瓶口', repair_type: '岁修', bamboo_cages: 90, macha: 20, sediment_removed: 1500, elevation_before: 726.35, elevation_after: 726.1, notes: '宝瓶口清淤' }
    ];
    
    renderRepairRecords(mockRecords);
}

function renderRepairRecords(records) {
    const container = document.getElementById('records-body');
    if (!container) return;
    
    container.innerHTML = records.map(record => `
        <tr>
            <td>${record.year}</td>
            <td>${record.location}</td>
            <td>${record.repair_type}</td>
            <td>${record.bamboo_cages}</td>
            <td>${record.macha}</td>
            <td>${record.sediment_removed}</td>
            <td>${record.elevation_before.toFixed(2)}</td>
            <td>${record.elevation_after.toFixed(2)}</td>
            <td>${record.notes || '-'}</td>
        </tr>
    `).join('');
}

function handleResize() {
    if (mainScene) mainScene.resize();
    if (miniScene) miniScene.resize();
    if (riverbedPanel) riverbedPanel.resize();
    if (typeof machaSim !== 'undefined' && machaSim) machaSim.initCanvas();
    if (typeof bambooSim !== 'undefined' && bambooSim) bambooSim.initCanvas();
    
    Object.values(charts).forEach(chart => {
        if (chart.resize) chart.resize();
    });
    
    initStationMap();
}

function setView(viewName) {
    if (mainScene) {
        mainScene.setViewPreset(viewName);
    }
}
