package models

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *pgxpool.Pool

func InitDB() error {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := DB.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established successfully")
	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}

func InsertHydrologyData(ctx context.Context, data *HydrologyData) error {
	query := `
		INSERT INTO hydrology_data 
		(time, station_id, station_name, water_level, flow_rate, 
		 sediment_concentration, bed_elevation, temperature, rainfall, sensor_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := DB.Exec(ctx, query,
		data.Time, data.StationID, data.StationName,
		data.WaterLevel, data.FlowRate, data.SedimentConcentration,
		data.BedElevation, data.Temperature, data.Rainfall, data.SensorStatus,
	)
	return err
}

func GetHydrologyData(ctx context.Context, stationID string, startTime, endTime time.Time, limit int) ([]HydrologyData, error) {
	query := `
		SELECT time, station_id, station_name, water_level, flow_rate,
			   sediment_concentration, bed_elevation, temperature, rainfall, sensor_status, created_at
		FROM hydrology_data
		WHERE station_id = $1 AND time BETWEEN $2 AND $3
		ORDER BY time DESC
		LIMIT $4
	`

	rows, err := DB.Query(ctx, query, stationID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []HydrologyData
	for rows.Next() {
		var d HydrologyData
		err := rows.Scan(&d.Time, &d.StationID, &d.StationName, &d.WaterLevel, &d.FlowRate,
			&d.SedimentConcentration, &d.BedElevation, &d.Temperature, &d.Rainfall,
			&d.SensorStatus, &d.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}

func GetLatestHydrologyData(ctx context.Context, stationID string) (*HydrologyData, error) {
	query := `
		SELECT time, station_id, station_name, water_level, flow_rate,
			   sediment_concentration, bed_elevation, temperature, rainfall, sensor_status, created_at
		FROM hydrology_data
		WHERE station_id = $1
		ORDER BY time DESC
		LIMIT 1
	`

	var d HydrologyData
	err := DB.QueryRow(ctx, query, stationID).Scan(
		&d.Time, &d.StationID, &d.StationName, &d.WaterLevel, &d.FlowRate,
		&d.SedimentConcentration, &d.BedElevation, &d.Temperature, &d.Rainfall,
		&d.SensorStatus, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func GetAllLatestHydrologyData(ctx context.Context) ([]HydrologyData, error) {
	query := `
		SELECT DISTINCT ON (station_id) 
			   time, station_id, station_name, water_level, flow_rate,
			   sediment_concentration, bed_elevation, temperature, rainfall, sensor_status, created_at
		FROM hydrology_data
		ORDER BY station_id, time DESC
	`

	rows, err := DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []HydrologyData
	for rows.Next() {
		var d HydrologyData
		err := rows.Scan(&d.Time, &d.StationID, &d.StationName, &d.WaterLevel, &d.FlowRate,
			&d.SedimentConcentration, &d.BedElevation, &d.Temperature, &d.Rainfall,
			&d.SensorStatus, &d.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}

func GetWolongIron(ctx context.Context) ([]WolongIron, error) {
	query := `
		SELECT id, name, location, elevation, description, installed_year, created_at
		FROM wolong_iron
		ORDER BY elevation
	`

	rows, err := DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []WolongIron
	for rows.Next() {
		var w WolongIron
		err := rows.Scan(&w.ID, &w.Name, &w.Location, &w.Elevation,
			&w.Description, &w.InstalledYear, &w.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, w)
	}
	return results, nil
}

func GetMonitoringStations(ctx context.Context) ([]MonitoringStation, error) {
	query := `
		SELECT id, station_id, name, location_lat, location_lng, reach_name, bedrock_elevation, created_at
		FROM monitoring_stations
		ORDER BY station_id
	`

	rows, err := DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MonitoringStation
	for rows.Next() {
		var s MonitoringStation
		err := rows.Scan(&s.ID, &s.StationID, &s.Name, &s.LocationLat, &s.LocationLng,
			&s.ReachName, &s.BedrockElevation, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func GetAlerts(ctx context.Context, acknowledged *bool, limit int) ([]Alert, error) {
	query := `
		SELECT id, alert_time, alert_type, alert_level, station_id, message,
			   bed_elevation, wolong_iron_elevation, exceeded_value, acknowledged,
			   acknowledged_at, acknowledged_by, mqtt_published, created_at
		FROM alerts
	`

	var args []interface{}
	if acknowledged != nil {
		query += " WHERE acknowledged = $1"
		args = append(args, *acknowledged)
	}

	query += " ORDER BY alert_time DESC LIMIT $2"
	args = append(args, limit)

	rows, err := DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Alert
	for rows.Next() {
		var a Alert
		err := rows.Scan(&a.ID, &a.AlertTime, &a.AlertType, &a.AlertLevel, &a.StationID, &a.Message,
			&a.BedElevation, &a.WolongIronElevation, &a.ExceededValue, &a.Acknowledged,
			&a.AcknowledgedAt, &a.AcknowledgedBy, &a.MqttPublished, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

func AcknowledgeAlert(ctx context.Context, alertID int, acknowledgedBy string) error {
	query := `
		UPDATE alerts
		SET acknowledged = TRUE, acknowledged_at = NOW(), acknowledged_by = $1
		WHERE id = $2
	`
	_, err := DB.Exec(ctx, query, acknowledgedBy, alertID)
	return err
}

func GetUnpublishedAlerts(ctx context.Context) ([]Alert, error) {
	query := `
		SELECT id, alert_time, alert_type, alert_level, station_id, message,
			   bed_elevation, wolong_iron_elevation, exceeded_value
		FROM alerts
		WHERE mqtt_published = FALSE
		ORDER BY alert_time ASC
	`

	rows, err := DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Alert
	for rows.Next() {
		var a Alert
		err := rows.Scan(&a.ID, &a.AlertTime, &a.AlertType, &a.AlertLevel, &a.StationID, &a.Message,
			&a.BedElevation, &a.WolongIronElevation, &a.ExceededValue)
		if err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

func MarkAlertAsPublished(ctx context.Context, alertID int) error {
	query := `
		UPDATE alerts
		SET mqtt_published = TRUE
		WHERE id = $1
	`
	_, err := DB.Exec(ctx, query, alertID)
	return err
}

func GetDailyStats(ctx context.Context, stationID string, startTime, endTime time.Time) ([]DailyStats, error) {
	query := `
		SELECT bucket, station_id, avg_water_level, max_water_level, min_water_level,
			   avg_flow_rate, max_flow_rate, avg_sediment, max_sediment, avg_bed_elevation, record_count
		FROM hydrology_daily_stats
		WHERE station_id = $1 AND bucket BETWEEN $2 AND $3
		ORDER BY bucket ASC
	`

	rows, err := DB.Query(ctx, query, stationID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DailyStats
	for rows.Next() {
		var s DailyStats
		err := rows.Scan(&s.Bucket, &s.StationID, &s.AvgWaterLevel, &s.MaxWaterLevel, &s.MinWaterLevel,
			&s.AvgFlowRate, &s.MaxFlowRate, &s.AvgSediment, &s.MaxSediment, &s.AvgBedElevation, &s.RecordCount)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func InsertBedEvolutionPrediction(ctx context.Context, p *BedEvolutionPrediction) error {
	query := `
		INSERT INTO bed_evolution_prediction
		(station_id, prediction_date, forecast_horizon_months, predicted_bed_elevation,
		 predicted_sediment_deposition, predicted_erosion, model_version, confidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := DB.Exec(ctx, query,
		p.StationID, p.PredictionDate, p.ForecastHorizonMonths,
		p.PredictedBedElevation, p.PredictedSedimentDeposition, p.PredictedErosion,
		p.ModelVersion, p.Confidence,
	)
	return err
}

func GetBedEvolutionPredictions(ctx context.Context, stationID string) ([]BedEvolutionPrediction, error) {
	query := `
		SELECT id, station_id, prediction_date, forecast_horizon_months, predicted_bed_elevation,
			   predicted_sediment_deposition, predicted_erosion, model_version, confidence, created_at
		FROM bed_evolution_prediction
		WHERE station_id = $1
		ORDER BY forecast_horizon_months ASC
	`

	rows, err := DB.Query(ctx, query, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []BedEvolutionPrediction
	for rows.Next() {
		var p BedEvolutionPrediction
		err := rows.Scan(&p.ID, &p.StationID, &p.PredictionDate, &p.ForecastHorizonMonths,
			&p.PredictedBedElevation, &p.PredictedSedimentDeposition, &p.PredictedErosion,
			&p.ModelVersion, &p.Confidence, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, nil
}

func InsertAnnualRepairSimulation(ctx context.Context, s *AnnualRepairSimulation) (int64, error) {
	query := `
		INSERT INTO annual_repair_simulation
		(simulation_name, simulation_type, status, parameters, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int64
	err := DB.QueryRow(ctx, query, s.SimulationName, s.SimulationType, s.Status, s.Parameters, s.CreatedBy).Scan(&id)
	return id, err
}

func GetAnnualRepairSimulations(ctx context.Context, limit int) ([]AnnualRepairSimulation, error) {
	query := `
		SELECT id, simulation_name, simulation_type, start_time, end_time, status, created_by, created_at
		FROM annual_repair_simulation
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := DB.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AnnualRepairSimulation
	for rows.Next() {
		var s AnnualRepairSimulation
		err := rows.Scan(&s.ID, &s.SimulationName, &s.SimulationType, &s.StartTime, &s.EndTime,
			&s.Status, &s.CreatedBy, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func InsertMachaInterceptionData(ctx context.Context, d *MachaInterceptionData) error {
	query := `
		INSERT INTO macha_interception_simulation
		(time, simulation_id, position_x, position_y, water_level_before, water_level_after,
		 flow_rate_before, flow_rate_after, interception_efficiency, macha_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := DB.Exec(ctx, query,
		d.Time, d.SimulationID, d.PositionX, d.PositionY,
		d.WaterLevelBefore, d.WaterLevelAfter, d.FlowRateBefore, d.FlowRateAfter,
		d.InterceptionEfficiency, d.MachaCount,
	)
	return err
}

func InsertBambooCageData(ctx context.Context, d *BambooCageData) error {
	query := `
		INSERT INTO bamboo_cage_simulation
		(simulation_id, cage_id, position_x, position_y, position_z, stone_count,
		 cage_diameter, cage_length, porosity, stability_coefficient, deposition_height)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := DB.Exec(ctx, query,
		d.SimulationID, d.CageID, d.PositionX, d.PositionY, d.PositionZ, d.StoneCount,
		d.CageDiameter, d.CageLength, d.Porosity, d.StabilityCoefficient, d.DepositionHeight,
	)
	return err
}

func GetAnnualRepairRecords(ctx context.Context) ([]AnnualRepairRecord, error) {
	query := `
		SELECT id, repair_year, start_date, end_date, location, repair_type,
			   bamboo_cage_count, macha_count, dredging_volume, bed_elevation_before,
			   bed_elevation_after, notes, created_at
		FROM annual_repair_records
		ORDER BY repair_year DESC
	`

	rows, err := DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AnnualRepairRecord
	for rows.Next() {
		var r AnnualRepairRecord
		err := rows.Scan(&r.ID, &r.RepairYear, &r.StartDate, &r.EndDate, &r.Location, &r.RepairType,
			&r.BambooCageCount, &r.MachaCount, &r.DredgingVolume, &r.BedElevationBefore,
			&r.BedElevationAfter, &r.Notes, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func GetMachaSimulationData(ctx context.Context, simulationID int) ([]MachaInterceptionData, error) {
	query := `
		SELECT time, simulation_id, position_x, position_y, water_level_before, water_level_after,
			   flow_rate_before, flow_rate_after, interception_efficiency, macha_count, created_at
		FROM macha_interception_simulation
		WHERE simulation_id = $1
		ORDER BY time ASC
	`

	rows, err := DB.Query(ctx, query, simulationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MachaInterceptionData
	for rows.Next() {
		var d MachaInterceptionData
		err := rows.Scan(&d.Time, &d.SimulationID, &d.PositionX, &d.PositionY,
			&d.WaterLevelBefore, &d.WaterLevelAfter, &d.FlowRateBefore, &d.FlowRateAfter,
			&d.InterceptionEfficiency, &d.MachaCount, &d.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}

func GetBambooCageSimulationData(ctx context.Context, simulationID int) ([]BambooCageData, error) {
	query := `
		SELECT id, simulation_id, cage_id, position_x, position_y, position_z, stone_count,
			   cage_diameter, cage_length, porosity, stability_coefficient, deposition_height, created_at
		FROM bamboo_cage_simulation
		WHERE simulation_id = $1
		ORDER BY cage_id
	`

	rows, err := DB.Query(ctx, query, simulationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []BambooCageData
	for rows.Next() {
		var d BambooCageData
		err := rows.Scan(&d.ID, &d.SimulationID, &d.CageID, &d.PositionX, &d.PositionY, &d.PositionZ,
			&d.StoneCount, &d.CageDiameter, &d.CageLength, &d.Porosity, &d.StabilityCoefficient,
			&d.DepositionHeight, &d.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}

func UpdateSimulationStatus(ctx context.Context, simulationID int64, status string) error {
	query := `
		UPDATE annual_repair_simulation
		SET status = $1
		WHERE id = $2
	`
	_, err := DB.Exec(ctx, query, status, simulationID)
	return err
}

func UpdateSimulationResult(ctx context.Context, simulationID int64, result interface{}) error {
	query := `
		UPDATE annual_repair_simulation
		SET status = 'completed', end_time = NOW(), result = $1
		WHERE id = $2
	`
	_, err := DB.Exec(ctx, query, result, simulationID)
	return err
}
