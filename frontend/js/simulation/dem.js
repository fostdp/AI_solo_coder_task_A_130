class DEMRenderer {
    constructor(canvasId) {
        this.canvas = document.getElementById(canvasId);
        this.ctx = this.canvas.getContext('2d');
        this.gridSize = 100;
        this.resolution = 2;
        this.baseElevation = 726.5;
        this.demGrid = null;
        this.currentYearOffset = 0;
        this.predictionData = null;
        this.initCanvas();
    }

    initCanvas() {
        const rect = this.canvas.parentElement.getBoundingClientRect();
        this.canvas.width = rect.width;
        this.canvas.height = 400;
        this.width = this.canvas.width;
        this.height = this.canvas.height;
    }

    async loadDEMGrid(centerX = 0, centerY = 0, gridSize = 100, resolution = 2) {
        this.gridSize = gridSize;
        this.resolution = resolution;
        
        try {
            const data = await API.evolution.getDEMGrid(centerX, centerY, gridSize, resolution, this.baseElevation);
            this.demGrid = data;
            this.render();
        } catch (error) {
            console.error('加载DEM数据失败:', error);
            this.generateMockDEM();
            this.render();
        }
    }

    generateMockDEM() {
        const cols = Math.ceil(this.gridSize / this.resolution) + 1;
        const rows = cols;
        this.demGrid = {
            center_x: 0,
            center_y: 0,
            grid_size: this.gridSize,
            resolution: this.resolution,
            base_elevation: this.baseElevation,
            elevations: []
        };

        for (let j = 0; j < rows; j++) {
            const row = [];
            for (let i = 0; i < cols; i++) {
                const x = (i - cols / 2) * this.resolution;
                const y = (j - rows / 2) * this.resolution;
                
                let elevation = this.baseElevation;
                
                const distFromCenter = Math.sqrt(x * x + y * y);
                elevation -= Math.exp(-distFromCenter * distFromCenter / 1000) * 3;
                
                const riverX = Math.abs(y - 5);
                elevation -= Math.exp(-riverX * riverX / 50) * 4;
                
                if (x > 20 && x < 40 && y > -10 && y < 20) {
                    elevation -= 2;
                }
                
                elevation += (Math.random() - 0.5) * 0.3;
                
                row.push(elevation);
            }
            this.demGrid.elevations.push(row);
        }
    }

    setPredictionData(predictionData) {
        this.predictionData = predictionData;
    }

    setYearOffset(yearOffset) {
        this.currentYearOffset = yearOffset;
        this.render();
    }

    render() {
        if (!this.demGrid) return;

        this.ctx.clearRect(0, 0, this.width, this.height);

        const elevations = this.demGrid.elevations;
        const rows = elevations.length;
        const cols = elevations[0].length;
        
        const cellWidth = this.width / cols;
        const cellHeight = this.height / rows;

        let minElev = Infinity;
        let maxElev = -Infinity;
        
        elevations.forEach(row => {
            row.forEach(elev => {
                const adjustedElev = this.getAdjustedElevation(elev);
                minElev = Math.min(minElev, adjustedElev);
                maxElev = Math.max(maxElev, adjustedElev);
            });
        });

        for (let j = 0; j < rows; j++) {
            for (let i = 0; i < cols; i++) {
                const elevation = this.getAdjustedElevation(elevations[j][i]);
                const color = this.getElevationColor(elevation, minElev, maxElev);
                
                this.ctx.fillStyle = color;
                this.ctx.fillRect(
                    i * cellWidth,
                    j * cellHeight,
                    cellWidth + 1,
                    cellHeight + 1
                );
            }
        }

        this.drawContours(elevations, minElev, maxElev, cellWidth, cellHeight);
        this.drawColorBar(minElev, maxElev);
        this.drawRiverChannel();
        this.drawWolongIronMarkers();
    }

    getAdjustedElevation(baseElevation) {
        if (!this.predictionData || this.predictionData.length === 0) {
            return baseElevation;
        }

        const monthIndex = Math.min(
            Math.floor(this.currentYearOffset),
            this.predictionData.length - 1
        );
        const prediction = this.predictionData[monthIndex];
        
        if (prediction && prediction.bed_elevation_change !== undefined) {
            return baseElevation + prediction.bed_elevation_change;
        }
        
        return baseElevation;
    }

    getElevationColor(elevation, minElev, maxElev) {
        const range = maxElev - minElev || 1;
        const normalized = (elevation - minElev) / range;
        
        let r, g, b;
        
        if (normalized < 0.25) {
            const t = normalized / 0.25;
            r = 0;
            g = Math.floor(100 + t * 50);
            b = Math.floor(200 + t * 55);
        } else if (normalized < 0.5) {
            const t = (normalized - 0.25) / 0.25;
            r = Math.floor(t * 100);
            g = Math.floor(150 + t * 105);
            b = Math.floor(255 - t * 100);
        } else if (normalized < 0.75) {
            const t = (normalized - 0.5) / 0.25;
            r = Math.floor(100 + t * 155);
            g = Math.floor(255 - t * 55);
            b = Math.floor(155 - t * 100);
        } else {
            const t = (normalized - 0.75) / 0.25;
            r = Math.floor(255 - t * 55);
            g = Math.floor(200 - t * 100);
            b = Math.floor(55 + t * 50);
        }
        
        return `rgb(${r}, ${g}, ${b})`;
    }

    drawContours(elevations, minElev, maxElev, cellWidth, cellHeight) {
        const contourInterval = 0.5;
        const startElev = Math.ceil(minElev / contourInterval) * contourInterval;
        
        this.ctx.strokeStyle = 'rgba(0, 0, 0, 0.3)';
        this.ctx.lineWidth = 1;

        for (let elev = startElev; elev <= maxElev; elev += contourInterval) {
            this.ctx.beginPath();
            
            for (let j = 0; j < elevations.length - 1; j++) {
                for (let i = 0; i < elevations[0].length - 1; i++) {
                    const e00 = this.getAdjustedElevation(elevations[j][i]);
                    const e10 = this.getAdjustedElevation(elevations[j][i + 1]);
                    const e01 = this.getAdjustedElevation(elevations[j + 1][i]);
                    const e11 = this.getAdjustedElevation(elevations[j + 1][i + 1]);

                    const crossings = [];

                    if ((e00 - elev) * (e10 - elev) < 0) {
                        const t = (elev - e00) / (e10 - e00);
                        crossings.push({
                            x: (i + t) * cellWidth,
                            y: j * cellHeight
                        });
                    }
                    if ((e10 - elev) * (e11 - elev) < 0) {
                        const t = (elev - e10) / (e11 - e10);
                        crossings.push({
                            x: (i + 1) * cellWidth,
                            y: (j + t) * cellHeight
                        });
                    }
                    if ((e01 - elev) * (e11 - elev) < 0) {
                        const t = (elev - e01) / (e11 - e01);
                        crossings.push({
                            x: (i + t) * cellWidth,
                            y: (j + 1) * cellHeight
                        });
                    }
                    if ((e00 - elev) * (e01 - elev) < 0) {
                        const t = (elev - e00) / (e01 - e00);
                        crossings.push({
                            x: i * cellWidth,
                            y: (j + t) * cellHeight
                        });
                    }

                    if (crossings.length >= 2) {
                        this.ctx.moveTo(crossings[0].x, crossings[0].y);
                        this.ctx.lineTo(crossings[1].x, crossings[1].y);
                    }
                }
            }
            
            this.ctx.stroke();
        }
    }

    drawColorBar(minElev, maxElev) {
        const barWidth = 30;
        const barHeight = this.height - 40;
        const barX = this.width - barWidth - 20;
        const barY = 20;

        const gradient = this.ctx.createLinearGradient(barX, barY + barHeight, barX, barY);
        gradient.addColorStop(0, this.getElevationColor(minElev, minElev, maxElev));
        gradient.addColorStop(0.25, this.getElevationColor(minElev + (maxElev - minElev) * 0.25, minElev, maxElev));
        gradient.addColorStop(0.5, this.getElevationColor(minElev + (maxElev - minElev) * 0.5, minElev, maxElev));
        gradient.addColorStop(0.75, this.getElevationColor(minElev + (maxElev - minElev) * 0.75, minElev, maxElev));
        gradient.addColorStop(1, this.getElevationColor(maxElev, minElev, maxElev));

        this.ctx.fillStyle = gradient;
        this.ctx.fillRect(barX, barY, barWidth, barHeight);

        this.ctx.strokeStyle = '#fff';
        this.ctx.lineWidth = 2;
        this.ctx.strokeRect(barX, barY, barWidth, barHeight);

        this.ctx.fillStyle = '#fff';
        this.ctx.font = '12px Arial';
        this.ctx.textAlign = 'left';
        
        const steps = 5;
        for (let i = 0; i <= steps; i++) {
            const t = i / steps;
            const elev = minElev + (maxElev - minElev) * t;
            const y = barY + barHeight - t * barHeight;
            
            this.ctx.fillText(elev.toFixed(1) + 'm', barX + barWidth + 10, y + 4);
            
            this.ctx.beginPath();
            this.ctx.moveTo(barX - 5, y);
            this.ctx.lineTo(barX, y);
            this.ctx.stroke();
        }
    }

    drawRiverChannel() {
        const cols = this.demGrid.elevations[0].length;
        const rows = this.demGrid.elevations.length;
        const cellWidth = this.width / cols;
        const cellHeight = this.height / rows;

        this.ctx.strokeStyle = 'rgba(0, 168, 255, 0.8)';
        this.ctx.lineWidth = 3;
        this.ctx.setLineDash([10, 5]);
        this.ctx.beginPath();
        
        for (let i = 0; i < cols; i++) {
            const x = i * cellWidth + cellWidth / 2;
            const baseY = rows / 2 + Math.sin(i * 0.1) * 3;
            const y = baseY * cellHeight + cellHeight / 2;
            
            if (i === 0) {
                this.ctx.moveTo(x, y);
            } else {
                this.ctx.lineTo(x, y);
            }
        }
        this.ctx.stroke();
        this.ctx.setLineDash([]);

        this.ctx.fillStyle = 'rgba(0, 168, 255, 0.9)';
        this.ctx.font = '14px Arial';
        this.ctx.textAlign = 'center';
        this.ctx.fillText('岷江主河道', this.width / 2, 40);
    }

    drawWolongIronMarkers() {
        if (this.demGrid && this.demGrid.wolong_iron) {
            this.demGrid.wolong_iron.forEach(iron => {
                const cols = this.demGrid.elevations[0].length;
                const rows = this.demGrid.elevations.length;
                const cellWidth = this.width / cols;
                const cellHeight = this.height / rows;
                
                const x = (iron.x - this.demGrid.center_x + this.demGrid.grid_size / 2) / this.demGrid.resolution * cellWidth;
                const y = (iron.y - this.demGrid.center_y + this.demGrid.grid_size / 2) / this.demGrid.resolution * cellHeight;

                this.ctx.fillStyle = CONFIG.COLORS.wolongIron;
                this.ctx.strokeStyle = '#000';
                this.ctx.lineWidth = 2;
                
                this.ctx.beginPath();
                this.ctx.moveTo(x, y - 15);
                this.ctx.lineTo(x + 8, y + 10);
                this.ctx.lineTo(x - 8, y + 10);
                this.ctx.closePath();
                this.ctx.fill();
                this.ctx.stroke();

                this.ctx.fillStyle = '#fff';
                this.ctx.font = '10px Arial';
                this.ctx.textAlign = 'center';
                this.ctx.fillText(iron.name, x, y + 25);
                this.ctx.fillText(iron.elevation.toFixed(2) + 'm', x, y + 37);
            });
        } else {
            const markers = [
                { name: '卧铁1', elev: 726.24, x: 0.3, y: 0.5 },
                { name: '卧铁2', elev: 726.18, x: 0.35, y: 0.52 },
                { name: '卧铁3', elev: 726.12, x: 0.4, y: 0.54 },
                { name: '卧铁4', elev: 726.06, x: 0.45, y: 0.56 }
            ];

            markers.forEach(marker => {
                const x = this.width * marker.x;
                const y = this.height * marker.y;

                this.ctx.fillStyle = CONFIG.COLORS.wolongIron;
                this.ctx.strokeStyle = '#000';
                this.ctx.lineWidth = 2;
                
                this.ctx.beginPath();
                this.ctx.moveTo(x, y - 15);
                this.ctx.lineTo(x + 8, y + 10);
                this.ctx.lineTo(x - 8, y + 10);
                this.ctx.closePath();
                this.ctx.fill();
                this.ctx.stroke();

                this.ctx.fillStyle = '#fff';
                this.ctx.font = '10px Arial';
                this.ctx.textAlign = 'center';
                this.ctx.fillText(marker.name, x, y + 25);
                this.ctx.fillText(marker.elev.toFixed(2) + 'm', x, y + 37);
            });
        }
    }

    resize() {
        this.initCanvas();
        this.render();
    }
}

let demRenderer = null;

function initDEMRenderer() {
    if (!demRenderer) {
        demRenderer = new DEMRenderer('dem-2d-canvas');
        demRenderer.loadDEMGrid();
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const timelineSlider = document.getElementById('timeline-slider');
    const timelineYear = document.getElementById('timeline-year');
    
    if (timelineSlider) {
        timelineSlider.addEventListener('input', (e) => {
            const value = parseInt(e.target.value);
            const year = 2024 + Math.floor(value / 12);
            const month = value % 12 + 1;
            timelineYear.textContent = `${year}年${month}月`;
            
            if (demRenderer) {
                demRenderer.setYearOffset(value);
            }
        });
    }

    const runPredictionBtn = document.getElementById('run-prediction');
    if (runPredictionBtn) {
        runPredictionBtn.addEventListener('click', async () => {
            initDEMRenderer();
            
            const stationId = document.getElementById('evolution-station').value;
            const years = parseInt(document.getElementById('prediction-years').value);
            
            try {
                const data = await API.prediction.run(stationId, years);
                if (data.predictions) {
                    demRenderer.setPredictionData(data.predictions);
                    demRenderer.setYearOffset(0);
                    
                    document.getElementById('timeline-slider').max = years * 12;
                    document.getElementById('timeline-slider').value = 0;
                    document.getElementById('timeline-year').textContent = '2024年1月';
                    
                    updateEvolutionStats(data);
                    updateEvolutionCharts(data);
                }
            } catch (error) {
                console.error('预测失败:', error);
                generateMockPrediction(years);
            }
        });
    }
});

function generateMockPrediction(years) {
    const predictions = [];
    const totalMonths = years * 12;
    
    for (let i = 0; i < totalMonths; i++) {
        const yearFraction = i / 12;
        const seasonalVariation = Math.sin(yearFraction * Math.PI * 2) * 0.05;
        const depositionTrend = yearFraction * 0.08;
        const randomVariation = (Math.random() - 0.5) * 0.02;
        
        const elevationChange = depositionTrend + seasonalVariation + randomVariation;
        const erosionRate = Math.max(0, -seasonalVariation * 0.5);
        const depositionRate = Math.max(0, seasonalVariation * 0.5 + 0.006);
        
        predictions.push({
            prediction_date: new Date(2024 + Math.floor(i / 12), i % 12, 1).toISOString(),
            bed_elevation_change: elevationChange,
            erosion_rate: erosionRate,
            deposition_rate: depositionRate,
            sediment_accumulation: elevationChange * 1000
        });
    }
    
    demRenderer.setPredictionData(predictions);
    demRenderer.setYearOffset(0);
    
    const avgDeposition = predictions.reduce((sum, p) => sum + Math.max(0, p.deposition_rate), 0) / predictions.length;
    const avgErosion = predictions.reduce((sum, p) => sum + Math.max(0, p.erosion_rate), 0) / predictions.length;
    const finalElevation = 726.5 + predictions[predictions.length - 1].bed_elevation_change;
    const risk = predictions[predictions.length - 1].bed_elevation_change > 0.3 ? '高' : 
                 predictions[predictions.length - 1].bed_elevation_change > 0.15 ? '中' : '低';
    
    document.getElementById('avg-deposition').textContent = (avgDeposition * 100).toFixed(2) + ' cm';
    document.getElementById('avg-erosion').textContent = (avgErosion * 100).toFixed(2) + ' cm';
    document.getElementById('final-elevation').textContent = finalElevation.toFixed(2) + ' m';
    document.getElementById('risk-level').textContent = risk;
    document.getElementById('risk-level').className = 'stat-value risk ' + 
        (risk === '高' ? 'high' : risk === '中' ? 'medium' : 'low');
    
    const mockData = {
        predictions: predictions,
        average_annual_deposition: avgDeposition * 12,
        average_annual_erosion: avgErosion * 12,
        final_elevation: finalElevation,
        risk_level: risk
    };
    updateEvolutionCharts(mockData);
}

function updateEvolutionStats(data) {
    if (data.average_annual_deposition !== undefined) {
        document.getElementById('avg-deposition').textContent = 
            (data.average_annual_deposition * 100).toFixed(2) + ' cm';
    }
    if (data.average_annual_erosion !== undefined) {
        document.getElementById('avg-erosion').textContent = 
            (data.average_annual_erosion * 100).toFixed(2) + ' cm';
    }
    if (data.final_elevation !== undefined) {
        document.getElementById('final-elevation').textContent = 
            data.final_elevation.toFixed(2) + ' m';
    }
    if (data.risk_level !== undefined) {
        document.getElementById('risk-level').textContent = data.risk_level;
        document.getElementById('risk-level').className = 'stat-value risk ' + 
            (data.risk_level === '高' ? 'high' : 
             data.risk_level === '中' ? 'medium' : 'low');
    }
}

function updateEvolutionCharts(data) {
    if (!data.predictions) return;

    const labels = data.predictions.filter((_, i) => i % 12 === 0).map(p => {
        const date = new Date(p.prediction_date);
        return date.getFullYear() + '年';
    });

    const elevationData = data.predictions.filter((_, i) => i % 12 === 0).map(p => 
        726.5 + p.bed_elevation_change
    );

    const erosionData = data.predictions.filter((_, i) => i % 12 === 0).map(p => 
        p.erosion_rate * 100
    );

    const depositionData = data.predictions.filter((_, i) => i % 12 === 0).map(p => 
        p.deposition_rate * 100
    );

    const predictionCtx = document.getElementById('prediction-chart');
    if (predictionCtx && !window.predictionChart) {
        window.predictionChart = new Chart(predictionCtx, {
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
    } else if (window.predictionChart) {
        window.predictionChart.data.labels = labels;
        window.predictionChart.data.datasets[0].data = elevationData;
        window.predictionChart.update('none');
    }

    const erosionCtx = document.getElementById('erosion-chart');
    if (erosionCtx && !window.erosionChart) {
        window.erosionChart = new Chart(erosionCtx, {
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
    } else if (window.erosionChart) {
        window.erosionChart.data.labels = labels;
        window.erosionChart.data.datasets[0].data = depositionData;
        window.erosionChart.data.datasets[1].data = erosionData;
        window.erosionChart.update('none');
    }
}
