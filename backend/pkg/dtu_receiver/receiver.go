package dtu_receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"dujiangyan-system/pkg/metrics"
	"dujiangyan-system/pkg/models"
	"dujiangyan-system/pkg/msg"
	"dujiangyan-system/pkg/simulation"
)

type DTUReceiver struct {
	ctx       context.Context
	bus       chan<- msg.BusMessage
	validIDs  map[string]bool
	rateLimit map[string]time.Time
}

type HydrologyDataRequest struct {
	StationID             string    `json:"station_id" binding:"required"`
	StationName           string    `json:"station_name"`
	WaterLevel            float64   `json:"water_level" binding:"required"`
	FlowRate              float64   `json:"flow_rate" binding:"required"`
	SedimentConcentration float64   `json:"sediment_concentration" binding:"required"`
	BedElevation          float64   `json:"bed_elevation" binding:"required"`
	Temperature           float64   `json:"temperature"`
	Rainfall              float64   `json:"rainfall"`
	SensorStatus          int       `json:"sensor_status"`
	Time                  time.Time `json:"time"`
}

func NewDTUReceiver(ctx context.Context, bus chan<- msg.BusMessage) *DTUReceiver {
	r := &DTUReceiver{
		ctx:       ctx,
		bus:       bus,
		validIDs:  make(map[string]bool),
		rateLimit: make(map[string]time.Time),
	}

	stationIDs := []string{
		"NEIJ-001", "NEIJ-002", "NEIJ-003",
		"WAIJ-001", "WAIJ-002",
		"FSSY-001", "FSSY-002",
		"RJK-001",
	}
	for _, id := range stationIDs {
		r.validIDs[id] = true
	}

	return r
}

func (r *DTUReceiver) ValidateStationID(stationID string) error {
	if !r.validIDs[stationID] {
		return fmt.Errorf("unknown station_id: %s", stationID)
	}
	return nil
}

func (r *DTUReceiver) CheckRateLimit(stationID string) error {
	lastTime, exists := r.rateLimit[stationID]
	if exists && time.Since(lastTime) < 10*time.Second {
		return fmt.Errorf("rate limit exceeded for station %s", stationID)
	}
	r.rateLimit[stationID] = time.Now()
	return nil
}

func (r *DTUReceiver) ValidateRange(data *models.HydrologyData) error {
	if data.WaterLevel < 700 || data.WaterLevel > 750 {
		return fmt.Errorf("water_level %.3f out of valid range [700, 750]", data.WaterLevel)
	}
	if data.FlowRate < 0 || data.FlowRate > 2000 {
		return fmt.Errorf("flow_rate %.2f out of valid range [0, 2000]", data.FlowRate)
	}
	if data.SedimentConcentration < 0 || data.SedimentConcentration > 50 {
		return fmt.Errorf("sediment_concentration %.4f out of valid range [0, 50]", data.SedimentConcentration)
	}
	if data.BedElevation < 720 || data.BedElevation > 740 {
		return fmt.Errorf("bed_elevation %.3f out of valid range [720, 740]", data.BedElevation)
	}
	if data.Temperature < -20 || data.Temperature > 50 {
		return fmt.Errorf("temperature %.2f out of valid range [-20, 50]", data.Temperature)
	}
	if math.IsNaN(data.WaterLevel) || math.IsInf(data.WaterLevel, 0) {
		return fmt.Errorf("water_level contains NaN or Inf")
	}
	if math.IsNaN(data.FlowRate) || math.IsInf(data.FlowRate, 0) {
		return fmt.Errorf("flow_rate contains NaN or Inf")
	}
	return nil
}

func (r *DTUReceiver) HandleReceive(c *gin.Context) {
	var req HydrologyDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		metrics.GetMetrics().HydrologyDataErrors.WithLabelValues("unknown", "parse_error").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m := metrics.GetMetrics()

	if err := r.ValidateStationID(req.StationID); err != nil {
		m.HydrologyDataErrors.WithLabelValues(req.StationID, "invalid_station").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.CheckRateLimit(req.StationID); err != nil {
		m.HydrologyDataErrors.WithLabelValues(req.StationID, "rate_limit").Inc()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	if req.Time.IsZero() {
		req.Time = time.Now()
	}

	data := &models.HydrologyData{
		Time:                  req.Time,
		StationID:             req.StationID,
		StationName:           req.StationName,
		WaterLevel:            req.WaterLevel,
		FlowRate:              req.FlowRate,
		SedimentConcentration: req.SedimentConcentration,
		BedElevation:          req.BedElevation,
		Temperature:           req.Temperature,
		Rainfall:              req.Rainfall,
		SensorStatus:          req.SensorStatus,
	}

	if data.SensorStatus == 0 {
		data.SensorStatus = 1
	}

	if err := r.ValidateRange(data); err != nil {
		m.HydrologyDataErrors.WithLabelValues(req.StationID, "out_of_range").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	if err := models.InsertHydrologyData(r.ctx, data); err != nil {
		m.HydrologyDataErrors.WithLabelValues(req.StationID, "db_error").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data: " + err.Error()})
		return
	}

	m.HydrologyDataReceived.WithLabelValues(req.StationID, "water_level").Inc()
	m.HydrologyDataReceived.WithLabelValues(req.StationID, "flow_rate").Inc()
	m.HydrologyDataReceived.WithLabelValues(req.StationID, "sediment").Inc()
	m.HydrologyDataReceived.WithLabelValues(req.StationID, "bed_elevation").Inc()
	m.LatestWaterLevel.WithLabelValues(req.StationID).Set(data.WaterLevel)
	m.LatestSedimentConcentration.WithLabelValues(req.StationID).Set(data.SedimentConcentration)
	m.LatestFlowRate.WithLabelValues(req.StationID).Set(data.FlowRate)
	m.LatestBedElevation.WithLabelValues(req.StationID).Set(data.BedElevation)

	r.bus <- msg.BusMessage{
		Type:      msg.TypeHydrologyData,
		Timestamp: time.Now(),
		Payload: msg.HydrologyPayload{Data: data},
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data received successfully",
		"data":    data,
	})
}

func (r *DTUReceiver) HandleGetHistory(c *gin.Context) {
	stationID := c.Param("station_id")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	limitStr := c.Query("limit")

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			startTime = time.Now().AddDate(0, -1, 0)
		}
	} else {
		startTime = time.Now().AddDate(0, -1, 0)
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			endTime = time.Now()
		}
	} else {
		endTime = time.Now()
	}

	limit := 1000
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	data, err := models.GetHydrologyData(r.ctx, stationID, startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"station_id": stationID,
		"count":      len(data),
		"data":       data,
	})
}

func (r *DTUReceiver) HandleGetLatest(c *gin.Context) {
	stationID := c.Param("station_id")
	data, err := models.GetLatestHydrologyData(r.ctx, stationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No data found"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (r *DTUReceiver) HandleGetAllLatest(c *gin.Context) {
	data, err := models.GetAllLatestHydrologyData(r.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": len(data),
		"data":  data,
	})
}

func (r *DTUReceiver) HandleGetDailyStats(c *gin.Context) {
	stationID := c.Param("station_id")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	startTime := time.Now().AddDate(0, -1, 0)
	endTime := time.Now()
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			startTime = time.Now().AddDate(0, -1, 0)
		}
	}
	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			endTime = time.Now()
		}
	}

	stats, err := models.GetDailyStats(r.ctx, stationID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"station_id": stationID,
		"count":      len(stats),
		"data":       stats,
	})
}

func (r *DTUReceiver) HandleGetStations(c *gin.Context) {
	stations, err := models.GetMonitoringStations(r.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": len(stations),
		"data":  stations,
	})
}

func (r *DTUReceiver) HandleGetWolongIron(c *gin.Context) {
	data, err := models.GetWolongIron(r.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": len(data),
		"data":  data,
	})
}

func (r *DTUReceiver) HandleGetDEMGrid(c *gin.Context) {
	centerX := 0.0
	centerY := 0.0
	gridSize := 100.0
	resolution := 5.0
	baseElevation := 726.5

	fmt.Sscanf(c.DefaultQuery("center_x", "0"), "%f", &centerX)
	fmt.Sscanf(c.DefaultQuery("center_y", "0"), "%f", &centerY)
	fmt.Sscanf(c.DefaultQuery("grid_size", "100"), "%f", &gridSize)
	fmt.Sscanf(c.DefaultQuery("resolution", "5"), "%f", &resolution)
	fmt.Sscanf(c.DefaultQuery("base_elevation", "726.5"), "%f", &baseElevation)

	grid := simulation.GenerateDEMGrid(r.ctx, centerX, centerY, gridSize, resolution, baseElevation)

	c.JSON(http.StatusOK, gin.H{
		"grid_size":      gridSize,
		"resolution":     resolution,
		"base_elevation": baseElevation,
		"dimensions":     len(grid),
		"data":           grid,
	})
}

func (r *DTUReceiver) HandleGetEvolutionRate(c *gin.Context) {
	stationID := c.Param("station_id")

	data, err := models.GetHydrologyData(
		r.ctx, stationID,
		time.Now().AddDate(0, -1, 0),
		time.Now(),
		8760,
	)
	if err != nil || len(data) < 2 {
		c.JSON(http.StatusOK, gin.H{
			"station_id":        stationID,
			"annual_deposition": 0,
			"annual_erosion":    0,
			"annual_net_change": 0,
			"has_data":          false,
		})
		return
	}

	deposition, erosion, netChange := simulation.CalculateEvolutionRate(data)

	c.JSON(http.StatusOK, gin.H{
		"station_id":         stationID,
		"annual_deposition":  deposition,
		"annual_erosion":     erosion,
		"annual_net_change":  netChange,
		"has_data":           true,
		"data_points":        len(data),
	})
}

func (r *DTUReceiver) HandleGetAnnualRepairRecords(c *gin.Context) {
	records, err := models.GetAnnualRepairRecords(r.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": len(records),
		"data":  records,
	})
}

func LoadCraftConfig(path string) (map[string]interface{}, error) {
	if path == "" {
		path = "config/craft_params.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: craft config not found at %s: %v", path, err)
		return nil, err
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid craft config JSON: %w", err)
	}
	return config, nil
}

func LoadSedimentConfig(path string) (map[string]interface{}, error) {
	if path == "" {
		path = "config/sediment_params.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: sediment config not found at %s: %v", path, err)
		return nil, err
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid sediment config JSON: %w", err)
	}
	return config, nil
}
