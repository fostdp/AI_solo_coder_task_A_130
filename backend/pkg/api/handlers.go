package api

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"dujiangyan-system/pkg/models"
	"dujiangyan-system/pkg/mqtt"
	"dujiangyan-system/pkg/simulation"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type APIHandler struct {
	ctx context.Context
}

func NewAPIHandler(ctx context.Context) *APIHandler {
	return &APIHandler{ctx: ctx}
}

func (h *APIHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	hydrology := api.Group("/hydrology")
	{
		hydrology.POST("/data", h.ReceiveHydrologyData)
		hydrology.GET("/data/:station_id", h.GetHydrologyData)
		hydrology.GET("/data/latest/:station_id", h.GetLatestHydrologyData)
		hydrology.GET("/data/all", h.GetAllLatestData)
		hydrology.GET("/stats/daily/:station_id", h.GetDailyStats)
		hydrology.GET("/stations", h.GetMonitoringStations)
	}

	api.GET("/wolong-iron", h.GetWolongIron)

	alerts := api.Group("/alerts")
	{
		alerts.GET("", h.GetAlerts)
		alerts.POST("/:id/acknowledge", h.AcknowledgeAlert)
		alerts.GET("/unpublished", h.GetUnpublishedAlerts)
	}

	prediction := api.Group("/prediction")
	{
		prediction.POST("/bed-evolution/:station_id", h.RunBedEvolutionPrediction)
		prediction.GET("/bed-evolution/:station_id", h.GetBedEvolutionPredictions)
	}

	simulation := api.Group("/simulation")
	{
		simulation.POST("/bamboo-cage", h.RunBambooCageSimulation)
		simulation.POST("/macha-interception", h.RunMachaInterceptionSimulation)
		simulation.GET("/list", h.GetSimulations)
		simulation.GET("/macha/:simulation_id", h.GetMachaSimulationData)
		simulation.GET("/bamboo-cage/:simulation_id", h.GetBambooCageSimulationData)
	}

	api.GET("/annual-repair-records", h.GetAnnualRepairRecords)
	api.GET("/dem-grid", h.GetDEMGrid)
	api.GET("/evolution-rate/:station_id", h.GetEvolutionRate)

	api.GET("/ws/realtime", h.RealTimeWebSocket)
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

func (h *APIHandler) ReceiveHydrologyData(c *gin.Context) {
	var req HydrologyDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	if err := models.InsertHydrologyData(h.ctx, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data: " + err.Error()})
		return
	}

	go func() {
		if err := mqtt.PublishHydrologyData(data); err != nil {
		}
		broadcastHydrologyData(data)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Data received successfully",
		"data":    data,
	})
}

func (h *APIHandler) GetHydrologyData(c *gin.Context) {
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
		limit, _ = strconv.Atoi(limitStr)
	}

	data, err := models.GetHydrologyData(h.ctx, stationID, startTime, endTime, limit)
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

func (h *APIHandler) GetLatestHydrologyData(c *gin.Context) {
	stationID := c.Param("station_id")

	data, err := models.GetLatestHydrologyData(h.ctx, stationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No data found"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *APIHandler) GetAllLatestData(c *gin.Context) {
	data, err := models.GetAllLatestHydrologyData(h.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(data),
		"data":  data,
	})
}

func (h *APIHandler) GetDailyStats(c *gin.Context) {
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

	stats, err := models.GetDailyStats(h.ctx, stationID, startTime, endTime)
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

func (h *APIHandler) GetMonitoringStations(c *gin.Context) {
	stations, err := models.GetMonitoringStations(h.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(stations),
		"data":  stations,
	})
}

func (h *APIHandler) GetWolongIron(c *gin.Context) {
	data, err := models.GetWolongIron(h.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(data),
		"data":  data,
	})
}

func (h *APIHandler) GetAlerts(c *gin.Context) {
	acknowledgedStr := c.Query("acknowledged")
	limitStr := c.Query("limit")

	var acknowledged *bool
	if acknowledgedStr != "" {
		ack, _ := strconv.ParseBool(acknowledgedStr)
		acknowledged = &ack
	}

	limit := 100
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	alerts, err := models.GetAlerts(h.ctx, acknowledged, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(alerts),
		"data":  alerts,
	})
}

func (h *APIHandler) AcknowledgeAlert(c *gin.Context) {
	alertID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := models.AcknowledgeAlert(h.ctx, alertID, req.AcknowledgedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged successfully"})
}

func (h *APIHandler) GetUnpublishedAlerts(c *gin.Context) {
	alerts, err := models.GetUnpublishedAlerts(h.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(alerts),
		"data":  alerts,
	})
}

type BedEvolutionRequest struct {
	Years int `json:"years"`
}

func (h *APIHandler) RunBedEvolutionPrediction(c *gin.Context) {
	stationID := c.Param("station_id")
	years, _ := strconv.Atoi(c.DefaultQuery("years", "10"))

	var req BedEvolutionRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.Years > 0 {
		years = req.Years
	}

	results, err := simulation.PredictBedEvolution(h.ctx, stationID, years)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var baseElevation float64 = 726.5
	latestData, err := models.GetLatestHydrologyData(h.ctx, stationID)
	if err == nil && latestData.BedElevation > 0 {
		baseElevation = latestData.BedElevation
	}

	monthlyPredictions := make([]map[string]interface{}, 0, years*12)
	var avgAnnualDeposition, avgAnnualErosion float64

	for yearIdx, annualResult := range results {
		for month := 0; month < 12; month++ {
			monthFraction := float64(yearIdx) + float64(month)/12.0
			seasonalFactor := math.Sin(monthFraction*2*math.Pi/1.0) * 0.02
			
			monthlyDeposition := annualResult.Deposition / 12.0 * (1 + seasonalFactor)
			monthlyErosion := annualResult.Erosion / 12.0 * (1 - seasonalFactor)
			elevationChange := annualResult.PredictedElevation - baseElevation
			elevationChange += (float64(month) / 12.0) * (annualResult.NetChange)

			predDate := time.Now().AddDate(yearIdx, month, 0)
			
			monthlyPredictions = append(monthlyPredictions, map[string]interface{}{
				"prediction_date":      predDate,
				"bed_elevation_change": elevationChange,
				"predicted_elevation":  annualResult.PredictedElevation,
				"erosion_rate":         monthlyErosion,
				"deposition_rate":      monthlyDeposition,
				"sediment_accumulation": elevationChange * 1000,
				"confidence":           annualResult.Confidence,
			})

			avgAnnualDeposition += monthlyDeposition
			avgAnnualErosion += monthlyErosion
		}
	}

	avgAnnualDeposition = avgAnnualDeposition / float64(years) * 12
	avgAnnualErosion = avgAnnualErosion / float64(years) * 12
	finalElevation := results[len(results)-1].PredictedElevation
	
	riskLevel := "低"
	elevationDiff := finalElevation - baseElevation
	if elevationDiff > 0.3 {
		riskLevel = "高"
	} else if elevationDiff > 0.15 {
		riskLevel = "中"
	}

	c.JSON(http.StatusOK, gin.H{
		"station_id":               stationID,
		"years":                    years,
		"model":                    "Sediment Transport Model v1.0",
		"base_elevation":           baseElevation,
		"predictions":              monthlyPredictions,
		"average_annual_deposition": avgAnnualDeposition,
		"average_annual_erosion":    avgAnnualErosion,
		"final_elevation":          finalElevation,
		"risk_level":               riskLevel,
		"annual_data":              results,
	})
}

func (h *APIHandler) GetBedEvolutionPredictions(c *gin.Context) {
	stationID := c.Param("station_id")

	results, err := models.GetBedEvolutionPredictions(h.ctx, stationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"station_id": stationID,
		"count":      len(results),
		"data":       results,
	})
}

type BambooCageSimulationRequest struct {
	Location   string `json:"location" binding:"required"`
	CageCount  int    `json:"cage_count" binding:"required,min=1,max=100"`
	SimName    string `json:"simulation_name"`
	CreatedBy  string `json:"created_by"`
}

func (h *APIHandler) RunBambooCageSimulation(c *gin.Context) {
	var req BambooCageSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SimName == "" {
		req.SimName = "竹笼装石仿真 - " + time.Now().Format("2006-01-02 15:04:05")
	}

	simRecord := &models.AnnualRepairSimulation{
		SimulationName: req.SimName,
		SimulationType: "bamboo_cage",
		Status:         "running",
		CreatedBy:      req.CreatedBy,
	}

	simID, err := models.InsertAnnualRepairSimulation(h.ctx, simRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cages, err := simulation.SimulateBambooCagePlacement(h.ctx, simID, req.Location, req.CageCount)
	if err != nil {
		models.UpdateSimulationStatus(h.ctx, simID, "failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := map[string]interface{}{
		"cages":      cages,
		"cage_count": len(cages),
	}
	models.UpdateSimulationResult(h.ctx, simID, result)

	c.JSON(http.StatusOK, gin.H{
		"simulation_id": simID,
		"location":      req.Location,
		"cage_count":    len(cages),
		"data":          cages,
	})
}

type MachaInterceptionRequest struct {
	Location          string  `json:"location" binding:"required"`
	MachaCount        int     `json:"macha_count" binding:"required,min=1,max=50"`
	InitialFlowRate   float64 `json:"initial_flow_rate" binding:"required,min=0"`
	InitialWaterLevel float64 `json:"initial_water_level" binding:"required,min=0"`
	SimName           string  `json:"simulation_name"`
	CreatedBy         string  `json:"created_by"`
}

func (h *APIHandler) RunMachaInterceptionSimulation(c *gin.Context) {
	var req MachaInterceptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SimName == "" {
		req.SimName = "杩槎截流仿真 - " + time.Now().Format("2006-01-02 15:04:05")
	}

	simRecord := &models.AnnualRepairSimulation{
		SimulationName: req.SimName,
		SimulationType: "macha_interception",
		Status:         "running",
		CreatedBy:      req.CreatedBy,
	}

	simID, err := models.InsertAnnualRepairSimulation(h.ctx, simRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	machas, interceptionData, err := simulation.SimulateMachaInterception(
		h.ctx, simID, req.Location, req.MachaCount,
		req.InitialFlowRate, req.InitialWaterLevel,
	)
	if err != nil {
		models.UpdateSimulationStatus(h.ctx, simID, "failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := map[string]interface{}{
		"machas":             machas,
		"interception_data":  interceptionData,
		"macha_count":        len(machas),
		"final_efficiency":   interceptionData[len(interceptionData)-1].InterceptionEfficiency,
		"final_flow_rate":    interceptionData[len(interceptionData)-1].FlowRateAfter,
	}
	models.UpdateSimulationResult(h.ctx, simID, result)

	c.JSON(http.StatusOK, gin.H{
		"simulation_id": simID,
		"location":      req.Location,
		"macha_count":   len(machas),
		"machas":        machas,
		"interception":  interceptionData,
	})
}

func (h *APIHandler) GetSimulations(c *gin.Context) {
	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	simulations, err := models.GetAnnualRepairSimulations(h.ctx, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(simulations),
		"data":  simulations,
	})
}

func (h *APIHandler) GetMachaSimulationData(c *gin.Context) {
	simID, err := strconv.Atoi(c.Param("simulation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid simulation ID"})
		return
	}

	data, err := models.GetMachaSimulationData(h.ctx, simID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"simulation_id": simID,
		"count":         len(data),
		"data":          data,
	})
}

func (h *APIHandler) GetBambooCageSimulationData(c *gin.Context) {
	simID, err := strconv.Atoi(c.Param("simulation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid simulation ID"})
		return
	}

	data, err := models.GetBambooCageSimulationData(h.ctx, simID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"simulation_id": simID,
		"count":         len(data),
		"data":          data,
	})
}

func (h *APIHandler) GetAnnualRepairRecords(c *gin.Context) {
	records, err := models.GetAnnualRepairRecords(h.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(records),
		"data":  records,
	})
}

func (h *APIHandler) GetDEMGrid(c *gin.Context) {
	centerX, _ := strconv.ParseFloat(c.DefaultQuery("center_x", "0"), 64)
	centerY, _ := strconv.ParseFloat(c.DefaultQuery("center_y", "0"), 64)
	gridSize, _ := strconv.ParseFloat(c.DefaultQuery("grid_size", "100"), 64)
	resolution, _ := strconv.ParseFloat(c.DefaultQuery("resolution", "5"), 64)
	baseElevation, _ := strconv.ParseFloat(c.DefaultQuery("base_elevation", "726.5"), 64)

	grid := simulation.GenerateDEMGrid(h.ctx, centerX, centerY, gridSize, resolution, baseElevation)

	c.JSON(http.StatusOK, gin.H{
		"grid_size":     gridSize,
		"resolution":    resolution,
		"base_elevation": baseElevation,
		"dimensions":    len(grid),
		"data":          grid,
	})
}

func (h *APIHandler) GetEvolutionRate(c *gin.Context) {
	stationID := c.Param("station_id")

	data, err := models.GetHydrologyData(
		h.ctx, stationID,
		time.Now().AddDate(0, -1, 0),
		time.Now(),
		8760,
	)
	if err != nil || len(data) < 2 {
		c.JSON(http.StatusOK, gin.H{
			"station_id": stationID,
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

var wsClients = make(map[*websocket.Conn]bool)
var wsBroadcast = make(chan interface{})

func (h *APIHandler) RealTimeWebSocket(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	wsClients[ws] = true

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			delete(wsClients, ws)
			break
		}
	}
}

func broadcastHydrologyData(data *models.HydrologyData) {
	msg := map[string]interface{}{
		"type": "hydrology",
		"data": data,
	}
	wsBroadcast <- msg
}

func broadcastAlert(alert *models.Alert) {
	msg := map[string]interface{}{
		"type": "alert",
		"data": alert,
	}
	wsBroadcast <- msg
}

func StartWebSocketBroadcaster(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-wsBroadcast:
				for client := range wsClients {
					payload, _ := json.Marshal(msg)
					err := client.WriteMessage(websocket.TextMessage, payload)
					if err != nil {
						client.Close()
						delete(wsClients, client)
					}
				}
			}
		}
	}()
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func SetupStaticFiles(r *gin.Engine) {
	frontendPath := os.Getenv("FRONTEND_PATH")
	if frontendPath == "" {
		frontendPath = "./frontend"
	}
	r.Static("/frontend", frontendPath)
	r.GET("/", func(c *gin.Context) {
		c.File(frontendPath + "/index.html")
	})
}
