-- ============================================
-- 都江堰岁修工艺仿真与河床演变分析系统
-- TimescaleDB 初始化脚本
-- ============================================

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ============================================
-- 1. 参考基准表：卧铁高程
-- ============================================
CREATE TABLE IF NOT EXISTS wolong_iron (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    location VARCHAR(100),
    elevation NUMERIC(8,3) NOT NULL,
    description TEXT,
    installed_year INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO wolong_iron (name, location, elevation, description, installed_year) VALUES
('内江河口卧铁', '都江堰内江进水口', 730.500, '内江第一卧铁，作为岁修淘滩深度基准', 1573),
('外江河口卧铁', '都江堰外江进水口', 730.200, '外江第一卧铁，作为外江岁修基准', 1573),
('飞沙堰卧铁', '飞沙堰段', 728.800, '飞沙堰泄洪道卧铁', 1744),
('宝瓶口卧铁', '宝瓶口入口处', 729.300, '宝瓶口控制基准卧铁', 1642)
ON CONFLICT DO NOTHING;

-- ============================================
-- 2. 水文监测数据表（时序数据）
-- ============================================
CREATE TABLE IF NOT EXISTS hydrology_data (
    time TIMESTAMPTZ NOT NULL,
    station_id VARCHAR(50) NOT NULL,
    station_name VARCHAR(100),
    water_level NUMERIC(8,3),
    flow_rate NUMERIC(10,2),
    sediment_concentration NUMERIC(8,4),
    bed_elevation NUMERIC(8,3),
    temperature NUMERIC(5,2),
    rainfall NUMERIC(8,2),
    sensor_status INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

SELECT create_hypertable('hydrology_data', 'time', 
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_hydrology_station_time ON hydrology_data (station_id, time DESC);

-- ============================================
-- 3. 监测站点表
-- ============================================
CREATE TABLE IF NOT EXISTS monitoring_stations (
    id SERIAL PRIMARY KEY,
    station_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    location_lat NUMERIC(10,6),
    location_lng NUMERIC(10,6),
    reach_name VARCHAR(100),
    bedrock_elevation NUMERIC(8,3),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO monitoring_stations (station_id, name, location_lat, location_lng, reach_name, bedrock_elevation) VALUES
('NEIJ-001', '内江进水口', 30.985, 103.621, '内江段', 726.500),
('NEIJ-002', '内江中段', 30.987, 103.625, '内江段', 725.800),
('NEIJ-003', '宝瓶口上游', 30.990, 103.628, '内江段', 726.200),
('WAIJ-001', '外江进水口', 30.983, 103.618, '外江段', 726.000),
('WAIJ-002', '外江中段', 30.980, 103.615, '外江段', 725.500),
('FSSY-001', '飞沙堰进口', 30.986, 103.622, '飞沙堰段', 727.000),
('FSSY-002', '飞沙堰出口', 30.984, 103.624, '飞沙堰段', 726.800),
('RJK-001', '人字堤', 30.982, 103.620, '外江段', 726.300)
ON CONFLICT DO NOTHING;

-- ============================================
-- 4. 河床演变预测数据表
-- ============================================
CREATE TABLE IF NOT EXISTS bed_evolution_prediction (
    id SERIAL PRIMARY KEY,
    station_id VARCHAR(50) NOT NULL,
    prediction_date TIMESTAMPTZ NOT NULL,
    forecast_horizon_months INT NOT NULL,
    predicted_bed_elevation NUMERIC(8,3),
    predicted_sediment_deposition NUMERIC(8,3),
    predicted_erosion NUMERIC(8,3),
    model_version VARCHAR(20),
    confidence NUMERIC(5,2),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

SELECT create_hypertable('bed_evolution_prediction', 'prediction_date', 
    chunk_time_interval => INTERVAL '1 month',
    if_not_exists => TRUE
);

-- ============================================
-- 5. 岁修工艺仿真数据表
-- ============================================
CREATE TABLE IF NOT EXISTS annual_repair_simulation (
    id SERIAL PRIMARY KEY,
    simulation_name VARCHAR(100) NOT NULL,
    simulation_type VARCHAR(50) NOT NULL,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'pending',
    parameters JSONB,
    result JSONB,
    created_by VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 6. 杩槎截流仿真数据表
-- ============================================
CREATE TABLE IF NOT EXISTS macha_interception_simulation (
    time TIMESTAMPTZ NOT NULL,
    simulation_id INT NOT NULL,
    position_x NUMERIC(8,2),
    position_y NUMERIC(8,2),
    water_level_before NUMERIC(8,3),
    water_level_after NUMERIC(8,3),
    flow_rate_before NUMERIC(10,2),
    flow_rate_after NUMERIC(10,2),
    interception_efficiency NUMERIC(5,2),
    macha_count INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

SELECT create_hypertable('macha_interception_simulation', 'time', 
    chunk_time_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- ============================================
-- 7. 竹笼装石仿真数据表
-- ============================================
CREATE TABLE IF NOT EXISTS bamboo_cage_simulation (
    id SERIAL PRIMARY KEY,
    simulation_id INT NOT NULL,
    cage_id VARCHAR(50) NOT NULL,
    position_x NUMERIC(8,2),
    position_y NUMERIC(8,2),
    position_z NUMERIC(8,2),
    stone_count INT,
    cage_diameter NUMERIC(5,2),
    cage_length NUMERIC(6,2),
    porosity NUMERIC(5,2),
    stability_coefficient NUMERIC(5,2),
    deposition_height NUMERIC(8,3),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 8. 告警数据表
-- ============================================
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    alert_time TIMESTAMPTZ NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    alert_level VARCHAR(20) NOT NULL,
    station_id VARCHAR(50),
    message TEXT NOT NULL,
    bed_elevation NUMERIC(8,3),
    wolong_iron_elevation NUMERIC(8,3),
    exceeded_value NUMERIC(8,3),
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(50),
    mqtt_published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

SELECT create_hypertable('alerts', 'alert_time', 
    chunk_time_interval => INTERVAL '1 week',
    if_not_exists => TRUE
);

-- ============================================
-- 9. 岁修记录表
-- ============================================
CREATE TABLE IF NOT EXISTS annual_repair_records (
    id SERIAL PRIMARY KEY,
    repair_year INT NOT NULL,
    start_date DATE,
    end_date DATE,
    location VARCHAR(100),
    repair_type VARCHAR(50),
    bamboo_cage_count INT,
    macha_count INT,
    dredging_volume NUMERIC(10,2),
    bed_elevation_before NUMERIC(8,3),
    bed_elevation_after NUMERIC(8,3),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 10. 连续聚合视图：水文数据日统计
-- ============================================
CREATE MATERIALIZED VIEW IF NOT EXISTS hydrology_daily_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS bucket,
    station_id,
    AVG(water_level) AS avg_water_level,
    MAX(water_level) AS max_water_level,
    MIN(water_level) AS min_water_level,
    AVG(flow_rate) AS avg_flow_rate,
    MAX(flow_rate) AS max_flow_rate,
    AVG(sediment_concentration) AS avg_sediment,
    MAX(sediment_concentration) AS max_sediment,
    AVG(bed_elevation) AS avg_bed_elevation,
    COUNT(*) AS record_count
FROM hydrology_data
GROUP BY bucket, station_id
WITH NO DATA;

-- ============================================
-- 11. 连续聚合视图：河床演变月统计
-- ============================================
CREATE MATERIALIZED VIEW IF NOT EXISTS bed_evolution_monthly_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 month', time) AS bucket,
    station_id,
    FIRST(bed_elevation, time) AS start_elevation,
    LAST(bed_elevation, time) AS end_elevation,
    LAST(bed_elevation, time) - FIRST(bed_elevation, time) AS elevation_change,
    AVG(sediment_concentration) AS avg_sediment,
    SUM(CASE WHEN bed_elevation > LAG(bed_elevation) OVER (PARTITION BY station_id ORDER BY time) 
             THEN bed_elevation - LAG(bed_elevation) OVER (PARTITION BY station_id ORDER BY time) 
             ELSE 0 END) AS total_deposition,
    SUM(CASE WHEN bed_elevation < LAG(bed_elevation) OVER (PARTITION BY station_id ORDER BY time) 
             THEN LAG(bed_elevation) OVER (PARTITION BY station_id ORDER BY time) - bed_elevation 
             ELSE 0 END) AS total_erosion
FROM hydrology_data
GROUP BY bucket, station_id
WITH NO DATA;

-- ============================================
-- 12. 创建告警检查函数
-- ============================================
CREATE OR REPLACE FUNCTION check_bed_elevation_alert()
RETURNS TRIGGER AS $$
DECLARE
    wolong_elev NUMERIC;
    station_name_var VARCHAR(100);
    alert_msg TEXT;
    exceeded NUMERIC;
BEGIN
    SELECT w.elevation, m.name INTO wolong_elev, station_name_var
    FROM monitoring_stations m
    LEFT JOIN wolong_iron w ON TRUE
    WHERE m.station_id = NEW.station_id
    ORDER BY ABS(w.elevation - m.bedrock_elevation)
    LIMIT 1;

    IF wolong_elev IS NOT NULL AND NEW.bed_elevation > wolong_elev THEN
        exceeded := NEW.bed_elevation - wolong_elev;
        alert_msg := format('【河床淤积告警】站点 %s 当前河床高程 %.3fm 超过卧铁基准 %.3fm，淤积量 %.3fm', 
                           station_name_var, NEW.bed_elevation, wolong_elev, exceeded);
        
        INSERT INTO alerts (
            alert_time, alert_type, alert_level, station_id, message,
            bed_elevation, wolong_iron_elevation, exceeded_value, mqtt_published
        ) VALUES (
            NEW.time, 'BED_ELEVATION_EXCEEDED', 
            CASE WHEN exceeded > 1.0 THEN 'CRITICAL'
                 WHEN exceeded > 0.5 THEN 'WARNING'
                 ELSE 'NOTICE' END,
            NEW.station_id, alert_msg,
            NEW.bed_elevation, wolong_elev, exceeded, FALSE
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_bed_elevation_alert ON hydrology_data;
CREATE TRIGGER trigger_bed_elevation_alert
AFTER INSERT ON hydrology_data
FOR EACH ROW
EXECUTE FUNCTION check_bed_elevation_alert();

-- ============================================
-- 13. 初始化一些历史数据用于演示
-- ============================================
INSERT INTO annual_repair_records 
    (repair_year, start_date, end_date, location, repair_type, 
     bamboo_cage_count, macha_count, dredging_volume, 
     bed_elevation_before, bed_elevation_after, notes)
VALUES
    (2020, '2020-12-01', '2020-12-20', '内江段', '全面岁修', 1200, 45, 8500.5, 728.500, 727.200, '岁修完成，淘滩深度达标'),
    (2021, '2021-12-05', '2021-12-22', '内江段', '重点维修', 980, 38, 7200.0, 728.300, 727.000, '竹笼加固飞沙堰'),
    (2022, '2022-11-28', '2022-12-18', '内外江', '全面岁修', 1500, 52, 9800.5, 728.800, 727.100, '杩槎截流效果良好'),
    (2023, '2023-12-02', '2023-12-20', '外江段', '重点维修', 850, 30, 6500.0, 728.000, 726.800, '外江淘滩深度达标')
ON CONFLICT DO NOTHING;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO dujiangyan_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO dujiangyan_user;
