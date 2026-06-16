class BambooCageSimulation {
    constructor(canvasId) {
        this.canvas = document.getElementById(canvasId);
        this.ctx = this.canvas.getContext('2d');
        this.running = false;
        this.animationId = null;
        this.cages = [];
        this.stones = [];
        this.timeStep = 0;
        this.stabilityData = [];
        this.initCanvas();
    }

    initCanvas() {
        const rect = this.canvas.parentElement.getBoundingClientRect();
        this.canvas.width = rect.width;
        this.canvas.height = 400;
        this.width = this.canvas.width;
        this.height = this.canvas.height;
        this.groundY = this.height - 60;
    }

    async runSimulation(location, cageCount) {
        this.running = true;
        this.timeStep = 0;
        this.cages = [];
        this.stones = [];
        this.stabilityData = [];

        try {
            const response = await API.simulation.bambooCage(
                location, cageCount,
                '前端竹笼仿真_' + Date.now(), '前端用户'
            );
            this.simulationData = response;
            
            if (response.particles) {
                this.initializeFromDEMData(response.particles);
            } else {
                this.initializeCages(location, cageCount);
            }
            
            this.animate();
            this.updateResults(response);
        } catch (error) {
            console.error('竹笼仿真失败:', error);
            this.initializeCages(location, cageCount);
            this.animate();
        }
    }

    initializeFromDEMData(particles) {
        particles.forEach(p => {
            const screenX = this.width * 0.2 + (p.x + 50) * (this.width * 0.6 / 100);
            const screenY = this.groundY - (p.y - 724) * 20;
            
            if (p.is_cage) {
                this.cages.push({
                    x: screenX,
                    y: screenY,
                    width: 40,
                    height: 30,
                    stability: p.stability || 0.85,
                    filled: false,
                    fillProgress: 0,
                    stonesInside: []
                });
            } else {
                this.stones.push({
                    x: screenX,
                    y: 50 + Math.random() * 100,
                    vx: (Math.random() - 0.5) * 2,
                    vy: 0,
                    radius: p.radius * 15 || 4 + Math.random() * 6,
                    mass: p.mass || 1,
                    resting: false,
                    targetCage: Math.floor(Math.random() * this.cages.length),
                    color: this.getStoneColor()
                });
            }
        });
    }

    initializeCages(location, count) {
        const startX = this.width * 0.25;
        const spacing = 50;
        const rows = Math.ceil(count / 6);
        
        for (let i = 0; i < count; i++) {
            const row = Math.floor(i / 6);
            const col = i % 6;
            this.cages.push({
                x: startX + col * spacing,
                y: this.groundY - row * 35 - 15,
                width: 40,
                height: 30,
                stability: 0.7 + Math.random() * 0.3,
                filled: false,
                fillProgress: 0,
                stonesInside: []
            });
        }

        const totalStones = count * 15;
        for (let i = 0; i < totalStones; i++) {
            this.stones.push({
                x: Math.random() * this.width,
                y: -20 - Math.random() * 200,
                vx: (Math.random() - 0.5) * 1,
                vy: 0,
                radius: 4 + Math.random() * 6,
                mass: 0.5 + Math.random() * 1.5,
                resting: false,
                targetCage: Math.floor(Math.random() * count),
                color: this.getStoneColor()
            });
        }
    }

    getStoneColor() {
        const colors = ['#666666', '#777777', '#888888', '#999999', '#555555', '#7a7a7a'];
        return colors[Math.floor(Math.random() * colors.length)];
    }

    animate() {
        if (!this.running) return;

        this.ctx.clearRect(0, 0, this.width, this.height);
        this.timeStep++;

        this.drawBackground();
        this.drawGround();
        this.updateAndDrawCages();
        this.updateAndDrawStones();
        
        if (this.timeStep % 15 === 0) {
            this.updateStats();
        }

        this.animationId = requestAnimationFrame(() => this.animate());
    }

    drawBackground() {
        const gradient = this.ctx.createLinearGradient(0, 0, 0, this.height);
        gradient.addColorStop(0, '#1a3a2a');
        gradient.addColorStop(0.5, '#2d5a4a');
        gradient.addColorStop(1, '#3d6a5a');
        this.ctx.fillStyle = gradient;
        this.ctx.fillRect(0, 0, this.width, this.height);

        this.ctx.strokeStyle = 'rgba(107, 142, 35, 0.15)';
        this.ctx.lineWidth = 1;
        for (let i = 0; i < this.width; i += 40) {
            this.ctx.beginPath();
            this.ctx.moveTo(i, 0);
            this.ctx.lineTo(i, this.height);
            this.ctx.stroke();
        }
        for (let i = 0; i < this.height; i += 40) {
            this.ctx.beginPath();
            this.ctx.moveTo(0, i);
            this.ctx.lineTo(this.width, i);
            this.ctx.stroke();
        }
    }

    drawGround() {
        this.ctx.fillStyle = CONFIG.COLORS.terrainDark;
        this.ctx.fillRect(0, this.groundY, this.width, this.height - this.groundY);

        this.ctx.fillStyle = CONFIG.COLORS.terrain;
        for (let x = 0; x < this.width; x += 8) {
            const h = 3 + Math.sin(x * 0.1) * 3 + Math.random() * 2;
            this.ctx.fillRect(x, this.groundY - h, 6, h);
        }

        this.ctx.fillStyle = CONFIG.COLORS.sand;
        for (let i = 0; i < 50; i++) {
            const x = Math.random() * this.width;
            const y = this.groundY + Math.random() * 40;
            this.ctx.beginPath();
            this.ctx.arc(x, y, 1 + Math.random() * 2, 0, Math.PI * 2);
            this.ctx.fill();
        }
    }

    updateAndDrawCages() {
        this.cages.forEach((cage, index) => {
            if (cage.fillProgress < 1) {
                cage.fillProgress = Math.min(1, cage.fillProgress + 0.005);
            }
            if (cage.fillProgress >= 0.9 && !cage.filled) {
                cage.filled = true;
            }

            this.ctx.save();
            this.ctx.translate(cage.x, cage.y);

            this.ctx.strokeStyle = CONFIG.COLORS.bamboo;
            this.ctx.lineWidth = 2;
            this.ctx.fillStyle = `rgba(107, 142, 35, ${0.2 + cage.fillProgress * 0.3})`;
            
            this.ctx.beginPath();
            this.ctx.roundRect(-cage.width / 2, -cage.height / 2, cage.width, cage.height, 3);
            this.ctx.fill();
            this.ctx.stroke();

            this.ctx.beginPath();
            this.ctx.moveTo(-cage.width / 2, 0);
            this.ctx.lineTo(cage.width / 2, 0);
            this.ctx.stroke();

            this.ctx.beginPath();
            this.ctx.moveTo(0, -cage.height / 2);
            this.ctx.lineTo(0, cage.height / 2);
            this.ctx.stroke();

            const stabilityColor = cage.stability > 0.8 ? '#00ff88' : 
                                   cage.stability > 0.6 ? '#ffaa00' : '#ff4444';
            this.ctx.fillStyle = stabilityColor;
            this.ctx.font = '10px Arial';
            this.ctx.textAlign = 'center';
            this.ctx.fillText(Math.round(cage.stability * 100) + '%', 0, cage.height / 2 + 15);

            this.ctx.restore();
        });
    }

    updateAndDrawStones() {
        const gravity = 0.3;
        const friction = 0.98;
        const restitution = 0.3;

        this.stones.forEach(stone => {
            if (!stone.resting) {
                stone.vy += gravity * stone.mass;
                stone.x += stone.vx;
                stone.y += stone.vy;

                const targetCage = this.cages[stone.targetCage];
                if (targetCage && !stone.resting) {
                    if (stone.y > targetCage.y - targetCage.height / 2 &&
                        stone.y < targetCage.y + targetCage.height / 2 &&
                        stone.x > targetCage.x - targetCage.width / 2 &&
                        stone.x < targetCage.x + targetCage.width / 2) {
                        
                        if (targetCage.stonesInside.length < 20) {
                            stone.resting = true;
                            stone.vx = 0;
                            stone.vy = 0;
                            targetCage.stonesInside.push(stone);
                            targetCage.stability = Math.max(0.5, targetCage.stability - 0.01);
                        }
                    }
                }

                if (stone.y + stone.radius > this.groundY) {
                    stone.y = this.groundY - stone.radius;
                    stone.vy *= -restitution;
                    stone.vx *= friction;
                    
                    if (Math.abs(stone.vy) < 0.5 && Math.abs(stone.vx) < 0.1) {
                        stone.resting = true;
                        stone.vy = 0;
                    }
                }

                if (stone.x < stone.radius) {
                    stone.x = stone.radius;
                    stone.vx *= -restitution;
                }
                if (stone.x > this.width - stone.radius) {
                    stone.x = this.width - stone.radius;
                    stone.vx *= -restitution;
                }

                this.stones.forEach(other => {
                    if (stone !== other && !stone.resting) {
                        const dx = other.x - stone.x;
                        const dy = other.y - stone.y;
                        const dist = Math.sqrt(dx * dx + dy * dy);
                        const minDist = stone.radius + other.radius;

                        if (dist < minDist && dist > 0) {
                            this.resolveCollision(stone, other, dx, dy, dist, minDist);
                        }
                    }
                });
            }

            this.ctx.fillStyle = stone.color;
            this.ctx.beginPath();
            this.ctx.arc(stone.x, stone.y, stone.radius, 0, Math.PI * 2);
            this.ctx.fill();

            this.ctx.fillStyle = 'rgba(255, 255, 255, 0.2)';
            this.ctx.beginPath();
            this.ctx.arc(stone.x - stone.radius * 0.3, stone.y - stone.radius * 0.3, stone.radius * 0.4, 0, Math.PI * 2);
            this.ctx.fill();
        });
    }

    resolveCollision(stone1, stone2, dx, dy, dist, minDist) {
        const overlap = (minDist - dist) / 2;
        const nx = dx / dist;
        const ny = dy / dist;

        stone1.x -= overlap * nx;
        stone1.y -= overlap * ny;
        stone2.x += overlap * nx;
        stone2.y += overlap * ny;

        const dvx = stone1.vx - stone2.vx;
        const dvy = stone1.vy - stone2.vy;
        const dvDotN = dvx * nx + dvy * ny;

        if (dvDotN > 0) {
            const restitution = 0.3;
            const impulse = -(1 + restitution) * dvDotN / (1 / stone1.mass + 1 / stone2.mass);
            
            stone1.vx += impulse * nx / stone1.mass;
            stone1.vy += impulse * ny / stone1.mass;
            stone2.vx -= impulse * nx / stone2.mass;
            stone2.vy -= impulse * ny / stone2.mass;
        }
    }

    updateStats() {
        const filledCages = this.cages.filter(c => c.filled).length;
        const totalCages = this.cages.length;
        const avgStability = this.cages.reduce((sum, c) => sum + c.stability, 0) / totalCages;
        const totalStones = this.stones.filter(s => s.resting).length;
        const maxHeight = this.cages.reduce((max, c) => {
            if (c.stonesInside.length > 0) {
                const topStone = Math.min(...c.stonesInside.map(s => s.y));
                const height = (this.groundY - topStone) / 20;
                return Math.max(max, height);
            }
            return max;
        }, 0);

        document.getElementById('avg-stability').textContent = 
            Math.round(avgStability * 100) + ' %';
        document.getElementById('total-stones').textContent = totalStones + ' 块';
        document.getElementById('deposition-height').textContent = 
            maxHeight.toFixed(2) + ' m';

        this.stabilityData.push({
            time: this.timeStep,
            stability: avgStability * 100,
            filled: filledCages
        });

        if (this.stabilityData.length > 40) {
            this.stabilityData.shift();
        }

        this.updateChart();
    }

    updateChart() {
        if (!this.stabilityChart) {
            const ctx = document.getElementById('stability-chart');
            if (ctx) {
                this.stabilityChart = new Chart(ctx, {
                    type: 'bar',
                    data: {
                        labels: this.stabilityData.map(d => d.time),
                        datasets: [
                            {
                                label: '平均稳定性 (%)',
                                data: this.stabilityData.map(d => d.stability),
                                backgroundColor: 'rgba(0, 212, 255, 0.6)',
                                borderColor: '#00d4ff',
                                borderWidth: 1,
                                yAxisID: 'y'
                            },
                            {
                                label: '已填充竹笼数',
                                data: this.stabilityData.map(d => d.filled),
                                type: 'line',
                                borderColor: '#ffaa00',
                                backgroundColor: 'rgba(255, 170, 0, 0.1)',
                                tension: 0.4,
                                fill: true,
                                yAxisID: 'y1'
                            }
                        ]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false,
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
            this.stabilityChart.data.labels = this.stabilityData.map(d => d.time);
            this.stabilityChart.data.datasets[0].data = this.stabilityData.map(d => d.stability);
            this.stabilityChart.data.datasets[1].data = this.stabilityData.map(d => d.filled);
            this.stabilityChart.update('none');
        }
    }

    updateResults(data) {
        if (data.average_stability !== undefined) {
            document.getElementById('avg-stability').textContent = 
                Math.round(data.average_stability * 100) + ' %';
        }
        if (data.total_stones !== undefined) {
            document.getElementById('total-stones').textContent = data.total_stones + ' 块';
        }
        if (data.deposition_height !== undefined) {
            document.getElementById('deposition-height').textContent = 
                data.deposition_height.toFixed(2) + ' m';
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
        this.cages = [];
        this.stones = [];
        if (this.stabilityChart) {
            this.stabilityChart.destroy();
            this.stabilityChart = null;
        }
        this.ctx.clearRect(0, 0, this.width, this.height);
    }
}

let bambooSim = null;

function initBambooSimulation() {
    if (!bambooSim) {
        bambooSim = new BambooCageSimulation('bamboo-canvas');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const runBtn = document.getElementById('run-bamboo');
    if (runBtn) {
        runBtn.addEventListener('click', () => {
            initBambooSimulation();
            const location = document.getElementById('bamboo-location').value;
            const count = parseInt(document.getElementById('bamboo-count').value);
            bambooSim.reset();
            setTimeout(() => {
                bambooSim.runSimulation(location, count);
            }, 100);
        });
    }
});
