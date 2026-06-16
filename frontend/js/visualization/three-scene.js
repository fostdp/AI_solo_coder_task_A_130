class DujiangyanScene {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            console.error(`Container ${containerId} not found`);
            return;
        }

        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        this.clock = new THREE.Clock();
        
        this.layers = {
            terrain: null,
            water: null,
            structures: null,
            wolongIron: null,
            stations: null
        };

        this.particleSystem = null;
        this.terrainMesh = null;
        this.structures = [];
        this.stationMarkers = [];
        this.wolongIronMarkers = [];

        this.waterScale = 3;
        this.particleCount = 2000;

        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.hoveredObject = null;

        this.init();
        this.animate();
        this.setupEventListeners();
    }

    init() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x0a1628);
        this.scene.fog = new THREE.Fog(0x0a1628, 150, 400);

        this.camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000);
        this.camera.position.set(100, 80, 100);

        this.renderer = new THREE.WebGLRenderer({ antialias: true });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(window.devicePixelRatio);
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        this.container.appendChild(this.renderer.domElement);

        this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement);
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.05;
        this.controls.maxPolarAngle = Math.PI / 2.1;
        this.controls.minDistance = 20;
        this.controls.maxDistance = 300;

        this.setupLighting();
        this.createLayers();
        this.createTerrain();
        this.createWater();
        this.createStructures();
        this.createWolongIronMarkers();
        this.createStationMarkers();
    }

    setupLighting() {
        const ambientLight = new THREE.AmbientLight(0x404050, 0.6);
        this.scene.add(ambientLight);

        const directionalLight = new THREE.DirectionalLight(0xffffff, 1);
        directionalLight.position.set(50, 100, 50);
        directionalLight.castShadow = true;
        directionalLight.shadow.mapSize.width = 2048;
        directionalLight.shadow.mapSize.height = 2048;
        directionalLight.shadow.camera.near = 0.5;
        directionalLight.shadow.camera.far = 500;
        directionalLight.shadow.camera.left = -150;
        directionalLight.shadow.camera.right = 150;
        directionalLight.shadow.camera.top = 150;
        directionalLight.shadow.camera.bottom = -150;
        this.scene.add(directionalLight);

        const fillLight = new THREE.DirectionalLight(0x88aaff, 0.3);
        fillLight.position.set(-50, 50, -50);
        this.scene.add(fillLight);

        const waterLight = new THREE.PointLight(0x00d4ff, 0.5, 200);
        waterLight.position.set(0, 5, 0);
        this.scene.add(waterLight);
    }

    createLayers() {
        this.layers.terrain = new THREE.Group();
        this.layers.water = new THREE.Group();
        this.layers.structures = new THREE.Group();
        this.layers.wolongIron = new THREE.Group();
        this.layers.stations = new THREE.Group();

        this.scene.add(this.layers.terrain);
        this.scene.add(this.layers.water);
        this.scene.add(this.layers.structures);
        this.scene.add(this.layers.wolongIron);
        this.scene.add(this.layers.stations);
    }

    createTerrain() {
        this.terrainMesh = TerrainGenerator.createDujiangyanTerrain();
        this.layers.terrain.add(this.terrainMesh);

        const gridHelper = new THREE.GridHelper(300, 60, 0x00d4ff, 0x00d4ff);
        gridHelper.material.opacity = 0.1;
        gridHelper.material.transparent = true;
        gridHelper.position.y = 724;
        this.layers.terrain.add(gridHelper);
    }

    createWater() {
        this.particleSystem = new WaterParticleSystem(this.particleCount);
        this.particleSystem.init(this.layers.water);
    }

    createStructures() {
        const structures = StructureGenerator.createAllStructures();
        structures.forEach(structure => {
            this.layers.structures.add(structure);
            this.structures.push(structure);
        });
    }

    createWolongIronMarkers() {
        const wolongPositions = [
            { x: -30, z: 10, elevation: 730.5, name: '内江河口卧铁' },
            { x: 30, z: 10, elevation: 730.2, name: '外江河口卧铁' },
            { x: 20, z: -20, elevation: 728.8, name: '飞沙堰卧铁' },
            { x: -10, z: 40, elevation: 729.3, name: '宝瓶口卧铁' }
        ];

        wolongPositions.forEach((pos, index) => {
            const geometry = new THREE.CylinderGeometry(0.5, 0.8, 2, 8);
            const material = new THREE.MeshStandardMaterial({
                color: 0xc0c0c0,
                metalness: 0.9,
                roughness: 0.3,
                emissive: 0x333333
            });
            const iron = new THREE.Mesh(geometry, material);
            iron.position.set(pos.x, pos.elevation, pos.z);
            iron.castShadow = true;
            iron.receiveShadow = true;
            iron.userData = { type: 'wolongIron', name: pos.name, elevation: pos.elevation, index };

            const glowGeometry = new THREE.CylinderGeometry(0.7, 1, 2.2, 8);
            const glowMaterial = new THREE.MeshBasicMaterial({
                color: 0x00d4ff,
                transparent: true,
                opacity: 0.2,
                side: THREE.BackSide
            });
            const glow = new THREE.Mesh(glowGeometry, glowMaterial);
            glow.position.copy(iron.position);
            this.layers.wolongIron.add(glow);

            const label = this.createLabel(pos.name, pos.x, pos.elevation + 3, pos.z);
            this.layers.wolongIron.add(label);

            this.layers.wolongIron.add(iron);
            this.wolongIronMarkers.push(iron);
        });
    }

    createStationMarkers() {
        const stationPositions = {
            'NEIJ-001': { x: -35, z: 15 },
            'NEIJ-002': { x: -25, z: 30 },
            'NEIJ-003': { x: -15, z: 45 },
            'WAIJ-001': { x: 35, z: 15 },
            'WAIJ-002': { x: 45, z: 0 },
            'FSSY-001': { x: 25, z: -15 },
            'FSSY-002': { x: 15, z: -25 },
            'RJK-001': { x: 40, z: -10 }
        };

        CONFIG.STATIONS.forEach(station => {
            const pos = stationPositions[station.id];
            if (!pos) return;

            const baseHeight = 728;

            const poleGeometry = new THREE.CylinderGeometry(0.2, 0.2, 8, 8);
            const poleMaterial = new THREE.MeshStandardMaterial({
                color: station.color,
                metalness: 0.5,
                roughness: 0.5,
                emissive: station.color,
                emissiveIntensity: 0.3
            });
            const pole = new THREE.Mesh(poleGeometry, poleMaterial);
            pole.position.set(pos.x, baseHeight + 4, pos.z);
            pole.castShadow = true;

            const sphereGeometry = new THREE.SphereGeometry(0.8, 16, 16);
            const sphereMaterial = new THREE.MeshStandardMaterial({
                color: station.color,
                emissive: station.color,
                emissiveIntensity: 0.5
            });
            const sphere = new THREE.Mesh(sphereGeometry, sphereMaterial);
            sphere.position.set(pos.x, baseHeight + 8.5, pos.z);
            sphere.userData = { type: 'station', id: station.id, name: station.name };

            const label = this.createLabel(station.name, pos.x, baseHeight + 10, pos.z, station.color);

            this.layers.stations.add(pole);
            this.layers.stations.add(sphere);
            this.layers.stations.add(label);
            this.stationMarkers.push({ pole, sphere, stationId: station.id });
        });
    }

    createLabel(text, x, y, z, color = '#00d4ff') {
        const canvas = document.createElement('canvas');
        const context = canvas.getContext('2d');
        canvas.width = 256;
        canvas.height = 64;

        context.fillStyle = 'rgba(10, 22, 40, 0.9)';
        context.fillRect(0, 0, canvas.width, canvas.height);
        
        context.font = 'bold 24px Microsoft YaHei';
        context.fillStyle = color;
        context.textAlign = 'center';
        context.textBaseline = 'middle';
        context.fillText(text, canvas.width / 2, canvas.height / 2);

        const texture = new THREE.CanvasTexture(canvas);
        const material = new THREE.SpriteMaterial({
            map: texture,
            transparent: true
        });
        const sprite = new THREE.Sprite(material);
        sprite.position.set(x, y, z);
        sprite.scale.set(12, 3, 1);

        return sprite;
    }

    updateStationData(stationId, data) {
        const marker = this.stationMarkers.find(m => m.stationId === stationId);
        if (!marker) return;

        if (data.bed_elevation) {
            marker.sphere.position.y = 728 + 0.5 + (data.bed_elevation - 726);
        }

        const pulseScale = 1 + Math.sin(Date.now() * 0.005) * 0.2;
        marker.sphere.scale.setScalar(pulseScale);
    }

    setView(viewName) {
        const view = CONFIG.VIEWS[viewName];
        if (!view) return;

        this.controls.target.set(view.target.x, view.target.y, view.target.z);
        this.camera.position.set(view.position.x, view.position.y, view.position.z);
        this.controls.update();
    }

    setLayerVisibility(layerName, visible) {
        if (this.layers[layerName]) {
            this.layers[layerName].visible = visible;
        }
    }

    setWaterScale(scale) {
        this.waterScale = scale;
        if (this.particleSystem) {
            this.particleSystem.setScale(scale);
        }
    }

    setParticleCount(count) {
        this.particleCount = count;
        if (this.particleSystem) {
            this.particleSystem.setCount(count);
        }
    }

    setupEventListeners() {
        window.addEventListener('resize', () => this.onResize());
        
        this.renderer.domElement.addEventListener('mousemove', (e) => this.onMouseMove(e));
        this.renderer.domElement.addEventListener('click', (e) => this.onClick(e));
    }

    onResize() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;

        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(width, height);
    }

    onMouseMove(event) {
        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        
        const intersects = this.raycaster.intersectObjects([
            ...this.wolongIronMarkers,
            ...this.stationMarkers.map(m => m.sphere)
        ]);

        if (intersects.length > 0) {
            this.hoveredObject = intersects[0].object;
            this.updateInfoPanel(this.hoveredObject.userData);
            document.body.style.cursor = 'pointer';
        } else {
            this.hoveredObject = null;
            this.clearInfoPanel();
            document.body.style.cursor = 'default';
        }
    }

    onClick(event) {
        if (this.hoveredObject) {
            console.log('Clicked:', this.hoveredObject.userData);
        }
    }

    updateInfoPanel(data) {
        const panel = document.getElementById('info-panel');
        if (!panel) return;

        if (data.type === 'station') {
            panel.innerHTML = `
                <h4>${data.name}</h4>
                <p>站点ID: ${data.id}</p>
                <p>类型: 监测站点</p>
            `;
        } else if (data.type === 'wolongIron') {
            panel.innerHTML = `
                <h4>${data.name}</h4>
                <p>卧铁高程: ${data.elevation.toFixed(3)} m</p>
                <p>作为岁修淘滩深度基准</p>
            `;
        }
    }

    clearInfoPanel() {
        const panel = document.getElementById('info-panel');
        if (panel) {
            panel.innerHTML = `
                <h4>信息面板</h4>
                <p>鼠标悬停查看详情</p>
            `;
        }
    }

    animate() {
        requestAnimationFrame(() => this.animate());

        const delta = this.clock.getDelta();

        if (this.particleSystem) {
            this.particleSystem.update(delta);
        }

        this.stationMarkers.forEach(marker => {
            const time = Date.now() * 0.001;
            marker.sphere.position.y = marker.sphere.position.y;
            marker.sphere.rotation.y += 0.01;
        });

        this.controls.update();
        this.renderer.render(this.scene, this.camera);
    }

    dispose() {
        if (this.renderer) {
            this.renderer.dispose();
            this.container.removeChild(this.renderer.domElement);
        }
    }
}

let mainScene = null;
let miniScene = null;

function initMainScene() {
    if (!mainScene && document.getElementById('three-container')) {
        mainScene = new DujiangyanScene('three-container');
    }
    return mainScene;
}

function initMiniScene() {
    if (!miniScene && document.getElementById('mini-3d-container')) {
        miniScene = new DujiangyanScene('mini-3d-container');
        if (miniScene.camera) {
            miniScene.camera.position.set(80, 60, 80);
            miniScene.controls.enabled = false;
        }
    }
    return miniScene;
}

function setView(viewName) {
    if (mainScene) {
        mainScene.setView(viewName);
    }
}
