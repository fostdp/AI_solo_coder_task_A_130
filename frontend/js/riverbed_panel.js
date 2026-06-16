class RiverbedPanel {
    constructor() {
        this.demRenderer = null;
        this.predictionData = null;
        this.currentYearOffset = 0;
        this.predictionChart = null;
        this.erosionChart = null;
        this.baseElevation = 726.5;
        this.isInitialized = false;
    }

    init() {
        if (this.isInitialized) return;

        this.demRenderer = new DEMRenderer('dem-2d-canvas');
        this.demRenderer.loadDEMGrid();
        this.initTimeline();
        this.initRunButton();
        this.isInitialized = true;
    }

    initTimeline() {
        const slider = document.getElementById('timeline-slider');
        const yearDisplay = document.getElementById('timeline-year');

        if (slider) {
            slider.value = 0;
            slider.max = 120;

            slider.addEventListener('input', (e) => {
                const value = parseInt(e.target.value);
                const year = 2024 + Math.floor(value / 12);
                const month = value % 12 + 1;
                if (yearDisplay) {
                    yearDisplay.textContent = `${year}年${month}月`;
                }
                this.currentYearOffset = value;
                if (this.demRenderer) {
                    this.demRenderer.setYearOffset(value);
                }
            });
        }
    }

    initRunButton() {
        const btn = document.getElementById('run-prediction');
        if (btn) {
            btn.addEventListener('click', () => this.runPrediction());
        }
    }

    async runPrediction() {
        const stationSelect = document.getElementById('evolution-station');
        const yearsSelect = document.getElementById('prediction-years');

        const stationId = stationSelect ? stationSelect.value : 'NEIJ-001';
        const years = yearsSelect ? parseInt(yearsSelect.value) : 10;

        try {
            const data = await API.prediction.run(stationId, years);
            if (data && data.predictions) {
                this.predictionData = data.predictions;
                if (this.demRenderer) {
                    this.demRenderer.setPredictionData(data.predictions);
                    this.demRenderer.setYearOffset(0);
                }

                const slider = document.getElementById('timeline-slider');
                if (slider) {
                    slider.max = years * 12;
                    slider.value = 0;
                }
                const yearDisplay = document.getElementById('timeline-year');
                if (yearDisplay) {
                    yearDisplay.textContent = '2024年1月';
                }

                this.updateStats(data);
                this.updateCharts(data);
            }
        } catch (error) {
            console.error('Prediction failed:', error);
            this.generateMockPrediction(years);
        }
    }

    generateMockPrediction(years) {
        const predictions = [];
        const totalMonths = years * 12;

        for (let i = 0; i < totalMonths; i++) {
            const yearFraction = i / 12;
            const seasonal = Math.sin(yearFraction * Math.PI * 2) * 0.05;
            const trend = yearFraction * 0.08;
            const random = (Math.random() - 0.5) * 0.02;
            const elevationChange = trend + seasonal + random;

            predictions.push({
                prediction_date: new Date(2024 + Math.floor(i / 12), i % 12, 1).toISOString(),
                bed_elevation_change: elevationChange,
                erosion_rate: Math.max(0, -seasonal * 0.5),
                deposition_rate: Math.max(0, seasonal * 0.5 + 0.006),
                sediment_accumulation: elevationChange * 1000
            });
        }

        this.predictionData = predictions;
        if (this.demRenderer) {
            this.demRenderer.setPredictionData(predictions);
            this.demRenderer.setYearOffset(0);
        }

        const avgDepo = predictions.reduce((s, p) => s + Math.max(0, p.deposition_rate), 0) / predictions.length;
        const avgErosion = predictions.reduce((s, p) => s + Math.max(0, p.erosion_rate), 0) / predictions.length;
        const finalElev = this.baseElevation + predictions[predictions.length - 1].bed_elevation_change;
        const risk = predictions[predictions.length - 1].bed_elevation_change > 0.3 ? '高' :
                     predictions[predictions.length - 1].bed_elevation_change > 0.15 ? '中' : '低';

        this.updateStats({
            average_annual_deposition: avgDepo * 12,
            average_annual_erosion: avgErosion * 12,
            final_elevation: finalElev,
            risk_level: risk
        });

        this.updateCharts({
            predictions: predictions,
            average_annual_deposition: avgDepo * 12,
            average_annual_erosion: avgErosion * 12,
            final_elevation: finalElev,
            risk_level: risk
        });
    }

    updateStats(data) {
        const el = (id) => document.getElementById(id);

        if (data.average_annual_deposition !== undefined) {
            const depEl = el('avg-deposition');
            if (depEl) depEl.textContent = (data.average_annual_deposition * 100).toFixed(2) + ' cm';
        }
        if (data.average_annual_erosion !== undefined) {
            const eroEl = el('avg-erosion');
            if (eroEl) eroEl.textContent = (data.average_annual_erosion * 100).toFixed(2) + ' cm';
        }
        if (data.final_elevation !== undefined) {
            const feEl = el('final-elevation');
            if (feEl) feEl.textContent = data.final_elevation.toFixed(2) + ' m';
        }
        if (data.risk_level !== undefined) {
            const riskEl = el('risk-level');
            if (riskEl) {
                riskEl.textContent = data.risk_level;
                riskEl.className = 'stat-value risk ' +
                    (data.risk_level === '高' ? 'high' :
                     data.risk_level === '中' ? 'medium' : 'low');
            }
        }
    }

    updateCharts(data) {
        if (!data.predictions || data.predictions.length === 0) return;

        const yearlyLabels = data.predictions.filter((_, i) => i % 12 === 0).map(p => {
            const d = new Date(p.prediction_date);
            return d.getFullYear() + '年';
        });

        const yearlyElevation = data.predictions.filter((_, i) => i % 12 === 0).map(p =>
            this.baseElevation + p.bed_elevation_change
        );

        const yearlyErosion = data.predictions.filter((_, i) => i % 12 === 0).map(p =>
            p.erosion_rate * 100
        );

        const yearlyDeposition = data.predictions.filter((_, i) => i % 12 === 0).map(p =>
            p.deposition_rate * 100
        );

        this.updatePredictionChart(yearlyLabels, yearlyElevation);
        this.updateErosionChart(yearlyLabels, yearlyDeposition, yearlyErosion);
    }

    updatePredictionChart(labels, elevationData) {
        const ctx = document.getElementById('prediction-chart');
        if (!ctx) return;

        if (this.predictionChart) {
            this.predictionChart.data.labels = labels;
            this.predictionChart.data.datasets[0].data = elevationData;
            this.predictionChart.update('none');
            return;
        }

        this.predictionChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [{
                    label: '河床高程 (m)',
                    data: elevationData,
                    borderColor: '#00d4ff',
                    backgroundColor: 'rgba(0, 212, 255, 0.1)',
                    tension: 0.4,
                    fill: true
                }, {
                    label: '卧铁高程 (m)',
                    data: labels.map(() => 726.12),
                    borderColor: '#c0c0c0',
                    borderDash: [5, 5],
                    tension: 0,
                    fill: false
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        min: 725.5,
                        max: 727.5,
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

    updateErosionChart(labels, depositionData, erosionData) {
        const ctx = document.getElementById('erosion-chart');
        if (!ctx) return;

        if (this.erosionChart) {
            this.erosionChart.data.labels = labels;
            this.erosionChart.data.datasets[0].data = depositionData;
            this.erosionChart.data.datasets[1].data = erosionData;
            this.erosionChart.update('none');
            return;
        }

        this.erosionChart = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: labels,
                datasets: [{
                    label: '淤积速率 (cm/年)',
                    data: depositionData,
                    backgroundColor: 'rgba(255, 170, 0, 0.6)',
                    borderColor: '#ffaa00',
                    borderWidth: 1
                }, {
                    label: '冲刷速率 (cm/年)',
                    data: erosionData,
                    backgroundColor: 'rgba(0, 255, 136, 0.6)',
                    borderColor: '#00ff88',
                    borderWidth: 1
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
                        ticks: { color: '#aaa' }
                    }
                },
                plugins: {
                    legend: { labels: { color: '#ccc' } }
                }
            }
        });
    }

    setYearOffset(offset) {
        this.currentYearOffset = offset;
        if (this.demRenderer) {
            this.demRenderer.setYearOffset(offset);
        }
    }

    resize() {
        if (this.demRenderer) {
            this.demRenderer.resize();
        }
    }

    destroy() {
        if (this.demRenderer) {
            this.demRenderer.destroy();
            this.demRenderer = null;
        }
        if (this.predictionChart) {
            this.predictionChart.destroy();
            this.predictionChart = null;
        }
        if (this.erosionChart) {
            this.erosionChart.destroy();
            this.erosionChart = null;
        }
        this.isInitialized = false;
    }
}
