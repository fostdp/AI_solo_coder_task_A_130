class WaterParticleSystem {
    constructor(count = 2000) {
        this.count = count;
        this.particles = null;
        this.positions = null;
        this.velocities = null;
        this.colors = null;
        this.sizes = null;
        this.scale = 3;

        this.flowPaths = this.createFlowPaths();
    }

    createFlowPaths() {
        return [
            { start: new THREE.Vector3(-80, 0, -50), end: new THREE.Vector3(-30, 0, 30), width: 15, name: 'neijiang' },
            { start: new THREE.Vector3(-30, 0, 30), end: new THREE.Vector3(-10, 0, 60), width: 12, name: 'baopingkou' },
            { start: new THREE.Vector3(-80, 0, -50), end: new THREE.Vector3(40, 0, -30), width: 20, name: 'waijiang' },
            { start: new THREE.Vector3(-20, 0, 10), end: new THREE.Vector3(30, 0, -40), width: 10, name: 'feishayan' },
            { start: new THREE.Vector3(40, 0, -30), end: new THREE.Vector3(60, 0, -60), width: 18, name: 'waijiang_downstream' }
        ];
    }

    init(parent) {
        this.geometry = new THREE.BufferGeometry();
        this.positions = new Float32Array(this.count * 3);
        this.velocities = new Float32Array(this.count * 3);
        this.colors = new Float32Array(this.count * 3);
        this.sizes = new Float32Array(this.count);

        for (let i = 0; i < this.count; i++) {
            this.resetParticle(i);
        }

        this.geometry.setAttribute('position', new THREE.BufferAttribute(this.positions, 3));
        this.geometry.setAttribute('color', new THREE.BufferAttribute(this.colors, 3));
        this.geometry.setAttribute('size', new THREE.BufferAttribute(this.sizes, 1));

        const canvas = document.createElement('canvas');
        canvas.width = 64;
        canvas.height = 64;
        const ctx = canvas.getContext('2d');
        const gradient = ctx.createRadialGradient(32, 32, 0, 32, 32, 32);
        gradient.addColorStop(0, 'rgba(0, 212, 255, 1)');
        gradient.addColorStop(0.3, 'rgba(0, 212, 255, 0.6)');
        gradient.addColorStop(1, 'rgba(0, 212, 255, 0)');
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, 64, 64);
        const texture = new THREE.CanvasTexture(canvas);

        this.material = new THREE.PointsMaterial({
            size: 0.8,
            vertexColors: true,
            map: texture,
            transparent: true,
            opacity: 0.8,
            blending: THREE.AdditiveBlending,
            depthWrite: false,
            sizeAttenuation: true
        });

        this.particles = new THREE.Points(this.geometry, this.material);
        parent.add(this.particles);

        this.createWaterSurface(parent);
    }

    createWaterSurface(parent) {
        const surfaceGroup = new THREE.Group();

        this.flowPaths.forEach((path, index) => {
            const curve = new THREE.QuadraticBezierCurve3(
                path.start,
                new THREE.Vector3(
                    (path.start.x + path.end.x) / 2 + Utils.randomRange(-5, 5),
                    0,
                    (path.start.z + path.end.z) / 2 + Utils.randomRange(-5, 5)
                ),
                path.end
            );

            const points = curve.getPoints(50);
            const geometry = new THREE.TubeGeometry(curve, 50, path.width / 2, 8, false);
            
            const material = new THREE.MeshStandardMaterial({
                color: 0x00a8ff,
                transparent: true,
                opacity: 0.3,
                metalness: 0.1,
                roughness: 0.1,
                side: THREE.DoubleSide
            });

            const tube = new THREE.Mesh(geometry, material);
            tube.position.y = 728;
            tube.userData = { pathIndex: index };
            surfaceGroup.add(tube);
        });

        this.waterSurface = surfaceGroup;
        parent.add(surfaceGroup);
    }

    resetParticle(index) {
        const path = this.flowPaths[Math.floor(Math.random() * this.flowPaths.length)];
        const t = Math.random();

        const x = path.start.x + (path.end.x - path.start.x) * t + Utils.randomRange(-path.width / 2, path.width / 2);
        const z = path.start.z + (path.end.z - path.start.z) * t + Utils.randomRange(-path.width / 2, path.width / 2);
        const y = 728 + Utils.randomRange(0, 2) * this.scale;

        this.positions[index * 3] = x;
        this.positions[index * 3 + 1] = y;
        this.positions[index * 3 + 2] = z;

        const dir = new THREE.Vector3(
            path.end.x - path.start.x,
            0,
            path.end.z - path.start.z
        ).normalize();

        const speed = Utils.randomRange(0.5, 2.0);
        this.velocities[index * 3] = dir.x * speed;
        this.velocities[index * 3 + 1] = Utils.randomRange(-0.1, 0.3);
        this.velocities[index * 3 + 2] = dir.z * speed;

        const colorVariation = Math.random();
        const baseColor = new THREE.Color(0x00a8ff);
        const deepColor = new THREE.Color(0x0066aa);
        const finalColor = baseColor.lerp(deepColor, colorVariation * 0.5);
        
        this.colors[index * 3] = finalColor.r;
        this.colors[index * 3 + 1] = finalColor.g;
        this.colors[index * 3 + 2] = finalColor.b;

        this.sizes[index] = Utils.randomRange(0.3, 1.2);
    }

    update(delta) {
        if (!this.particles) return;

        for (let i = 0; i < this.count; i++) {
            const i3 = i * 3;

            this.positions[i3] += this.velocities[i3] * delta * 10;
            this.positions[i3 + 1] += this.velocities[i3 + 1] * delta * 5;
            this.positions[i3 + 2] += this.velocities[i3 + 2] * delta * 10;

            this.positions[i3 + 1] += Math.sin(Date.now() * 0.002 + i) * 0.01;

            if (this.positions[i3 + 1] > 732) {
                this.velocities[i3 + 1] = -Math.abs(this.velocities[i3 + 1]);
            }
            if (this.positions[i3 + 1] < 727.5) {
                this.velocities[i3 + 1] = Math.abs(this.velocities[i3 + 1]);
            }

            const x = this.positions[i3];
            const z = this.positions[i3 + 2];
            if (x > 80 || x < -80 || z > 80 || z < -80) {
                this.resetParticle(i);
            }
        }

        this.geometry.attributes.position.needsUpdate = true;
        this.geometry.attributes.color.needsUpdate = true;
        this.geometry.attributes.size.needsUpdate = true;

        if (this.waterSurface) {
            const time = Date.now() * 0.001;
            this.waterSurface.children.forEach((tube, index) => {
                tube.material.opacity = 0.25 + Math.sin(time + index) * 0.05;
            });
        }
    }

    setScale(scale) {
        this.scale = scale;
    }

    setCount(count) {
        this.count = count;
        if (this.particles) {
            this.particles.parent.remove(this.particles);
            this.init(this.particles.parent);
        }
    }
}
