-- ============================================
-- 都江堰岁修工艺仿真与河床演变分析系统
-- TimescaleDB 降采样与保留策略脚本
-- ============================================

-- ============================================
-- 1. 数据保留策略 (Data Retention)
-- ============================================

-- 原始水文数据保留 2 年
SELECT add_retention_policy('hydrology_data', INTERVAL '2 years', if_not_exists => TRUE);

-- 告警数据保留 5 年
SELECT add_retention_policy('alerts', INTERVAL '5 years', if_not_exists => TRUE);

-- 杩槎截流仿真时序数据保留 1 年
SELECT add_retention_policy('macha_interception_simulation', INTERVAL '1 year', if_not_exists => TRUE);

-- 河床演变预测数据保留 10 年
SELECT add_retention_policy('bed_evolution_prediction', INTERVAL '10 years', if_not_exists => TRUE);

-- ============================================
-- 2. 连续聚合：小时级降采样
-- ============================================

CREATE MATERIALIZED VIEW IF NOT EXISTS hydrology_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    station_id,
    AVG(water_level) AS avg_water_level,
    MAX(water_level) AS max_water_level,
    MIN(water_level) AS min_water_level,
    FIRST(water_level, time) AS first_water_level,
    LAST(water_level, time) AS last_water_level,
    AVG(flow_rate) AS avg_flow_rate,
    MAX(flow_rate) AS max_flow_rate,
    MIN(flow_rate) AS min_flow_rate,
    AVG(sediment_concentration) AS avg_sediment,
    MAX(sediment_concentration) AS max_sediment,
    MIN(sediment_concentration) AS min_sediment,
    AVG(bed_elevation) AS avg_bed_elevation,
    MAX(bed_elevation) AS max_bed_elevation,
    MIN(bed_elevation) AS min_bed_elevation,
    AVG(temperature) AS avg_temperature,
    SUM(rainfall) AS total_rainfall,
    COUNT(*) AS record_count
FROM hydrology_data
GROUP BY bucket, station_id
WITH NO DATA;

-- 小时级聚合保留 1 年
SELECT add_retention_policy('hydrology_hourly', INTERVAL '1 year', if_not_exists => TRUE);

-- 每 30 分钟刷新一次小时级聚合
SELECT add_continuous_aggregate_policy('hydrology_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '30 minutes',
    schedule_interval => INTERVAL '30 minutes',
    if_not_exists => TRUE
);

-- ============================================
-- 3. 连续聚合：日级降采样
-- ============================================

CREATE MATERIALIZED VIEW IF NOT EXISTS hydrology_daily
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS bucket,
    station_id,
    AVG(water_level) AS avg_water_level,
    MAX(water_level) AS max_water_level,
    MIN(water_level) AS min_water_level,
    FIRST(water_level, time) AS open_water_level,
    LAST(water_level, time) AS close_water_level,
    AVG(flow_rate) AS avg_flow_rate,
    MAX(flow_rate) AS max_flow_rate,
    MIN(flow_rate) AS min_flow_rate,
    AVG(sediment_concentration) AS avg_sediment,
    MAX(sediment_concentration) AS max_sediment,
    MIN(sediment_concentration) AS min_sediment,
    AVG(bed_elevation) AS avg_bed_elevation,
    MAX(bed_elevation) AS max_bed_elevation,
    MIN(bed_elevation) AS min_bed_elevation,
    FIRST(bed_elevation, time) AS open_bed_elevation,
    LAST(bed_elevation, time) AS close_bed_elevation,
    (LAST(bed_elevation, time) - FIRST(bed_elevation, time)) AS bed_elevation_change,
    AVG(temperature) AS avg_temperature,
    MAX(temperature) AS max_temperature,
    MIN(temperature) AS min_temperature,
    SUM(rainfall) AS total_rainfall,
    COUNT(*) AS record_count
FROM hydrology_data
GROUP BY bucket, station_id
WITH NO DATA;

-- 日级聚合保留 5 年
SELECT add_retention_policy('hydrology_daily', INTERVAL '5 years', if_not_exists => TRUE);

-- 每 2 小时刷新一次日级聚合
SELECT add_continuous_aggregate_policy('hydrology_daily',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '2 hours',
    if_not_exists => TRUE
);

-- ============================================
-- 4. 连续聚合：月级降采样
-- ============================================

CREATE MATERIALIZED VIEW IF NOT EXISTS hydrology_monthly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 month', time) AS bucket,
    station_id,
    AVG(water_level) AS avg_water_level,
    MAX(water_level) AS max_water_level,
    MIN(water_level) AS min_water_level,
    AVG(flow_rate) AS avg_flow_rate,
    MAX(flow_rate) AS max_flow_rate,
    MIN(flow_rate) AS min_flow_rate,
    AVG(sediment_concentration) AS avg_sediment,
    MAX(sediment_concentration) AS max_sediment,
    AVG(bed_elevation) AS avg_bed_elevation,
    MAX(bed_elevation) AS max_bed_elevation,
    MIN(bed_elevation) AS min_bed_elevation,
    FIRST(bed_elevation, time) AS start_bed_elevation,
    LAST(bed_elevation, time) AS end_bed_elevation,
    (LAST(bed_elevation, time) - FIRST(bed_elevation, time)) AS bed_elevation_change,
    SUM(rainfall) AS total_rainfall,
    COUNT(*) AS record_count
FROM hydrology_data
GROUP BY bucket, station_id
WITH NO DATA;

-- 月级聚合永久保留 (用于长期趋势分析)
-- 不设置保留策略

-- 每天刷新一次月级聚合
SELECT add_continuous_aggregate_policy('hydrology_monthly',
    start_offset => INTERVAL '3 months',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- ============================================
-- 5. 告警数据日统计聚合
-- ============================================

CREATE MATERIALIZED VIEW IF NOT EXISTS alerts_daily_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', alert_time) AS bucket,
    station_id,
    alert_level,
    COUNT(*) AS alert_count,
    COUNT(*) FILTER (WHERE acknowledged) AS acknowledged_count
FROM alerts
GROUP BY bucket, station_id, alert_level
WITH NO DATA;

-- 告警日统计保留 5 年
SELECT add_retention_policy('alerts_daily_stats', INTERVAL '5 years', if_not_exists => TRUE);

-- 每小时刷新一次告警统计
SELECT add_continuous_aggregate_policy('alerts_daily_stats',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '30 minutes',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- ============================================
-- 6. 数据压缩策略
-- ============================================

-- 启用原始水文数据压缩
ALTER TABLE hydrology_data SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'station_id',
    timescaledb.compress_orderby = 'time DESC'
);

-- 7 天后的数据自动压缩
SELECT add_compression_policy('hydrology_data', INTERVAL '7 days', if_not_exists => TRUE);

-- 启用告警数据压缩
ALTER TABLE alerts SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'station_id, alert_level',
    timescaledb.compress_orderby = 'alert_time DESC'
);

-- 30 天后的告警数据自动压缩
SELECT add_compression_policy('alerts', INTERVAL '30 days', if_not_exists => TRUE);

-- 启用杩槎仿真数据压缩
ALTER TABLE macha_interception_simulation SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'simulation_id',
    timescaledb.compress_orderby = 'time DESC'
);

-- 1 天后的仿真数据自动压缩
SELECT add_compression_policy('macha_interception_simulation', INTERVAL '1 day', if_not_exists => TRUE);

-- ============================================
-- 7. 信息函数：查询保留策略状态
-- ============================================

CREATE OR REPLACE FUNCTION get_retention_info()
RETURNS TABLE (
    table_name TEXT,
    retention_interval INTERVAL,
    drop_after TEXT,
    next_run TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        j.job_id::TEXT AS table_name,
        j.config ->> 'drop_after' AS retention_interval,
        j.config ->> 'drop_after' AS drop_after,
        js.next_start AS next_run
    FROM timescaledb_information.jobs j
    JOIN timescaledb_information.job_stats js ON j.job_id = js.job_id
    WHERE j.proc_name = 'policy_retention'
    ORDER BY j.job_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 8. 信息函数：查询压缩策略状态
-- ============================================

CREATE OR REPLACE FUNCTION get_compression_info()
RETURNS TABLE (
    table_name TEXT,
    compress_after INTERVAL,
    total_chunks BIGINT,
    compressed_chunks BIGINT,
    compression_ratio NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        h.table_name::TEXT,
        (j.config ->> 'compress_after')::INTERVAL AS compress_after,
        h.total_chunks,
        h.compressed_chunks,
        CASE
            WHEN h.total_chunks > 0
            THEN ROUND((h.compressed_chunks::NUMERIC / h.total_chunks::NUMERIC) * 100, 2)
            ELSE 0
        END AS compression_ratio
    FROM timescaledb_information.compression_settings h
    LEFT JOIN timescaledb_information.jobs j
        ON j.hypertable_name = h.table_name AND j.proc_name = 'policy_compression'
    ORDER BY h.table_name;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 9. 验证：列出所有策略
-- ============================================

-- 执行以下查询查看所有策略：
-- SELECT * FROM timescaledb_information.jobs WHERE proc_name = 'policy_retention';
-- SELECT * FROM timescaledb_information.jobs WHERE proc_name = 'policy_compression';
-- SELECT * FROM timescaledb_information.continuous_aggregates;
-- SELECT * FROM get_retention_info();
-- SELECT * FROM get_compression_info();
