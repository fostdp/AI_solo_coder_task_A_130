const CONFIG = {
    API_BASE_URL: '/api/v1',
    WS_URL: 'ws://' + window.location.host + '/api/v1/ws/realtime',
    
    STATIONS: [
        { id: 'NEIJ-001', name: '内江进水口', color: '#00d4ff' },
        { id: 'NEIJ-002', name: '内江中段', color: '#00ff88' },
        { id: 'NEIJ-003', name: '宝瓶口上游', color: '#ffaa00' },
        { id: 'WAIJ-001', name: '外江进水口', color: '#ff4444' },
        { id: 'WAIJ-002', name: '外江中段', color: '#ff66cc' },
        { id: 'FSSY-001', name: '飞沙堰进口', color: '#aa88ff' },
        { id: 'FSSY-002', name: '飞沙堰出口', color: '#88ffaa' },
        { id: 'RJK-001', name: '人字堤', color: '#ff8844' }
    ],

    COLORS: {
        water: '#00a8ff',
        waterDeep: '#0066aa',
        terrain: '#8b7355',
        terrainDark: '#5c4a3a',
        sand: '#d4a574',
        stone: '#888888',
        bamboo: '#6b8e23',
        wood: '#8b4513',
        wolongIron: '#c0c0c0',
        station: '#ff4444',
        grid: 'rgba(0, 212, 255, 0.1)'
    },

    ELEVATION_RANGE: {
        min: 724,
        max: 732
    },

    VIEWS: {
        overview: { position: { x: 100, y: 80, z: 100 }, target: { x: 0, y: 0, z: 0 } },
        neijiang: { position: { x: 50, y: 40, z: 30 }, target: { x: -20, y: 0, z: 0 } },
        waijiang: { position: { x: -50, y: 40, z: 30 }, target: { x: 20, y: 0, z: 0 } },
        baopingkou: { position: { x: 0, y: 30, z: 60 }, target: { x: 0, y: 0, z: 40 } },
        feishayan: { position: { x: 30, y: 25, z: -20 }, target: { x: 20, y: 0, z: -10 } }
    }
};
