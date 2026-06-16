class MachaSimulation {
    constructor(canvasId) {
        this.canvas = document.getElementById(canvasId);
        this.ctx = this.canvas.getContext('2d');
        this.running = false;
        this.animationId = null;
        this.machaStructures = [];
        this.waterParticles = [];
        this.timeStep = 0;
        this.simulationData = null;
        this.initCanvas();
    }

    initCanvas() {
        const rect = this.canvas.parentElement.getBoundingClientRect();
        this.canvas.width = rect.width;
        this.canvas.height = 400;
        this.width = this.canvas.width;
        this.height = this.canvas.height;
    }

    async runSimulation(location, machaCount, initialFlow, initialWaterLevel) {
        this.running = true;
        this.timeStep = 0;
        this.machaStructures = [];
        this.waterParticles = [];
        this.efficiencyData = [];
        this.flowData = [];
        this.waterLevelData = [];

        try {
            const response = await API.simulation.machaInterception(
                location, machaCount, initialFlow, initialWaterLevel,
                '前端仿真_' + Date.now(), '前端用户'
            );
            this.simulationData = response;
            
            if (response.time_series_data) {
                this.timeSeriesData = response.time_series_data;
            }
            
            this.initializeMachaStructures(location, machaCount);
            this.initializeWaterParticles(initialFlow);
            
            this.animate();
            
            if (response.interception_efficiency !== undefined) {
                this.updateResults(response);
            }
        } catch (error) {
            console.error('杩槎仿真失败:', error);
            this.initializeMachaStructures(location, machaCount);
            this.initializeWaterParticles(initialFlow);
            this.animate();
        }
    }

    initializeMachaStructures(location, count) {
        const startX = this.width * 0.4;
        const spacing = 30;
        const rows = Math.ceil(count / 5);
        
        for (let i = 0; i < count; i++) {
            const row = Math.floor(i / 5);
            const col = i % 5;
            this.machaStructures.push({
                x: startX + col * spacing - (spacing * 2),
                y: this.height - 80 - row * 25,
                width: 25,
                height: 60,
                angle: (Math.random() - 0.5) * 0.3,
                interceptionRate: 0.8 + Math.random() * 0.2,
                deployed: false,
                deployProgress: 0
            });
        }
    }

    initializeWaterParticles(flowRate) {
        const particleCount = Math.floor(flowRate / 2);
        for (let i = 0; i < particleCount; i++) {
            this.waterParticles.push({
                x: Math.random() * this.width * 0.3,
                y: this.height - 100 - Math.random() * 60,
                vx: 2 + Math.random() * 3,
                vy: (Math.random() - 0.5) * 0.5,
                size: 2 + Math.random() * 3,
                opacity: 0.6 + Math.random() * 0.4,
                intercepted: false
            });
        }
    }

    animate() {
        if (!this.running) return;

        this.ctx.clearRect(0, 0, this.width, this.height);
        this.timeStep++;

        this.drawBackground();
        this.drawRiverbed();
        this.updateAndDrawMacha();
        this.updateAndDrawWater();
        
        this.updateSimulationData();
        
        this.animationId = requestAnimationFrame(() => this.animate());
    }

    drawBackground() {
        const gradient = this.ctx.createLinearGradient(0, 0, 0, this.height);
        gradient.addColorStop(0, '#1a2a4a');
        gradient.addColorStop(0.6, '#2d4a6a');
        gradient.addColorStop(1, '#3d5a7a');
        this.ctx.fillStyle = gradient;
        this.ctx.fillRect(0, 0, this.width, this.height);

        this.ctx.strokeStyle = 'rgba(0, 212, 255, 0.1)';
        this.ctx.lineWidth = 1;
        for (let i = 0; i < this.width; i += 50) {
            this.ctx.beginPath();
            this.ctx.moveTo(i, 0);
            this.ctx.lineTo(i, this.height);
            this.ctx.stroke();
        }
        for (let i = 0; i < this.height; i += 50) {
            this.ctx.beginPath();
            this.ctx.moveTo(0, i);
            this.ctx.lineTo(this.width, i);
            this.ctx.stroke();
        }
    }

    drawRiverbed() {
        this.ctx.fillStyle = CONFIG.COLORS.terrain;
        this.ctx.beginPath();
        this.ctx.moveTo(0, this.height - 50);
        
        for (let x = 0; x <= this.width; x += 20) {
            const y = this.height - 50 - Math.sin(x * 0.02) * 10 - Math.sin(x * 0.05) * 5;
            this.ctx.lineTo(x, y);
        }
        
        this.ctx.lineTo(this.width, this.height);
        this.ctx.lineTo(0, this.height);
        this.ctx.closePath();
        this.ctx.fill();

        this.ctx.fillStyle = CONFIG.COLORS.sand;
        for (let x = 0; x < this.width; x += 15) {
            for (let y = this.height - 45; y < this.height; y += 10) {
                if (Math.random() > 0.7) {
                    this.ctx.beginPath();
                    this.ctx.arc(x + Math.random() * 10, y + Math.random() * 5, 1 + Math.random() * 2, 0, Math.PI * 2);
                    this.ctx.fill();
                }
            }
        }
    }

    updateAndDrawMacha() {
        this.machaStructures.forEach((macha, index) => {
            if (!macha.deployed && index < Math.floor(this.timeStep / 5)) {
                macha.deployProgress = Math.min(1, macha.deployProgress + 0.05);
                if (macha.deployProgress >= 1) {
                    macha.deployed = true;
                }
            }

            const currentY = macha.y + (1 - macha.deployProgress) * 100;
            const opacity = 0.3 + macha.deployProgress * 0.7;

            this.ctx.save();
            this.ctx.translate(macha.x, currentY);
            this.ctx.rotate(macha.angle);
            this.ctx.globalAlpha = opacity;

            this.ctx.strokeStyle = CONFIG.COLORS.wood;
            this.ctx.lineWidth = 3;
            this.ctx.beginPath();
            this.ctx.moveTo(0, 0);
            this.ctx.lineTo(-macha.width / 2, macha.height);
            this.ctx.stroke();
            this.ctx.beginPath();
            this.ctx.moveTo(0, 0);
            this.ctx.lineTo(macha.width / 2, macha.height);
            this.ctx.stroke();
            this.ctx.beginPath();
            this.ctx.moveTo(-macha.width / 3, macha.height * 0.6);
            this.ctx.lineTo(macha.width / 3, macha.height * 0.6);
            this.ctx.stroke();

            this.ctx.fillStyle = CONFIG.COLORS.bamboo;
            this.ctx.fillRect(-macha.width / 2 - 5, macha.height - 10, macha.width + 10, 15);

            this.ctx.fillStyle = '#8b4513';
            this.ctx.beginPath();
            this.ctx.arc(-macha.width / 3, macha.height * 0.6, 4, 0, Math.PI * 2);
            this.ctx.fill();
            this.ctx.beginPath();
            this.ctx.arc(macha.width / 3, macha.height * 0.6, 4, 0, Math.PI * 2);
            this.ctx.fill();

            this.ctx.restore();
        });
    }

    updateAndDrawWater() {
        let interceptedCount = 0;
        const totalCount = this.waterParticles.length;

        this.waterParticles.forEach(particle => {
            if (!particle.intercepted) {
                particle.x += particle.vx;
                particle.y += particle.vy;
                particle.y += Math.sin(this.timeStep * 0.1 + particle.x * 0.05) * 0.3;

                this.machaStructures.forEach(macha => {
                    if (macha.deployed && this.checkCollision(particle, macha)) {
                        if (Math.random() < macha.interceptionRate * 0.3) {
                            particle.intercepted = true;
                            interceptedCount++;
                        } else {
                            particle.vx *= 0.5;
                            particle.vy += (Math.random() - 0.5) * 2;
                        }
                    }
                });

                if (particle.x > this.width) {
                    particle.x = -10;
                    particle.y = this.height - 100 - Math.random() * 60;
                    particle.vx = 2 + Math.random() * 3;
                }
            } else {
                particle.vy += 0.1;
                particle.y += particle.vy;
                if (particle.y > this.height - 50) {
                    particle.y = this.height - 50;
                    particle.vy = 0;
                }
            }

            this.ctx.fillStyle = particle.intercepted 
                ? `rgba(0, 168, 255, ${particle.opacity * 0.3})`
                : `rgba(0, 212, 255, ${particle.opacity})`;
            this.ctx.beginPath();
            this.ctx.arc(particle.x, particle.y, particle.size, 0, Math.PI * 2);
            this.ctx.fill();
        });

        const interceptionRate = interceptedCount / totalCount;
        document.getElementById('final-efficiency').textContent = 
            Math.round(interceptionRate * 100) + ' %';
    }

    checkCollision(particle, macha) {
        return particle.x > macha.x - macha.width &&
               particle.x < macha.x + macha.width &&
               particle.y > macha.y &&
               particle.y < macha.y + macha.height;
    }

    updateSimulationData() {
        if (this.timeStep % 10 === 0) {
            const deployedCount = this.machaStructures.filter(m => m.deployed).length;
            const totalCount = this.machaStructures.length;
            const progress = deployedCount / totalCount;
            
            const efficiency = progress * (0.7 + Math.random() * 0.2);
            const remainingFlow = 300 * (1 - efficiency);
            const waterRise = progress * 1.5;

            this.efficiencyData.push({ x: this.timeStep, y: efficiency * 100 });
            this.flowData.push({ x: this.timeStep, y: remainingFlow });
            this.waterLevelData.push({ x: this.timeStep, y: waterRise });

            if (this.efficiencyData.length > 50) {
                this.efficiencyData.shift();
                this.flowData.shift();
                this.waterLevelData.shift();
            }

            this.updateCharts();
            
            if (document.getElementById('final-flow')) {
                document.getElementById('final-flow').textContent = 
                    remainingFlow.toFixed(1) + ' m³/s';
            }
            if (document.getElementById('water-rise')) {
                document.getElementById('water-rise').textContent = 
                    waterRise.toFixed(2) + ' m';
            }
        }
    }

    updateCharts() {
        if (!this.interceptionChart) {
            const ctx = document.getElementById('interception-chart');
            if (ctx) {
                this.interceptionChart = new Chart(ctx, {
                    type: 'line',
                    data: {
                        labels: this.efficiencyData.map(d => d.x),
                        datasets: [
                            {
                                label: '截流效率 (%)',
                                data: this.efficiencyData.map(d => d.y),
                                borderColor: '#00d4ff',
                                backgroundColor: 'rgba(0, 212, 255, 0.1)',
                                tension: 0.4,
                                fill: true,
                                yAxisID: 'y'
                            },
                            {
                                label: '剩余流量 (m³/s)',
                                data: this.flowData.map(d => d.y),
                                borderColor: '#ff6600',
                                backgroundColor: 'rgba(255, 102, 0, 0.1)',
                                tension: 0.4,
                                fill: true,
                                yAxisID: 'y1'
                            }
                        ]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false,
                        interaction: {
                            mode: 'index',
                            intersect: false
                        },
                        scales: {
                            y: {
                                type: 'linear',
                                display: true,
                                position: 'left',
                                min: 0,
                                max: 100,
                                grid: { color: 'rgba(255,255,255,0.1)' },
                                ticks: { color: '#aaa' }
                            },
                            y1: {
                                type: 'linear',
                                display: true,
                                position: 'right',
                                min: 0,
                                max: 350,
                                grid: { display: false },
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
        } else {
            this.interceptionChart.data.labels = this.efficiencyData.map(d => d.x);
            this.interceptionChart.data.datasets[0].data = this.efficiencyData.map(d => d.y);
            this.interceptionChart.data.datasets[1].data = this.flowData.map(d => d.y);
            this.interceptionChart.update('none');
        }
    }

    updateResults(data) {
        if (data.interception_efficiency !== undefined) {
            document.getElementById('final-efficiency').textContent = 
                Math.round(data.interception_efficiency * 100) + ' %';
        }
        if (data.final_flow_rate !== undefined) {
            document.getElementById('final-flow').textContent = 
                data.final_flow_rate.toFixed(1) + ' m³/s';
        }
        if (data.water_level_rise !== undefined) {
            document.getElementById('water-rise').textContent = 
                data.water_level_rise.toFixed(2) + ' m';
        }
    }

    stop() {
        this.running = false;
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
        }
    }

    reset() {
        this.stop();
        this.timeStep = 0;
        this.machaStructures = [];
        this.waterParticles = [];
        if (this.interceptionChart) {
            this.interceptionChart.destroy();
            this.interceptionChart = null;
        }
        this.ctx.clearRect(0, 0, this.width, this.height);
    }
}

let machaSim = null;

function initMachaSimulation() {
    if (!machaSim) {
        machaSim = new MachaSimulation('macha-canvas');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const runBtn = document.getElementById('run-macha');
    if (runBtn) {
        runBtn.addEventListener('click', () => {
            initMachaSimulation();
            const location = document.getElementById('macha-location').value;
            const count = parseInt(document.getElementById('macha-count').value);
            const flow = parseFloat(document.getElementById('initial-flow').value);
            const waterLevel = parseFloat(document.getElementById('initial-water-level').value);
            machaSim.reset();
            setTimeout(() => {
                machaSim.runSimulation(location, count, flow, waterLevel);
            }, 100);
        });
    }
});
