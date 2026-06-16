class Dujiangyan3D {
    constructor(containerId) {
        this.containerId = containerId;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        this.raycaster = null;
        this.mouse = null;
        this.layers = {};
        this.animationId = null;
        this.waterSystem = null;
        this.clock = null;
        this.isInitialized = false;
        this.viewPresets = {
            overview: { pos: [0, 150, 150], target: [0, 0, 0] },
            yuzui: { pos: [-60, 50, 30], target: [-40, 0, 0] },
            feishayan: { pos: [20, 40, 50], target: [10, 0, 20] },
            baopingkou: { pos: [60, 30, 20], target: [40, 0, 0] },
            top: { pos: [0, 200, 0], target: [0, 0, 0] }
        };
        this.layerVisibility = {
            terrain: true,
            water: true,
            structures: true,
            wolongIron: true,
            stations: true
        };
    }

    init() {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x0a1628);
        this.scene.fog = new THREE.FogExp2(0x0a1628, 0.003);

        const rect = container.getBoundingClientRect();
        const aspect = rect.width / rect.height;
        this.camera = new THREE.PerspectiveCamera(60, aspect, 0.1, 2000);
        this.camera.position.set(0, 150, 150);

        this.renderer = new THREE.WebGLRenderer({ antialias: true });
        this.renderer.setSize(rect.width, rect.height);
        this.renderer.setPixelRatio(window.devicePixelRatio);
        this.renderer.shadowMap.enabled = true;
        container.appendChild(this.renderer.domElement);

        this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement);
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.05;
        this.controls.maxPolarAngle = Math.PI / 2.1;

        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.clock = new THREE.Clock();

        this.createLights();
        this.createLayers();
        this.createTerrain();
        this.createWater();
        this.createStructures();
        this.createWolongIronMarkers();
        this.createStationMarkers();

        this.isInitialized = true;
        this.animate();
    }

    createLights() {
        const ambient = new THREE.AmbientLight(0x4466aa, 0.6);
        this.scene.add(ambient);

        const directional = new THREE.DirectionalLight(0xffeedd, 0.8);
        directional.position.set(50, 100, 50);
        directional.castShadow = true;
        directional.shadow.mapSize.width = 2048;
        directional.shadow.mapSize.height = 2048;
        this.scene.add(directional);

        const hemisphere = new THREE.HemisphereLight(0x87ceeb, 0x362d1b, 0.3);
        this.scene.add(hemisphere);
    }

    createLayers() {
        this.layers = {
            terrain: new THREE.Group(),
            water: new THREE.Group(),
            structures: new THREE.Group(),
            wolongIron: new THREE.Group(),
            stations: new THREE.Group()
        };
        Object.values(this.layers).forEach(layer => this.scene.add(layer));
    }

    createTerrain() {
        if (typeof TerrainGenerator === 'undefined') return;
        const terrain = TerrainGenerator.createDujiangyanTerrain(this.scene);
        this.layers.terrain.add(terrain);

        const gridHelper = new THREE.GridHelper(200, 20, 0x003366, 0x002244);
        gridHelper.position.y = 723;
        this.layers.terrain.add(gridHelper);
    }

    createWater() {
        if (typeof WaterParticleSystem === 'undefined') return;
        this.waterSystem = new WaterParticleSystem(2000);
        const particles = this.waterSystem.init(this.scene);
        this.layers.water.add(particles);

        const surface = this.waterSystem.createWaterSurface();
        this.layers.water.add(surface);
    }

    createStructures() {
        if (typeof StructureGenerator === 'undefined') return;
        const gen = StructureGenerator;

        const yuzui = gen.createYuzui();
        this.layers.structures.add(yuzui);

        const feishayan = gen.createFeishayan();
        this.layers.structures.add(feishayan);

        const baopingkou = gen.createBaopingkou();
        this.layers.structures.add(baopingkou);

        const renzidi = gen.createRenzidi();
        this.layers.structures.add(renzidi);

        const buildings = gen.createAncientBuildings();
        this.layers.structures.add(buildings);
    }

    createWolongIronMarkers() {
        const CONFIG = window.CONFIG || {};
        const ironPositions = [
            { name: '卧铁1', x: -30, z: 15, elev: 726.24 },
            { name: '卧铁2', x: -20, z: 20, elev: 726.18 },
            { name: '卧铁3', x: -10, z: 25, elev: 726.12 },
            { name: '卧铁4', x: 0, z: 30, elev: 726.06 }
        ];

        ironPositions.forEach(iron => {
            const group = new THREE.Group();

            const cylinderGeom = new THREE.CylinderGeometry(0.8, 0.8, 1.2, 16);
            const cylinderMat = new THREE.MeshStandardMaterial({
                color: 0xc0c0c0,
                metalness: 0.9,
                roughness: 0.2,
                emissive: 0x333333
            });
            const cylinder = new THREE.Mesh(cylinderGeom, cylinderMat);
            cylinder.position.y = 0.6;
            group.add(cylinder);

            const glowGeom = new THREE.SphereGeometry(1.5, 16, 16);
            const glowMat = new THREE.MeshBasicMaterial({
                color: 0xffaa00,
                transparent: true,
                opacity: 0.3
            });
            const glow = new THREE.Mesh(glowGeom, glowMat);
            glow.position.y = 0.6;
            group.add(glow);

            group.position.set(iron.x, iron.elev - 726, iron.z);
            group.userData = { type: 'wolongIron', name: iron.name, elevation: iron.elev };
            this.layers.wolongIron.add(group);
        });
    }

    createStationMarkers() {
        const stations = (window.CONFIG && window.CONFIG.STATIONS) || [
            { id: 'NEIJ-001', name: '内江1号', x: -15, z: 10 },
            { id: 'NEIJ-002', name: '内江2号', x: -5, z: 15 },
            { id: 'NEIJ-003', name: '内江3号', x: 5, z: 20 },
            { id: 'WAIJ-001', name: '外江1号', x: -15, z: -10 },
            { id: 'WAIJ-002', name: '外江2号', x: -5, z: -15 },
            { id: 'FSSY-001', name: '飞沙堰1号', x: 10, z: 5 },
            { id: 'FSSY-002', name: '飞沙堰2号', x: 15, z: 10 },
            { id: 'RJK-001', name: '人字堤1号', x: 25, z: 5 }
        ];

        stations.forEach(station => {
            const group = new THREE.Group();

            const poleGeom = new THREE.CylinderGeometry(0.15, 0.15, 8, 8);
            const poleMat = new THREE.MeshStandardMaterial({ color: 0x00ff88 });
            const pole = new THREE.Mesh(poleGeom, poleMat);
            pole.position.y = 4;
            group.add(pole);

            const sphereGeom = new THREE.SphereGeometry(0.6, 16, 16);
            const sphereMat = new THREE.MeshStandardMaterial({
                color: 0x00d4ff,
                emissive: 0x004488
            });
            const sphere = new THREE.Mesh(sphereGeom, sphereMat);
            sphere.position.y = 8.5;
            group.add(sphere);

            group.position.set(station.x, 0, station.z);
            group.userData = { type: 'station', id: station.id, name: station.name };
            this.layers.stations.add(group);
        });
    }

    setLayerVisibility(layerName, visible) {
        if (this.layers[layerName]) {
            this.layers[layerName].visible = visible;
            this.layerVisibility[layerName] = visible;
        }
    }

    setViewPreset(presetName) {
        const preset = this.viewPresets[presetName];
        if (!preset || !this.camera || !this.controls) return;

        const duration = 1000;
        const startPos = this.camera.position.clone();
        const startTarget = this.controls.target.clone();
        const endPos = new THREE.Vector3(...preset.pos);
        const endTarget = new THREE.Vector3(...preset.target);
        const startTime = performance.now();

        const animateView = (time) => {
            const elapsed = time - startTime;
            const t = Math.min(elapsed / duration, 1);
            const eased = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;

            this.camera.position.lerpVectors(startPos, endPos, eased);
            this.controls.target.lerpVectors(startTarget, endTarget, eased);
            this.controls.update();

            if (t < 1) {
                requestAnimationFrame(animateView);
            }
        };
        requestAnimationFrame(animateView);
    }

    updateStationData(stationId, data) {
        this.layers.stations.children.forEach(group => {
            if (group.userData.type === 'station' && group.userData.id === stationId) {
                const sphere = group.children.find(c => c.geometry && c.geometry.type === 'SphereGeometry');
                if (sphere && data.bed_elevation > 726.5) {
                    sphere.material.emissive.setHex(0xff4400);
                } else if (sphere) {
                    sphere.material.emissive.setHex(0x004488);
                }
            }
        });
    }

    handleClick(event) {
        if (!this.raycaster || !this.camera) return;

        const container = document.getElementById(this.containerId);
        const rect = container.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);

        const allObjects = [
            ...this.layers.wolongIron.children,
            ...this.layers.stations.children
        ];

        const intersects = this.raycaster.intersectObjects(allObjects, true);
        if (intersects.length > 0) {
            let obj = intersects[0].object;
            while (obj.parent && !obj.userData.type) {
                obj = obj.parent;
            }
            if (obj.userData.type) {
                this.updateInfoPanel(obj.userData);
            }
        }
    }

    updateInfoPanel(data) {
        const panel = document.getElementById('info-panel');
        if (!panel) return;

        if (data.type === 'wolongIron') {
            panel.innerHTML = `
                <h4>${data.name}</h4>
                <p>高程: ${data.elevation.toFixed(2)}m</p>
                <p>类型: 卧铁标记</p>
            `;
        } else if (data.type === 'station') {
            panel.innerHTML = `
                <h4>${data.name}</h4>
                <p>站点编号: ${data.id}</p>
                <p>类型: 水文监测站</p>
            `;
        }
        panel.style.display = 'block';
    }

    animate() {
        this.animationId = requestAnimationFrame(() => this.animate());

        if (this.waterSystem) {
            const delta = this.clock.getDelta();
            this.waterSystem.update(delta);
        }

        if (this.controls) {
            this.controls.update();
        }

        if (this.renderer && this.scene && this.camera) {
            this.renderer.render(this.scene, this.camera);
        }
    }

    resize() {
        const container = document.getElementById(this.containerId);
        if (!container || !this.camera || !this.renderer) return;

        const rect = container.getBoundingClientRect();
        this.camera.aspect = rect.width / rect.height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(rect.width, rect.height);
    }

    destroy() {
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
        }
        if (this.renderer) {
            this.renderer.dispose();
        }
        if (this.controls) {
            this.controls.dispose();
        }
        this.isInitialized = false;
    }
}
