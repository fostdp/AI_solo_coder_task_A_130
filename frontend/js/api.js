const API = {
    async get(endpoint) {
        try {
            const response = await fetch(CONFIG.API_BASE_URL + endpoint);
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            return await response.json();
        } catch (error) {
            console.error(`API GET error [${endpoint}]:`, error);
            throw error;
        }
    },

    async post(endpoint, data) {
        try {
            const response = await fetch(CONFIG.API_BASE_URL + endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            return await response.json();
        } catch (error) {
            console.error(`API POST error [${endpoint}]:`, error);
            throw error;
        }
    },

    hydrology: {
        getLatest(stationId) {
            return API.get(`/hydrology/data/latest/${stationId}`);
        },
        getAllLatest() {
            return API.get('/hydrology/data/all');
        },
        getHistory(stationId, startTime, endTime, limit = 1000) {
            const params = new URLSearchParams();
            if (startTime) params.append('start_time', startTime.toISOString());
            if (endTime) params.append('end_time', endTime.toISOString());
            params.append('limit', limit);
            return API.get(`/hydrology/data/${stationId}?${params}`);
        },
        getDailyStats(stationId, startTime, endTime) {
            const params = new URLSearchParams();
            if (startTime) params.append('start_time', startTime.toISOString());
            if (endTime) params.append('end_time', endTime.toISOString());
            return API.get(`/hydrology/stats/daily/${stationId}?${params}`);
        },
        getStations() {
            return API.get('/hydrology/stations');
        },
        submit(data) {
            return API.post('/hydrology/data', data);
        }
    },

    wolongIron: {
        getAll() {
            return API.get('/wolong-iron');
        }
    },

    alerts: {
        get(acknowledged = null, limit = 100) {
            const params = new URLSearchParams();
            if (acknowledged !== null) params.append('acknowledged', acknowledged);
            params.append('limit', limit);
            return API.get(`/alerts?${params}`);
        },
        acknowledge(alertId, acknowledgedBy) {
            return API.post(`/alerts/${alertId}/acknowledge`, { acknowledged_by: acknowledgedBy });
        }
    },

    prediction: {
        run(stationId, years = 10) {
            return API.post(`/prediction/bed-evolution/${stationId}?years=${years}`, {});
        },
        get(stationId) {
            return API.get(`/prediction/bed-evolution/${stationId}`);
        }
    },

    simulation: {
        bambooCage(location, cageCount, simulationName, createdBy) {
            return API.post('/simulation/bamboo-cage', {
                location,
                cage_count: cageCount,
                simulation_name: simulationName,
                created_by: createdBy
            });
        },
        machaInterception(location, machaCount, initialFlowRate, initialWaterLevel, simulationName, createdBy) {
            return API.post('/simulation/macha-interception', {
                location,
                macha_count: machaCount,
                initial_flow_rate: initialFlowRate,
                initial_water_level: initialWaterLevel,
                simulation_name: simulationName,
                created_by: createdBy
            });
        },
        list(limit = 50) {
            return API.get(`/simulation/list?limit=${limit}`);
        },
        getMachaData(simulationId) {
            return API.get(`/simulation/macha/${simulationId}`);
        },
        getBambooData(simulationId) {
            return API.get(`/simulation/bamboo-cage/${simulationId}`);
        }
    },

    evolution: {
        getRate(stationId) {
            return API.get(`/evolution-rate/${stationId}`);
        },
        getDEMGrid(centerX = 0, centerY = 0, gridSize = 100, resolution = 5, baseElevation = 726.5) {
            const params = new URLSearchParams({
                center_x: centerX,
                center_y: centerY,
                grid_size: gridSize,
                resolution,
                base_elevation: baseElevation
            });
            return API.get(`/dem-grid?${params}`);
        }
    },

    records: {
        getAnnualRepair() {
            return API.get('/annual-repair-records');
        }
    }
};

class WebSocketClient {
    constructor(url) {
        this.url = url;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.listeners = {};
        this.connect();
    }

    connect() {
        try {
            this.ws = new WebSocket(this.url);
            
            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.reconnectAttempts = 0;
                this.emit('open');
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.emit(data.type, data.data);
                } catch (e) {
                    console.error('WebSocket message parse error:', e);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.emit('error', error);
            };

            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this.emit('close');
                this.tryReconnect();
            };
        } catch (error) {
            console.error('WebSocket connection error:', error);
            this.tryReconnect();
        }
    }

    tryReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
            console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
            setTimeout(() => this.connect(), delay);
        }
    }

    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(cb => cb(data));
        }
    }

    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        }
    }

    close() {
        if (this.ws) {
            this.ws.close();
        }
    }
}
