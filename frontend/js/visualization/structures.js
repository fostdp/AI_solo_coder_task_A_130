const StructureGenerator = {
    createAllStructures() {
        const structures = [];

        structures.push(...this.createYuzui());
        structures.push(...this.createFeishayan());
        structures.push(...this.createBaopingkou());
        structures.push(...this.createRenzhidi());
        structures.push(...this.createAncientBuildings());

        return structures;
    },

    createYuzui() {
        const group = new THREE.Group();

        const bodyGeometry = new THREE.BoxGeometry(70, 4, 25);
        const bodyMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b7355,
            roughness: 0.9,
            metalness: 0.1
        });
        const body = new THREE.Mesh(bodyGeometry, bodyMaterial);
        body.position.set(-45, 727, -20);
        body.castShadow = true;
        body.receiveShadow = true;
        body.userData = { type: 'yuzui', name: '鱼嘴' };
        group.add(body);

        const tipGeometry = new THREE.ConeGeometry(15, 20, 4);
        const tip = new THREE.Mesh(tipGeometry, bodyMaterial);
        tip.position.set(-85, 727, -20);
        tip.rotation.y = Math.PI / 4;
        tip.castShadow = true;
        tip.receiveShadow = true;
        group.add(tip);

        const tailGeometry = new THREE.ConeGeometry(12, 30, 4);
        const tail = new THREE.Mesh(tailGeometry, bodyMaterial);
        tail.position.set(-10, 727, -20);
        tail.rotation.y = -Math.PI / 4 + Math.PI;
        tail.castShadow = true;
        tail.receiveShadow = true;
        group.add(tail);

        for (let i = 0; i < 20; i++) {
            const cage = this.createBambooCageMesh();
            cage.position.set(
                -80 + i * 3.5,
                726.5,
                -20 + Utils.randomRange(-8, 8)
            );
            cage.rotation.y = Utils.randomRange(-0.3, 0.3);
            group.add(cage);
        }

        group.position.y = 0;
        return [group];
    },

    createFeishayan() {
        const group = new THREE.Group();

        const weirGeometry = new THREE.BoxGeometry(50, 2, 15);
        const weirMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b7355,
            roughness: 0.9,
            metalness: 0.1
        });
        const weir = new THREE.Mesh(weirGeometry, weirMaterial);
        weir.position.set(5, 727.5, -25);
        weir.rotation.y = Math.PI / 6;
        weir.castShadow = true;
        weir.receiveShadow = true;
        weir.userData = { type: 'feishayan', name: '飞沙堰' };
        group.add(weir);

        const spillwayGeometry = new THREE.BoxGeometry(40, 0.5, 10);
        const spillwayMaterial = new THREE.MeshStandardMaterial({
            color: 0x6b8e23,
            roughness: 0.8,
            metalness: 0.1
        });
        const spillway = new THREE.Mesh(spillwayGeometry, spillwayMaterial);
        spillway.position.set(35, 728, -35);
        spillway.rotation.y = Math.PI / 6;
        spillway.castShadow = true;
        spillway.receiveShadow = true;
        group.add(spillway);

        group.position.y = 0;
        return [group];
    },

    createBaopingkou() {
        const group = new THREE.Group();

        const rockMaterial = new THREE.MeshStandardMaterial({
            color: 0x696969,
            roughness: 0.95,
            metalness: 0.05
        });

        const leftWallGeometry = new THREE.BoxGeometry(3, 12, 25);
        const leftWall = new THREE.Mesh(leftWallGeometry, rockMaterial);
        leftWall.position.set(-12, 731, 55);
        leftWall.castShadow = true;
        leftWall.receiveShadow = true;
        group.add(leftWall);

        const rightWallGeometry = new THREE.BoxGeometry(3, 12, 25);
        const rightWall = new THREE.Mesh(rightWallGeometry, rockMaterial);
        rightWall.position.set(12, 731, 55);
        rightWall.castShadow = true;
        rightWall.receiveShadow = true;
        group.add(rightWall);

        const topGeometry = new THREE.BoxGeometry(28, 2, 8);
        const top = new THREE.Mesh(topGeometry, rockMaterial);
        top.position.set(0, 736, 58);
        top.castShadow = true;
        top.receiveShadow = true;
        top.userData = { type: 'baopingkou', name: '宝瓶口' };
        group.add(top);

        const gateGeometry = new THREE.BoxGeometry(20, 6, 1);
        const gateMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b4513,
            roughness: 0.7,
            metalness: 0.2
        });
        const gate = new THREE.Mesh(gateGeometry, gateMaterial);
        gate.position.set(0, 730, 48);
        gate.castShadow = true;
        gate.receiveShadow = true;
        group.add(gate);

        group.position.y = 0;
        return [group];
    },

    createRenzhidi() {
        const group = new THREE.Group();

        const dikeShape = new THREE.Shape();
        dikeShape.moveTo(0, 0);
        dikeShape.lineTo(30, 0);
        dikeShape.lineTo(25, 8);
        dikeShape.lineTo(5, 8);
        dikeShape.lineTo(0, 0);

        const extrudeSettings = { depth: 60, bevelEnabled: false };
        const dikeGeometry = new THREE.ExtrudeGeometry(dikeShape, extrudeSettings);
        const dikeMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b7355,
            roughness: 0.9,
            metalness: 0.1
        });
        const dike = new THREE.Mesh(dikeGeometry, dikeMaterial);
        dike.position.set(15, 725.5, -40);
        dike.rotation.x = Math.PI / 2;
        dike.rotation.y = Math.PI / 8;
        dike.castShadow = true;
        dike.receiveShadow = true;
        dike.userData = { type: 'renzhidi', name: '人字堤' };
        group.add(dike);

        for (let i = 0; i < 10; i++) {
            const tree = this.createTree();
            tree.position.set(
                20 + Utils.randomRange(-5, 20),
                730,
                -40 + i * 5
            );
            group.add(tree);
        }

        group.position.y = 0;
        return [group];
    },

    createAncientBuildings() {
        const buildings = [];

        const temple = this.createTemple();
        temple.position.set(-50, 728, 20);
        temple.scale.set(0.8, 0.8, 0.8);
        buildings.push(temple);

        const pavilion = this.createPavilion();
        pavilion.position.set(20, 730, 20);
        buildings.push(pavilion);

        const tower = this.createTower();
        tower.position.set(50, 728, 10);
        tower.scale.set(0.7, 0.7, 0.7);
        buildings.push(tower);

        return buildings;
    },

    createTemple() {
        const group = new THREE.Group();

        const baseGeometry = new THREE.BoxGeometry(15, 1, 12);
        const stoneMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b7355,
            roughness: 0.9,
            metalness: 0.1
        });
        const base = new THREE.Mesh(baseGeometry, stoneMaterial);
        base.position.y = 0.5;
        base.castShadow = true;
        base.receiveShadow = true;
        group.add(base);

        const wallGeometry = new THREE.BoxGeometry(13, 6, 10);
        const wallMaterial = new THREE.MeshStandardMaterial({
            color: 0xf5f5dc,
            roughness: 0.8,
            metalness: 0.1
        });
        const walls = new THREE.Mesh(wallGeometry, wallMaterial);
        walls.position.y = 4;
        walls.castShadow = true;
        walls.receiveShadow = true;
        group.add(walls);

        const roofGeometry = new THREE.ConeGeometry(11, 4, 4);
        const roofMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b0000,
            roughness: 0.7,
            metalness: 0.2
        });
        const roof = new THREE.Mesh(roofGeometry, roofMaterial);
        roof.position.y = 9;
        roof.rotation.y = Math.PI / 4;
        roof.castShadow = true;
        roof.receiveShadow = true;
        group.add(roof);

        const doorGeometry = new THREE.BoxGeometry(2, 3.5, 0.2);
        const doorMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b4513,
            roughness: 0.6,
            metalness: 0.3
        });
        const door = new THREE.Mesh(doorGeometry, doorMaterial);
        door.position.set(0, 2.75, 5.1);
        group.add(door);

        return group;
    },

    createPavilion() {
        const group = new THREE.Group();

        const poleGeometry = new THREE.CylinderGeometry(0.2, 0.25, 5, 8);
        const woodMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b4513,
            roughness: 0.7,
            metalness: 0.2
        });

        for (let i = 0; i < 6; i++) {
            const angle = (i / 6) * Math.PI * 2;
            const pole = new THREE.Mesh(poleGeometry, woodMaterial);
            pole.position.set(Math.cos(angle) * 4, 2.5, Math.sin(angle) * 4);
            pole.castShadow = true;
            group.add(pole);
        }

        const roofGeometry = new THREE.ConeGeometry(5.5, 2, 6);
        const roofMaterial = new THREE.MeshStandardMaterial({
            color: 0x228b22,
            roughness: 0.8,
            metalness: 0.1
        });
        const roof = new THREE.Mesh(roofGeometry, roofMaterial);
        roof.position.y = 6;
        roof.rotation.y = Math.PI / 6;
        roof.castShadow = true;
        group.add(roof);

        const topGeometry = new THREE.SphereGeometry(0.4, 16, 16);
        const topMaterial = new THREE.MeshStandardMaterial({
            color: 0xffd700,
            roughness: 0.3,
            metalness: 0.9
        });
        const top = new THREE.Mesh(topGeometry, topMaterial);
        top.position.y = 7.5;
        group.add(top);

        return group;
    },

    createTower() {
        const group = new THREE.Group();

        for (let i = 0; i < 3; i++) {
            const floorGeometry = new THREE.BoxGeometry(8 - i * 0.5, 4, 8 - i * 0.5);
            const floorMaterial = new THREE.MeshStandardMaterial({
                color: 0xf5f5dc,
                roughness: 0.8,
                metalness: 0.1
            });
            const floor = new THREE.Mesh(floorGeometry, floorMaterial);
            floor.position.y = 2 + i * 4.5;
            floor.castShadow = true;
            floor.receiveShadow = true;
            group.add(floor);

            const roofGeometry = new THREE.ConeGeometry(6 - i * 0.5, 2, 4);
            const roofMaterial = new THREE.MeshStandardMaterial({
                color: 0x8b0000,
                roughness: 0.7,
                metalness: 0.2
            });
            const roof = new THREE.Mesh(roofGeometry, roofMaterial);
            roof.position.y = 5 + i * 4.5;
            roof.rotation.y = Math.PI / 4;
            roof.castShadow = true;
            group.add(roof);
        }

        const topGeometry = new THREE.ConeGeometry(1, 3, 8);
        const topMaterial = new THREE.MeshStandardMaterial({
            color: 0xffd700,
            roughness: 0.3,
            metalness: 0.9
        });
        const top = new THREE.Mesh(topGeometry, topMaterial);
        top.position.y = 16;
        group.add(top);

        return group;
    },

    createBambooCageMesh() {
        const group = new THREE.Group();

        const cageGeometry = new THREE.CylinderGeometry(0.5, 0.5, 2, 8);
        const cageMaterial = new THREE.MeshStandardMaterial({
            color: 0x6b8e23,
            roughness: 0.8,
            metalness: 0.1,
            transparent: true,
            opacity: 0.7
        });
        const cage = new THREE.Mesh(cageGeometry, cageMaterial);
        cage.rotation.z = Math.PI / 2;
        cage.castShadow = true;
        group.add(cage);

        for (let i = 0; i < 15; i++) {
            const stoneGeometry = new THREE.SphereGeometry(Utils.randomRange(0.1, 0.25), 8, 8);
            const stoneMaterial = new THREE.MeshStandardMaterial({
                color: 0x888888,
                roughness: 0.95,
                metalness: 0.05
            });
            const stone = new THREE.Mesh(stoneGeometry, stoneMaterial);
            stone.position.set(
                Utils.randomRange(-0.35, 0.35),
                Utils.randomRange(-0.8, 0.8),
                Utils.randomRange(-0.35, 0.35)
            );
            stone.rotation.set(
                Math.random() * Math.PI,
                Math.random() * Math.PI,
                Math.random() * Math.PI
            );
            stone.castShadow = true;
            group.add(stone);
        }

        return group;
    },

    createTree() {
        const group = new THREE.Group();

        const trunkGeometry = new THREE.CylinderGeometry(0.2, 0.3, 3, 8);
        const trunkMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b4513,
            roughness: 0.9,
            metalness: 0.1
        });
        const trunk = new THREE.Mesh(trunkGeometry, trunkMaterial);
        trunk.position.y = 1.5;
        trunk.castShadow = true;
        group.add(trunk);

        const crownGeometry = new THREE.SphereGeometry(2, 8, 8);
        const crownMaterial = new THREE.MeshStandardMaterial({
            color: 0x228b22,
            roughness: 0.8,
            metalness: 0.1
        });
        const crown = new THREE.Mesh(crownGeometry, crownMaterial);
        crown.position.y = 4;
        crown.castShadow = true;
        group.add(crown);

        return group;
    },

    createMachaStructure(x, y, z, height, angle) {
        const group = new THREE.Group();

        const logMaterial = new THREE.MeshStandardMaterial({
            color: 0x8b4513,
            roughness: 0.7,
            metalness: 0.2
        });

        for (let i = 0; i < 3; i++) {
            const logGeometry = new THREE.CylinderGeometry(0.2, 0.2, height, 8);
            const log = new THREE.Mesh(logGeometry, logMaterial);
            
            const radAngle = (angle + (i - 1) * 15) * Math.PI / 180;
            log.position.set(
                x + Math.sin(radAngle) * 1.5,
                y + height / 2,
                z + Math.cos(radAngle) * 1.5
            );
            log.rotation.z = radAngle * 0.3;
            log.rotation.x = -radAngle * 0.2;
            log.castShadow = true;
            group.add(log);
        }

        const tieGeometry = new THREE.CylinderGeometry(0.1, 0.1, 4, 8);
        const tieMaterial = new THREE.MeshStandardMaterial({
            color: 0x6b8e23,
            roughness: 0.8,
            metalness: 0.1
        });

        for (let i = 0; i < 3; i++) {
            const tie = new THREE.Mesh(tieGeometry, tieMaterial);
            tie.position.set(x, y + height * (0.2 + i * 0.3), z);
            tie.rotation.y = Math.PI / 2;
            tie.rotation.z = (i * 60) * Math.PI / 180;
            group.add(tie);
        }

        return group;
    }
};
