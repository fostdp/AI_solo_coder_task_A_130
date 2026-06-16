const TerrainGenerator = {
    createDujiangyanTerrain() {
        const size = 300;
        const segments = 100;
        const baseElevation = 726;

        const geometry = new THREE.PlaneGeometry(size, size, segments, segments);
        geometry.rotateX(-Math.PI / 2);

        const positions = geometry.attributes.position.array;
        const colors = new Float32Array(positions.length);

        for (let i = 0; i < positions.length; i += 3) {
            const x = positions[i];
            const z = positions[i + 2];

            let elevation = this.calculateElevation(x, z, baseElevation);
            positions[i + 1] = elevation;

            const color = this.getTerrainColor(elevation, baseElevation);
            colors[i] = color.r / 255;
            colors[i + 1] = color.g / 255;
            colors[i + 2] = color.b / 255;
        }

        geometry.computeVertexNormals();
        geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));

        const material = new THREE.MeshStandardMaterial({
            vertexColors: true,
            roughness: 0.9,
            metalness: 0.1,
            flatShading: false
        });

        const terrain = new THREE.Mesh(geometry, material);
        terrain.receiveShadow = true;
        terrain.castShadow = true;
        terrain.position.y = 0;

        return terrain;
    },

    calculateElevation(x, z, baseElev) {
        let elevation = baseElev;

        const noise1 = Utils.perlinNoise(x * 0.01, z * 0.01, 4, 1) * 3;
        const noise2 = Utils.perlinNoise(x * 0.03 + 100, z * 0.03 + 100, 3, 2) * 1.5;
        elevation += noise1 + noise2;

        const neijiangDist = this.distanceToLine(x, z, -80, -50, -10, 60);
        const waijiangDist = this.distanceToLine(x, z, -80, -50, 60, -60);
        const feishayanDist = this.distanceToLine(x, z, -20, 10, 30, -40);

        if (neijiangDist < 15) {
            const depth = (1 - neijiangDist / 15) * 3;
            elevation -= depth;
        }

        if (waijiangDist < 20) {
            const depth = (1 - waijiangDist / 20) * 3.5;
            elevation -= depth;
        }

        if (feishayanDist < 10) {
            const depth = (1 - feishayanDist / 10) * 2;
            elevation -= depth;
        }

        if (x > -15 && x < 15 && z > 50 && z < 70) {
            const bottleDist = Math.sqrt((x) ** 2 + (z - 60) ** 2);
            if (bottleDist < 15) {
                elevation -= (1 - bottleDist / 15) * 2;
            }
            if (bottleDist < 12) {
                elevation += 1.5;
            }
        }

        if (x > -40 && x < 40 && z > -5 && z < 5) {
            const weirDist = Math.abs(z);
            if (weirDist < 5) {
                elevation += (1 - weirDist / 5) * 1.5;
            }
        }

        if (x > 20 && x < 50 && z > -45 && z < -15) {
            const spillwayDist = this.distanceToLine(x, z, 20, -15, 50, -45);
            if (spillwayDist < 8) {
                elevation -= (1 - spillwayDist / 8) * 1.5;
            }
        }

        return elevation;
    },

    distanceToLine(px, pz, x1, z1, x2, z2) {
        const A = px - x1;
        const B = pz - z1;
        const C = x2 - x1;
        const D = z2 - z1;

        const dot = A * C + B * D;
        const lenSq = C * C + D * D;
        let param = -1;

        if (lenSq !== 0) param = dot / lenSq;

        let xx, zz;

        if (param < 0) {
            xx = x1;
            zz = z1;
        } else if (param > 1) {
            xx = x2;
            zz = z2;
        } else {
            xx = x1 + param * C;
            zz = z1 + param * D;
        }

        const dx = px - xx;
        const dz = pz - zz;
        return Math.sqrt(dx * dx + dz * dz);
    },

    getTerrainColor(elevation, baseElev) {
        const relElev = elevation - baseElev;

        if (relElev < -2) {
            return { r: 0, g: 100, b: 150 };
        } else if (relElev < -1) {
            const t = (relElev + 2);
            return this.lerpColor({ r: 0, g: 100, b: 150 }, { r: 139, g: 115, b: 85 }, t);
        } else if (relElev < 0.5) {
            const t = (relElev + 1) / 1.5;
            return this.lerpColor({ r: 139, g: 115, b: 85 }, { r: 210, g: 180, b: 140 }, t);
        } else if (relElev < 2) {
            const t = (relElev - 0.5) / 1.5;
            return this.lerpColor({ r: 210, g: 180, b: 140 }, { r: 100, g: 120, b: 80 }, t);
        } else {
            const t = Math.min((relElev - 2) / 2, 1);
            return this.lerpColor({ r: 100, g: 120, b: 80 }, { r: 80, g: 90, b: 70 }, t);
        }
    },

    lerpColor(c1, c2, t) {
        return {
            r: Math.round(c1.r + (c2.r - c1.r) * t),
            g: Math.round(c1.g + (c2.g - c1.g) * t),
            b: Math.round(c1.b + (c2.b - c1.b) * t)
        };
    },

    createHeightMap(gridSize, resolution, baseElevation) {
        const gridCells = Math.floor(gridSize / resolution);
        const heightMap = [];

        for (let i = 0; i < gridCells; i++) {
            heightMap[i] = [];
            for (let j = 0; j < gridCells; j++) {
                const x = (i - gridCells / 2) * resolution;
                const z = (j - gridCells / 2) * resolution;

                const elevation = this.calculateElevation(x, z, baseElevation);
                heightMap[i][j] = elevation;
            }
        }

        return heightMap;
    }
};
