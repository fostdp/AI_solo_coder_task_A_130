package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"

	"dujiangyan-system/pkg/alarm_mqtt"
	"dujiangyan-system/pkg/dtu_receiver"
	"dujiangyan-system/pkg/maintenance_simulator"
	"dujiangyan-system/pkg/models"
	"dujiangyan-system/pkg/msg"
	"dujiangyan-system/pkg/riverbed_analyzer"
)

type Application struct {
	ctx              context.Context
	bus              chan msg.BusMessage
	alarmBus         chan msg.BusMessage
	dtu              *dtu_receiver.DTUReceiver
	simulator        *maintenance_simulator.MaintenanceSimulator
	analyzer         *riverbed_analyzer.RiverbedAnalyzer
	alarmService     *alarm_mqtt.AlarmService
	craftConfig      *maintenance_simulator.CraftConfig
	sedimentConfig   *riverbed_analyzer.SedimentConfig
	wsClients        map[*websocket.Conn]bool
	wsMu             sync.RWMutex
	wsBroadcast      chan interface{}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	if err := models.InitDB(); err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		log.Println("Continuing without database connection...")
	}
	defer models.CloseDB()

	bus := make(chan msg.BusMessage, 256)
	alarmBus := make(chan msg.BusMessage, 64)

	craftConfig, err := maintenance_simulator.LoadCraftConfig(os.Getenv("CRAFT_CONFIG_PATH"))
	if err != nil {
		log.Printf("Warning: Failed to load craft config: %v, using defaults", err)
		craftConfig = maintenance_simulator.DefaultCraftConfig()
	}

	sedimentConfig, err := riverbed_analyzer.LoadSedimentConfig(os.Getenv("SEDIMENT_CONFIG_PATH"))
	if err != nil {
		log.Printf("Warning: Failed to load sediment config: %v, using defaults", err)
		sedimentConfig = riverbed_analyzer.DefaultSedimentConfig()
	}

	dtu := dtu_receiver.NewDTUReceiver(ctx, bus)
	sim := maintenance_simulator.NewMaintenanceSimulator(ctx, bus, craftConfig)
	analyzer := riverbed_analyzer.NewRiverbedAnalyzer(ctx, bus, sedimentConfig)

	mqttBroker := os.Getenv("MQTT_BROKER")
	alarmService := alarm_mqtt.NewAlarmService(ctx, alarmBus, mqttBroker)

	app := &Application{
		ctx:            ctx,
		bus:            bus,
		alarmBus:       alarmBus,
		dtu:            dtu,
		simulator:      sim,
		analyzer:       analyzer,
		alarmService:   alarmService,
		craftConfig:    craftConfig,
		sedimentConfig: sedimentConfig,
		wsClients:      make(map[*websocket.Conn]bool),
		wsBroadcast:    make(chan interface{}, 64),
	}

	alarmService.Start()
	go app.startBusDispatcher()
	go app.startWebSocketBroadcaster()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(app.corsMiddleware())

	app.registerRoutes(r)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	addr := host + ":" + port

	server := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("Server starting on %s", addr)
		log.Printf("API: http://%s/api/v1", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

func (app *Application) startBusDispatcher() {
	for {
		select {
		case <-app.ctx.Done():
			return
		case message := <-app.bus:
			switch message.Type {
			case msg.TypeHydrologyData:
				select {
				case app.alarmBus <- message:
				default:
					log.Printf("Alarm bus full, dropping hydrology message")
				}
				app.wsBroadcast <- message
			case msg.TypeSimResult:
				app.wsBroadcast <- message
			case msg.TypePredictResult:
				app.wsBroadcast <- message
			}
		}
	}
}

func (app *Application) startWebSocketBroadcaster() {
	for {
		select {
		case <-app.ctx.Done():
			return
		case message := <-app.wsBroadcast:
			app.wsMu.RLock()
			for client := range app.wsClients {
				if err := client.WriteJSON(message); err != nil {
					go func(c *websocket.Conn) {
						app.wsMu.Lock()
						delete(app.wsClients, c)
						app.wsMu.Unlock()
						c.Close()
					}(client)
				}
			}
			app.wsMu.RUnlock()
		}
	}
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (app *Application) handleWebSocket(c *gin.Context) {
	ws, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	app.wsMu.Lock()
	app.wsClients[ws] = true
	app.wsMu.Unlock()

	defer func() {
		app.wsMu.Lock()
		delete(app.wsClients, ws)
		app.wsMu.Unlock()
		ws.Close()
	}()

	for {
		select {
		case <-app.ctx.Done():
			return
		default:
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}
}

func (app *Application) registerRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.POST("/hydrology/data", app.dtu.HandleReceive)
		v1.GET("/hydrology/:station_id/history", app.dtu.HandleGetHistory)
		v1.GET("/hydrology/:station_id/latest", app.dtu.HandleGetLatest)
		v1.GET("/hydrology/latest", app.dtu.HandleGetAllLatest)
		v1.GET("/hydrology/:station_id/daily-stats", app.dtu.HandleGetDailyStats)
		v1.GET("/hydrology/:station_id/evolution-rate", app.dtu.HandleGetEvolutionRate)

		v1.GET("/stations", app.dtu.HandleGetStations)
		v1.GET("/wolong-iron", app.dtu.HandleGetWolongIron)
		v1.GET("/dem-grid", app.dtu.HandleGetDEMGrid)
		v1.GET("/annual-repair/records", app.dtu.HandleGetAnnualRepairRecords)

		v1.POST("/simulation/bamboo-cage", app.handleBambooCageSimulation)
		v1.POST("/simulation/macha-interception", app.handleMachaInterceptionSimulation)

		v1.POST("/prediction/bed-evolution", app.handleBedEvolutionPrediction)

		v1.GET("/alerts", app.handleGetAlerts)
		v1.PUT("/alerts/:id/acknowledge", app.handleAcknowledgeAlert)

		v1.GET("/ws", app.handleWebSocket)
	}
}

func (app *Application) handleBambooCageSimulation(c *gin.Context) {
	var req struct {
		Location   string `json:"location" binding:"required"`
		CageCount  int    `json:"cage_count" binding:"required,min=1,max=50"`
		SimName    string `json:"sim_name"`
		CreatedBy  string `json:"created_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	endTime := now.Add(24 * time.Hour)
	simRecord := &models.AnnualRepairSimulation{
		SimulationName: req.SimName,
		SimulationType: "bamboo_cage",
		StartTime:      &now,
		EndTime:        &endTime,
		Status:         "running",
		CreatedBy:      req.CreatedBy,
	}
	if err := models.InsertAnnualRepairSimulation(app.ctx, simRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create simulation record"})
		return
	}

	simReq := msg.SimRequestPayload{
		SimType:   "bamboo_cage",
		Location:  req.Location,
		CageCount: req.CageCount,
		CreatedBy: req.CreatedBy,
		SimName:   req.SimName,
		SimID:     int64(simRecord.ID),
	}

	cages, err := app.simulator.RunBambooCageSimulation(simReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"simulation_id": simRecord.ID,
		"cage_count":    len(cages),
		"cages":         cages,
		"status":        "completed",
	})
}

func (app *Application) handleMachaInterceptionSimulation(c *gin.Context) {
	var req struct {
		Location   string  `json:"location" binding:"required"`
		MachaCount int     `json:"macha_count" binding:"required,min=1,max=30"`
		FlowRate   float64 `json:"flow_rate" binding:"required"`
		WaterLevel float64 `json:"water_level" binding:"required"`
		SimName    string  `json:"sim_name"`
		CreatedBy  string  `json:"created_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	endTime := now.Add(48 * time.Hour)
	simRecord := &models.AnnualRepairSimulation{
		SimulationName: req.SimName,
		SimulationType: "macha_interception",
		StartTime:      &now,
		EndTime:        &endTime,
		Status:         "running",
		CreatedBy:      req.CreatedBy,
	}
	if err := models.InsertAnnualRepairSimulation(app.ctx, simRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create simulation record"})
		return
	}

	simReq := msg.SimRequestPayload{
		SimType:    "macha_interception",
		Location:   req.Location,
		MachaCount: req.MachaCount,
		FlowRate:   req.FlowRate,
		WaterLevel: req.WaterLevel,
		CreatedBy:  req.CreatedBy,
		SimName:    req.SimName,
		SimID:      int64(simRecord.ID),
	}

	machas, interceptionData, err := app.simulator.RunMachaInterception(simReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"simulation_id":      simRecord.ID,
		"macha_count":        len(machas),
		"machas":             machas,
		"interception_data":  interceptionData,
		"status":             "completed",
	})
}

func (app *Application) handleBedEvolutionPrediction(c *gin.Context) {
	var req struct {
		StationID string `json:"station_id" binding:"required"`
		Years     int    `json:"years" binding:"required,min=1,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := app.analyzer.PredictBedEvolution(req.StationID, req.Years)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{"predictions": []interface{}{}})
		return
	}

	monthlyPredictions := make([]map[string]interface{}, 0, req.Years*12)
	for _, yr := range results {
		for m := 0; m < 12; m++ {
			fraction := float64(m) / 12.0
			var nextDepo, nextErosion, nextNet float64
			if yr.Year < req.Years {
				nextYr := results[yr.Year]
				nextDepo = yr.Deposition + (nextYr.Deposition-yr.Deposition)*fraction
				nextErosion = yr.Erosion + (nextYr.Erosion-yr.Erosion)*fraction
				nextNet = yr.NetChange + (nextYr.NetChange-yr.NetChange)*fraction
			} else {
				nextDepo = yr.Deposition
				nextErosion = yr.Erosion
				nextNet = yr.NetChange
			}

			predDate := time.Date(time.Now().Year()+yr.Year, time.Month(m+1), 1, 0, 0, 0, 0, time.Local)
			monthlyPredictions = append(monthlyPredictions, map[string]interface{}{
				"prediction_date":       predDate.Format(time.RFC3339),
				"bed_elevation_change":  nextNet / 12.0,
				"predicted_elevation":   yr.PredictedElevation + nextNet/12.0*float64(m),
				"erosion_rate":          nextErosion / 12.0,
				"deposition_rate":       nextDepo / 12.0,
				"sediment_accumulation": nextNet * 1000 / 12.0,
				"confidence":           yr.Confidence,
			})
		}
	}

	avgDepo := 0.0
	avgErosion := 0.0
	for _, r := range results {
		avgDepo += r.Deposition
		avgErosion += r.Erosion
	}
	if len(results) > 0 {
		avgDepo /= float64(len(results))
		avgErosion /= float64(len(results))
	}

	finalElev := results[len(results)-1].PredictedElevation
	riskLevel := "低"
	if finalElev > 728.0 {
		riskLevel = "高"
	} else if finalElev > 727.0 {
		riskLevel = "中"
	}

	c.JSON(http.StatusOK, gin.H{
		"station_id":                req.StationID,
		"years":                     req.Years,
		"predictions":               monthlyPredictions,
		"average_annual_deposition": avgDepo,
		"average_annual_erosion":    avgErosion,
		"final_elevation":           finalElev,
		"risk_level":                riskLevel,
		"model_version":             "v2.0-weir-boundary",
		"base_elevation":            results[0].PredictedElevation - results[0].NetChange,
	})
}

func (app *Application) handleGetAlerts(c *gin.Context) {
	alerts, err := models.GetAlerts(app.ctx, nil, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(alerts), "data": alerts})
}

func (app *Application) handleAcknowledgeAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}
	if err := app.alarmService.HandleAcknowledgeAlert(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged"})
}

func (app *Application) corsMiddleware() gin.HandlerFunc {
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
