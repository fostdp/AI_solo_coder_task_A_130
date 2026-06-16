package msg

import (
	"time"

	"dujiangyan-system/pkg/models"
)

type MessageType string

const (
	TypeHydrologyData  MessageType = "hydrology"
	TypeAlert          MessageType = "alert"
	TypeSimRequest     MessageType = "sim_request"
	TypeSimResult      MessageType = "sim_result"
	TypePredictRequest MessageType = "predict_request"
	TypePredictResult  MessageType = "predict_result"
)

type BusMessage struct {
	Type      MessageType   `json:"type"`
	Timestamp time.Time     `json:"timestamp"`
	Payload   interface{}   `json:"payload"`
}

type HydrologyPayload struct {
	Data *models.HydrologyData `json:"data"`
}

type AlertPayload struct {
	Alert *models.Alert `json:"alert"`
}

type SimRequestPayload struct {
	SimType    string  `json:"sim_type"`
	Location   string  `json:"location"`
	CageCount  int     `json:"cage_count,omitempty"`
	MachaCount int     `json:"macha_count,omitempty"`
	FlowRate   float64 `json:"flow_rate,omitempty"`
	WaterLevel float64 `json:"water_level,omitempty"`
	CreatedBy  string  `json:"created_by"`
	SimName    string  `json:"sim_name"`
	SimID      int64   `json:"sim_id"`
}

type SimResultPayload struct {
	SimID   int64       `json:"sim_id"`
	SimType string      `json:"sim_type"`
	Result  interface{} `json:"result"`
	Error   string      `json:"error,omitempty"`
}

type PredictRequestPayload struct {
	StationID string `json:"station_id"`
	Years     int    `json:"years"`
}

type PredictResultPayload struct {
	StationID               string                      `json:"station_id"`
	Years                   int                         `json:"years"`
	Predictions             []interface{}               `json:"predictions"`
	AverageAnnualDeposition float64                     `json:"average_annual_deposition"`
	AverageAnnualErosion    float64                     `json:"average_annual_erosion"`
	FinalElevation          float64                     `json:"final_elevation"`
	RiskLevel               string                      `json:"risk_level"`
	ModelVersion            string                      `json:"model_version"`
	BaseElevation           float64                     `json:"base_elevation"`
}
