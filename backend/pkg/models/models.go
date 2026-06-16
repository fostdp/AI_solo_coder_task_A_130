package models

import (
	"time"
)

type HydrologyData struct {
	Time                  time.Time `json:"time" db:"time"`
	StationID             string    `json:"station_id" db:"station_id"`
	StationName           string    `json:"station_name" db:"station_name"`
	WaterLevel            float64   `json:"water_level" db:"water_level"`
	FlowRate              float64   `json:"flow_rate" db:"flow_rate"`
	SedimentConcentration float64   `json:"sediment_concentration" db:"sediment_concentration"`
	BedElevation          float64   `json:"bed_elevation" db:"bed_elevation"`
	Temperature           float64   `json:"temperature" db:"temperature"`
	Rainfall              float64   `json:"rainfall" db:"rainfall"`
	SensorStatus          int       `json:"sensor_status" db:"sensor_status"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

type WolongIron struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Location      string    `json:"location" db:"location"`
	Elevation     float64   `json:"elevation" db:"elevation"`
	Description   string    `json:"description" db:"description"`
	InstalledYear int       `json:"installed_year" db:"installed_year"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type MonitoringStation struct {
	ID              int       `json:"id" db:"id"`
	StationID       string    `json:"station_id" db:"station_id"`
	Name            string    `json:"name" db:"name"`
	LocationLat     float64   `json:"location_lat" db:"location_lat"`
	LocationLng     float64   `json:"location_lng" db:"location_lng"`
	ReachName       string    `json:"reach_name" db:"reach_name"`
	BedrockElevation float64  `json:"bedrock_elevation" db:"bedrock_elevation"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type Alert struct {
	ID                  int        `json:"id" db:"id"`
	AlertTime           time.Time  `json:"alert_time" db:"alert_time"`
	AlertType           string     `json:"alert_type" db:"alert_type"`
	AlertLevel          string     `json:"alert_level" db:"alert_level"`
	StationID           string     `json:"station_id" db:"station_id"`
	Message             string     `json:"message" db:"message"`
	BedElevation        float64    `json:"bed_elevation" db:"bed_elevation"`
	WolongIronElevation float64    `json:"wolong_iron_elevation" db:"wolong_iron_elevation"`
	ExceededValue       float64    `json:"exceeded_value" db:"exceeded_value"`
	Acknowledged        bool       `json:"acknowledged" db:"acknowledged"`
	AcknowledgedAt      *time.Time `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	AcknowledgedBy      string     `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	MqttPublished       bool       `json:"mqtt_published" db:"mqtt_published"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

type BedEvolutionPrediction struct {
	ID                        int       `json:"id" db:"id"`
	StationID                 string    `json:"station_id" db:"station_id"`
	PredictionDate            time.Time `json:"prediction_date" db:"prediction_date"`
	ForecastHorizonMonths     int       `json:"forecast_horizon_months" db:"forecast_horizon_months"`
	PredictedBedElevation     float64   `json:"predicted_bed_elevation" db:"predicted_bed_elevation"`
	PredictedSedimentDeposition float64 `json:"predicted_sediment_deposition" db:"predicted_sediment_deposition"`
	PredictedErosion          float64   `json:"predicted_erosion" db:"predicted_erosion"`
	ModelVersion              string    `json:"model_version" db:"model_version"`
	Confidence                float64   `json:"confidence" db:"confidence"`
	CreatedAt                 time.Time `json:"created_at" db:"created_at"`
}

type AnnualRepairSimulation struct {
	ID             int64           `json:"id" db:"id"`
	SimulationName string          `json:"simulation_name" db:"simulation_name"`
	SimulationType string          `json:"simulation_type" db:"simulation_type"`
	StartTime      *time.Time      `json:"start_time,omitempty" db:"start_time"`
	EndTime        *time.Time      `json:"end_time,omitempty" db:"end_time"`
	Status         string          `json:"status" db:"status"`
	Parameters     SimulationParams `json:"parameters" db:"parameters"`
	Result         interface{}     `json:"result" db:"result"`
	CreatedBy      string          `json:"created_by" db:"created_by"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

type SimulationParams struct {
	Location     string    `json:"location"`
	StartTime    time.Time `json:"start_time"`
	DurationDays int       `json:"duration_days"`
	GridSize     float64   `json:"grid_size"`
	TimeStep     int       `json:"time_step"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

type MachaInterceptionData struct {
	Time                  time.Time `json:"time" db:"time"`
	SimulationID          int       `json:"simulation_id" db:"simulation_id"`
	PositionX             float64   `json:"position_x" db:"position_x"`
	PositionY             float64   `json:"position_y" db:"position_y"`
	WaterLevelBefore      float64   `json:"water_level_before" db:"water_level_before"`
	WaterLevelAfter       float64   `json:"water_level_after" db:"water_level_after"`
	FlowRateBefore        float64   `json:"flow_rate_before" db:"flow_rate_before"`
	FlowRateAfter         float64   `json:"flow_rate_after" db:"flow_rate_after"`
	InterceptionEfficiency float64  `json:"interception_efficiency" db:"interception_efficiency"`
	MachaCount            int       `json:"macha_count" db:"macha_count"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

type BambooCageData struct {
	ID                    int       `json:"id" db:"id"`
	SimulationID          int       `json:"simulation_id" db:"simulation_id"`
	CageID                string    `json:"cage_id" db:"cage_id"`
	PositionX             float64   `json:"position_x" db:"position_x"`
	PositionY             float64   `json:"position_y" db:"position_y"`
	PositionZ             float64   `json:"position_z" db:"position_z"`
	StoneCount            int       `json:"stone_count" db:"stone_count"`
	CageDiameter          float64   `json:"cage_diameter" db:"cage_diameter"`
	CageLength            float64   `json:"cage_length" db:"cage_length"`
	Porosity              float64   `json:"porosity" db:"porosity"`
	StabilityCoefficient  float64   `json:"stability_coefficient" db:"stability_coefficient"`
	DepositionHeight      float64   `json:"deposition_height" db:"deposition_height"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

type AnnualRepairRecord struct {
	ID                 int       `json:"id" db:"id"`
	RepairYear         int       `json:"repair_year" db:"repair_year"`
	StartDate          string    `json:"start_date" db:"start_date"`
	EndDate            string    `json:"end_date" db:"end_date"`
	Location           string    `json:"location" db:"location"`
	RepairType         string    `json:"repair_type" db:"repair_type"`
	BambooCageCount    int       `json:"bamboo_cage_count" db:"bamboo_cage_count"`
	MachaCount         int       `json:"macha_count" db:"macha_count"`
	DredgingVolume     float64   `json:"dredging_volume" db:"dredging_volume"`
	BedElevationBefore float64   `json:"bed_elevation_before" db:"bed_elevation_before"`
	BedElevationAfter  float64   `json:"bed_elevation_after" db:"bed_elevation_after"`
	Notes              string    `json:"notes" db:"notes"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

type DailyStats struct {
	Bucket          time.Time `json:"bucket" db:"bucket"`
	StationID       string    `json:"station_id" db:"station_id"`
	AvgWaterLevel   float64   `json:"avg_water_level" db:"avg_water_level"`
	MaxWaterLevel   float64   `json:"max_water_level" db:"max_water_level"`
	MinWaterLevel   float64   `json:"min_water_level" db:"min_water_level"`
	AvgFlowRate     float64   `json:"avg_flow_rate" db:"avg_flow_rate"`
	MaxFlowRate     float64   `json:"max_flow_rate" db:"max_flow_rate"`
	AvgSediment     float64   `json:"avg_sediment" db:"avg_sediment"`
	MaxSediment     float64   `json:"max_sediment" db:"max_sediment"`
	AvgBedElevation float64   `json:"avg_bed_elevation" db:"avg_bed_elevation"`
	RecordCount     int       `json:"record_count" db:"record_count"`
}

type DEMGrid struct {
	GridX     int     `json:"grid_x"`
	GridY     int     `json:"grid_y"`
	Elevation float64 `json:"elevation"`
	WaterDepth float64 `json:"water_depth"`
}
