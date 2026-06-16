const Utils = {
    formatDateTime(date) {
        if (!(date instanceof Date)) {
            date = new Date(date);
        }
        const pad = n => n.toString().padStart(2, '0');
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
    },

    formatDate(date) {
        if (!(date instanceof Date)) {
            date = new Date(date);
        }
        const pad = n => n.toString().padStart(2, '0');
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
    },

    formatNumber(num, decimals = 2) {
        return Number(num).toFixed(decimals);
    },

    getElevationColor(elevation, minElev, maxElev) {
        const ratio = (elevation - minElev) / (maxElev - minElev);
        const clampedRatio = Math.max(0, Math.min(1, ratio));
        
        if (clampedRatio < 0.25) {
            const t = clampedRatio / 0.25;
            return Utils.interpolateColor([0, 136, 255], [0, 255, 136], t);
        } else if (clampedRatio < 0.5) {
            const t = (clampedRatio - 0.25) / 0.25;
            return Utils.interpolateColor([0, 255, 136], [255, 170, 0], t);
        } else if (clampedRatio < 0.75) {
            const t = (clampedRatio - 0.5) / 0.25;
            return Utils.interpolateColor([255, 170, 0], [255, 68, 68], t);
        } else {
            const t = (clampedRatio - 0.75) / 0.25;
            return Utils.interpolateColor([255, 68, 68], [200, 0, 0], t);
        }
    },

    interpolateColor(c1, c2, t) {
        const r = Math.round(c1[0] + (c2[0] - c1[0]) * t);
        const g = Math.round(c1[1] + (c2[1] - c1[1]) * t);
        const b = Math.round(c1[2] + (c2[2] - c1[2]) * t);
        return `rgb(${r}, ${g}, ${b})`;
    },

    rgbToArray(rgb) {
        const match = rgb.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
        if (match) {
            return [parseInt(match[1]), parseInt(match[2]), parseInt(match[3])];
        }
        return [0, 0, 0];
    },

    hslToRgb(h, s, l) {
        let r, g, b;
        if (s === 0) {
            r = g = b = l;
        } else {
            const hue2rgb = (p, q, t) => {
                if (t < 0) t += 1;
                if (t > 1) t -= 1;
                if (t < 1/6) return p + (q - p) * 6 * t;
                if (t < 1/2) return q;
                if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
                return p;
            };
            const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
            const p = 2 * l - q;
            r = hue2rgb(p, q, h + 1/3);
            g = hue2rgb(p, q, h);
            b = hue2rgb(p, q, h - 1/3);
        }
        return [Math.round(r * 255), Math.round(g * 255), Math.round(b * 255)];
    },

    throttle(func, wait) {
        let lastTime = 0;
        return function(...args) {
            const now = Date.now();
            if (now - lastTime >= wait) {
                lastTime = now;
                func.apply(this, args);
            }
        };
    },

    debounce(func, wait) {
        let timeout;
        return function(...args) {
            clearTimeout(timeout);
            timeout = setTimeout(() => func.apply(this, args), wait);
        };
    },

    noise(x, y, seed = 0) {
        const n = Math.sin(x * 12.9898 + y * 78.233 + seed) * 43758.5453;
        return n - Math.floor(n);
    },

    smoothNoise(x, y, seed) {
        const corners = (Utils.noise(x-1, y-1, seed) + Utils.noise(x+1, y-1, seed) +
                        Utils.noise(x-1, y+1, seed) + Utils.noise(x+1, y+1, seed)) / 16;
        const sides = (Utils.noise(x-1, y, seed) + Utils.noise(x+1, y, seed) +
                      Utils.noise(x, y-1, seed) + Utils.noise(x, y+1, seed)) / 8;
        const center = Utils.noise(x, y, seed) / 4;
        return corners + sides + center;
    },

    interpolatedNoise(x, y, seed) {
        const intX = Math.floor(x);
        const fracX = x - intX;
        const intY = Math.floor(y);
        const fracY = y - intY;

        const v1 = Utils.smoothNoise(intX, intY, seed);
        const v2 = Utils.smoothNoise(intX + 1, intY, seed);
        const v3 = Utils.smoothNoise(intX, intY + 1, seed);
        const v4 = Utils.smoothNoise(intX + 1, intY + 1, seed);

        const i1 = v1 * (1 - fracX) + v2 * fracX;
        const i2 = v3 * (1 - fracX) + v4 * fracX;

        return i1 * (1 - fracY) + i2 * fracY;
    },

    perlinNoise(x, y, octaves = 4, seed = 0) {
        let total = 0;
        let frequency = 1;
        let amplitude = 1;
        let maxValue = 0;

        for (let i = 0; i < octaves; i++) {
            total += Utils.interpolatedNoise(x * frequency, y * frequency, seed + i) * amplitude;
            maxValue += amplitude;
            amplitude *= 0.5;
            frequency *= 2;
        }

        return total / maxValue;
    },

    clamp(value, min, max) {
        return Math.max(min, Math.min(max, value));
    },

    lerp(a, b, t) {
        return a + (b - a) * t;
    },

    randomRange(min, max) {
        return Math.random() * (max - min) + min;
    },

    distance(x1, y1, x2, y2) {
        return Math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2);
    },

    distance3D(x1, y1, z1, x2, y2, z2) {
        return Math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2 + (z2 - z1) ** 2);
    },

    normalize(value, min, max) {
        return (value - min) / (max - min);
    },

    getStationColor(stationId) {
        const station = CONFIG.STATIONS.find(s => s.id === stationId);
        return station ? station.color : '#888888';
    },

    getStationName(stationId) {
        const station = CONFIG.STATIONS.find(s => s.id === stationId);
        return station ? station.name : stationId;
    }
};
