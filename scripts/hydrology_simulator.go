package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type StationConfig struct {
	StationID       string  `json:"station_id"`
	StationName     string  `json:"station_name"`
	BaseWaterLevel  float64 `json:"base_water_level"`
	BaseFlowRate    float64 `json:"base_flow_rate"`
	BaseSediment    float64 `json:"base_sediment"`
	BaseBedElevation float64 `json:"base_bed_elevation"`
	FlowVariation   float64 `json:"flow_variation"`
	SedimentTrend   float64 `json:"sediment_trend"`
}

type HydrologyData struct {
	Time                  time.Time `json:"time"`
	StationID             string    `json:"station_id"`
	StationName           string    `json:"station_name"`
	WaterLevel            float64   `json:"water_level"`
	FlowRate              float64   `json:"flow_rate"`
	SedimentConcentration float64   `json:"sediment_concentration"`
	BedElevation          float64   `json:"bed_elevation"`
	Temperature           float64   `json:"temperature"`
	Rainfall              float64   `json:"rainfall"`
	SensorStatus          int       `json:"sensor_status"`
}

type Simulator struct {
	apiBaseURL  string
	interval    time.Duration
	stations    []StationConfig
	bedState    map[string]float64
	accumulatedDeposition map[string]float64
	startTime   time.Time
	speedFactor float64
}

var stations = []StationConfig{
	{StationID: "NEIJ-001", StationName: "内江进水口", BaseWaterLevel: 728.5, BaseFlowRate: 250, BaseSediment: 0.8, BaseBedElevation: 726.5, FlowVariation: 100, SedimentTrend: 0.00002},
	{StationID: "NEIJ-002", StationName: "内江中段", BaseWaterLevel: 727.8, BaseFlowRate: 220, BaseSediment: 0.75, BaseBedElevation: 725.8, FlowVariation: 90, SedimentTrend: 0.000025},
	{StationID: "NEIJ-003", StationName: "宝瓶口上游", BaseWaterLevel: 728.2, BaseFlowRate: 180, BaseSediment: 0.65, BaseBedElevation: 726.2, FlowVariation: 70, SedimentTrend: 0.00003},
	{StationID: "WAIJ-001", StationName: "外江进水口", BaseWaterLevel: 728.0, BaseFlowRate: 350, BaseSediment: 1.0, BaseBedElevation: 726.0, FlowVariation: 150, SedimentTrend: 0.000018},
	{StationID: "WAIJ-002", StationName: "外江中段", BaseWaterLevel: 727.2, BaseFlowRate: 320, BaseSediment: 0.95, BaseBedElevation: 725.5, FlowVariation: 140, SedimentTrend: 0.000022},
	{StationID: "FSSY-001", StationName: "飞沙堰进口", BaseWaterLevel: 729.0, BaseFlowRate: 80, BaseSediment: 1.2, BaseBedElevation: 727.0, FlowVariation: 40, SedimentTrend: 0.000035},
	{StationID: "FSSY-002", StationName: "飞沙堰出口", BaseWaterLevel: 728.5, BaseFlowRate: 60, BaseSediment: 0.4, BaseBedElevation: 726.8, FlowVariation: 30, SedimentTrend: 0.00001},
	{StationID: "RJK-001", StationName: "人字堤", BaseWaterLevel: 727.5, BaseFlowRate: 40, BaseSediment: 0.5, BaseBedElevation: 726.3, FlowVariation: 20, SedimentTrend: 0.000015},
}

func NewSimulator(apiBaseURL string, interval time.Duration, speedFactor float64) *Simulator {
	bedState := make(map[string]float64)
	accumulatedDeposition := make(map[string]float64)

	for _, s := range stations {
		bedState[s.StationID] = s.BaseBedElevation
		accumulatedDeposition[s.StationID] = 0
	}

	return &Simulator{
		apiBaseURL:            apiBaseURL,
		interval:              interval,
		stations:              stations,
		bedState:              bedState,
		accumulatedDeposition: accumulatedDeposition,
		startTime:             time.Now(),
		speedFactor:           speedFactor,
	}
}

func (s *Simulator) generateData(station StationConfig, simTime time.Time) HydrologyData {
	hourOfDay := float64(simTime.Hour())
	dayOfYear := float64(simTime.YearDay())

	seasonalFactor := 1.0 + 0.5*math.Sin((dayOfYear-150)*2*math.Pi/365)
	dailyFactor := 1.0 + 0.1*math.Sin((hourOfDay-6)*2*math.Pi/24)

	flowRate := station.BaseFlowRate*seasonalFactor*dailyFactor +
		rand.NormFloat64()*station.FlowVariation*0.1
	if flowRate < 10 {
		flowRate = 10
	}

	waterLevel := station.BaseWaterLevel + (flowRate-station.BaseFlowRate)*0.003 +
		rand.NormFloat64()*0.1
	if waterLevel < station.BaseBedElevation+0.5 {
		waterLevel = station.BaseBedElevation + 0.5
	}

	stormFactor := 0.0
	rainfall := 0.0
	if rand.Float64() < 0.02 {
		rainfall = rand.Float64() * 50
		stormFactor = rainfall / 50 * 0.5
	}

	sedimentBase := station.BaseSediment * seasonalFactor * (1 + stormFactor)
	sedimentConcentration := sedimentBase + rand.NormFloat64()*0.2
	if sedimentConcentration < 0.01 {
		sedimentConcentration = 0.01
	}

	timeElapsed := simTime.Sub(s.startTime).Hours() * s.speedFactor
	deposition := station.SedimentTrend * timeElapsed * (sedimentConcentration / station.BaseSediment)
	erosion := 0.0
	if flowRate > station.BaseFlowRate*1.5 {
		erosion = (flowRate - station.BaseFlowRate*1.5) * 0.00001
	}

	s.accumulatedDeposition[station.StationID] += deposition - erosion

	newBedElevation := station.BaseBedElevation + s.accumulatedDeposition[station.StationID]

	bedrockLimit := station.BaseBedElevation - 0.5
	if newBedElevation < bedrockLimit {
		newBedElevation = bedrockLimit
	}

	maxLimit := 732.0
	if newBedElevation > maxLimit {
		newBedElevation = maxLimit
	}

	s.bedState[station.StationID] = newBedElevation

	temperature := 15 + 10*math.Sin((dayOfYear-150)*2*math.Pi/365) +
		-5*math.Cos((hourOfDay-14)*2*math.Pi/24) + rand.NormFloat64()*0.5

	sensorStatus := 1
	if rand.Float64() < 0.001 {
		sensorStatus = 0
	}

	return HydrologyData{
		Time:                  simTime,
		StationID:             station.StationID,
		StationName:           station.StationName,
		WaterLevel:            math.Round(waterLevel*1000) / 1000,
		FlowRate:              math.Round(flowRate*100) / 100,
		SedimentConcentration: math.Round(sedimentConcentration*10000) / 10000,
		BedElevation:          math.Round(newBedElevation*1000) / 1000,
		Temperature:           math.Round(temperature*100) / 100,
		Rainfall:              math.Round(rainfall*100) / 100,
		SensorStatus:          sensorStatus,
	}
}

func (s *Simulator) sendData(data HydrologyData) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/hydrology/data", s.apiBaseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status: %d", resp.StatusCode)
	}

	return nil
}

func (s *Simulator) sendHistoricalData(days int) error {
	log.Printf("Generating %d days of historical data...", days)

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	step := time.Hour
	totalSteps := days * 24
	processed := 0

	for simTime := startTime; simTime.Before(endTime); simTime = simTime.Add(step) {
		for _, station := range s.stations {
			data := s.generateData(station, simTime)
			if err := s.sendData(data); err != nil {
				log.Printf("Warning: Failed to send historical data for %s: %v", station.StationID, err)
			}
		}

		processed++
		if processed%24 == 0 {
			progress := float64(processed) / float64(totalSteps) * 100
			log.Printf("Historical data progress: %.1f%% (%d/%d hours)", progress, processed, totalSteps)
		}
	}

	log.Println("Historical data generation complete")
	return nil
}

func (s *Simulator) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	simTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Println("Simulator stopped")
			return
		case <-ticker.C:
			simTime = simTime.Add(time.Hour * time.Duration(s.speedFactor))

			totalSent := 0
			for _, station := range s.stations {
				data := s.generateData(station, simTime)

				if err := s.sendData(data); err != nil {
					log.Printf("Warning: Failed to send data for %s: %v", station.StationID, err)
				} else {
					totalSent++
				}
			}

			log.Printf("[%s] Sent %d/%d stations | Sim time: %s | Bed elevation range: %.3fm - %.3fm",
				time.Now().Format("15:04:05"),
				totalSent, len(s.stations),
				simTime.Format("2006-01-02 15:04:05"),
				s.getMinBedElevation(), s.getMaxBedElevation(),
			)
		}
	}
}

func (s *Simulator) getMinBedElevation() float64 {
	min := math.MaxFloat64
	for _, v := range s.bedState {
		if v < min {
			min = v
		}
	}
	return min
}

func (s *Simulator) getMaxBedElevation() float64 {
	max := -math.MaxFloat64
	for _, v := range s.bedState {
		if v > max {
			max = v
		}
	}
	return max
}

func main() {
	apiBaseURL := flag.String("api", "http://localhost:8080", "API server base URL")
	interval := flag.Duration("interval", 1*time.Second, "Simulation tick interval")
	speedFactor := flag.Float64("speed", 1.0, "Simulation speed factor (hours per tick)")
	historicalDays := flag.Int("historical", 0, "Generate N days of historical data first")
	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.Println("============================================")
	log.Println("都江堰水文模拟器启动")
	log.Println("============================================")
	log.Printf("API Server: %s", *apiBaseURL)
	log.Printf("Interval: %s", *interval)
	log.Printf("Speed Factor: %.1fx (%.1f hours per tick)", *speedFactor, *speedFactor)
	log.Printf("Historical Data: %d days", *historicalDays)
	log.Printf("Stations: %d", len(stations))
	log.Println("============================================")

	rand.Seed(time.Now().UnixNano())

	simulator := NewSimulator(*apiBaseURL, *interval, *speedFactor)

	if *historicalDays > 0 {
		if err := simulator.sendHistoricalData(*historicalDays); err != nil {
			log.Printf("Warning: Historical data generation had errors: %v", err)
		}
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

	log.Println("Starting real-time simulation...")
	simulator.Run(ctx)

	log.Println("Simulator exited")
}
