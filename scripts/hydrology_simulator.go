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
	StationID        string  `json:"station_id"`
	StationName      string  `json:"station_name"`
	BaseWaterLevel   float64 `json:"base_water_level"`
	BaseFlowRate     float64 `json:"base_flow_rate"`
	BaseSediment     float64 `json:"base_sediment"`
	BaseBedElevation float64 `json:"base_bed_elevation"`
	FlowVariation    float64 `json:"flow_variation"`
	SedimentTrend    float64 `json:"sediment_trend"`
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

type ScenarioPreset struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	WaterLevelBonus    float64 `json:"water_level_bonus"`
	FlowRateMultiplier float64 `json:"flow_rate_multiplier"`
	SedimentBonus      float64 `json:"sediment_bonus"`
	SedimentMultiplier float64 `json:"sediment_multiplier"`
	RainfallBonus      float64 `json:"rainfall_bonus"`
	BedElevationStart  float64 `json:"bed_elevation_start"`
	DepositionRate     float64 `json:"deposition_rate"`
	StormFrequency     float64 `json:"storm_frequency"`
}

var scenarioPresets = map[string]ScenarioPreset{
	"normal": {
		Name:               "normal",
		Description:        "平水期 - 正常水文条件",
		WaterLevelBonus:    0.0,
		FlowRateMultiplier: 1.0,
		SedimentBonus:      0.0,
		SedimentMultiplier: 1.0,
		RainfallBonus:      0.0,
		BedElevationStart:  0.0,
		DepositionRate:     1.0,
		StormFrequency:     0.02,
	},
	"flood": {
		Name:               "flood",
		Description:        "丰水期/洪水 - 高水位、大流量、高含沙量",
		WaterLevelBonus:    3.0,
		FlowRateMultiplier: 2.0,
		SedimentBonus:      1.5,
		SedimentMultiplier: 2.5,
		RainfallBonus:      30.0,
		BedElevationStart:  0.5,
		DepositionRate:     3.0,
		StormFrequency:     0.15,
	},
	"drought": {
		Name:               "drought",
		Description:        "枯水期 - 低水位、小流量、低含沙量",
		WaterLevelBonus:    -2.5,
		FlowRateMultiplier: 0.4,
		SedimentBonus:      -0.3,
		SedimentMultiplier: 0.5,
		RainfallBonus:      -5.0,
		BedElevationStart:  0.0,
		DepositionRate:     0.3,
		StormFrequency:     0.005,
	},
	"maintenance": {
		Name:               "maintenance",
		Description:        "岁修期 - 截流后低水位、河床冲刷",
		WaterLevelBonus:    -4.0,
		FlowRateMultiplier: 0.15,
		SedimentBonus:      -0.5,
		SedimentMultiplier: 0.3,
		RainfallBonus:      0.0,
		BedElevationStart:  1.0,
		DepositionRate:     -0.5,
		StormFrequency:     0.01,
	},
	"high_sediment": {
		Name:               "high_sediment",
		Description:        "高含沙期 - 模拟汛期泥沙大量下泄",
		WaterLevelBonus:    1.0,
		FlowRateMultiplier: 1.2,
		SedimentBonus:      3.0,
		SedimentMultiplier: 4.0,
		RainfallBonus:      10.0,
		BedElevationStart:  0.8,
		DepositionRate:     5.0,
		StormFrequency:     0.08,
	},
	"erosion": {
		Name:               "erosion",
		Description:        "冲刷期 - 大流量低含沙，河床下切",
		WaterLevelBonus:    2.0,
		FlowRateMultiplier: 1.8,
		SedimentBonus:      -0.2,
		SedimentMultiplier: 0.8,
		RainfallBonus:      20.0,
		BedElevationStart:  0.0,
		DepositionRate:     -2.0,
		StormFrequency:     0.1,
	},
}

type Simulator struct {
	apiBaseURL   string
	interval     time.Duration
	stations     []StationConfig
	bedState     map[string]float64
	accumulated  map[string]float64
	startTime    time.Time
	speedFactor  float64
	scenario     ScenarioPreset
	waterBonus   float64
	sedimentBonus float64
	flowMultiplier float64
}

var baseStations = []StationConfig{
	{StationID: "NEIJ-001", StationName: "内江进水口", BaseWaterLevel: 728.5, BaseFlowRate: 250, BaseSediment: 0.8, BaseBedElevation: 726.5, FlowVariation: 100, SedimentTrend: 0.00002},
	{StationID: "NEIJ-002", StationName: "内江中段", BaseWaterLevel: 727.8, BaseFlowRate: 220, BaseSediment: 0.75, BaseBedElevation: 725.8, FlowVariation: 90, SedimentTrend: 0.000025},
	{StationID: "NEIJ-003", StationName: "宝瓶口上游", BaseWaterLevel: 728.2, BaseFlowRate: 180, BaseSediment: 0.65, BaseBedElevation: 726.2, FlowVariation: 70, SedimentTrend: 0.00003},
	{StationID: "WAIJ-001", StationName: "外江进水口", BaseWaterLevel: 728.0, BaseFlowRate: 350, BaseSediment: 1.0, BaseBedElevation: 726.0, FlowVariation: 150, SedimentTrend: 0.000018},
	{StationID: "WAIJ-002", StationName: "外江中段", BaseWaterLevel: 727.2, BaseFlowRate: 320, BaseSediment: 0.95, BaseBedElevation: 725.5, FlowVariation: 140, SedimentTrend: 0.000022},
	{StationID: "FSSY-001", StationName: "飞沙堰进口", BaseWaterLevel: 729.0, BaseFlowRate: 80, BaseSediment: 1.2, BaseBedElevation: 727.0, FlowVariation: 40, SedimentTrend: 0.000035},
	{StationID: "FSSY-002", StationName: "飞沙堰出口", BaseWaterLevel: 728.5, BaseFlowRate: 60, BaseSediment: 0.4, BaseBedElevation: 726.8, FlowVariation: 30, SedimentTrend: 0.00001},
	{StationID: "RJK-001", StationName: "人字堤", BaseWaterLevel: 727.5, BaseFlowRate: 40, BaseSediment: 0.5, BaseBedElevation: 726.3, FlowVariation: 20, SedimentTrend: 0.000015},
}

func listScenarios() {
	fmt.Println("Available scenarios:")
	fmt.Println("====================")
	for name, s := range scenarioPresets {
		fmt.Printf("  %-15s %s\n", name, s.Description)
	}
	fmt.Println("")
}

func NewSimulator(apiBaseURL string, interval time.Duration, speedFactor float64,
	scenarioName string, waterBonus, sedimentBonus, flowMultiplier float64) *Simulator {

	scenario, ok := scenarioPresets[scenarioName]
	if !ok {
		log.Printf("Warning: Unknown scenario '%s', using 'normal'", scenarioName)
		scenario = scenarioPresets["normal"]
	}

	if waterBonus != 0 {
		scenario.WaterLevelBonus = waterBonus
	}
	if sedimentBonus != 0 {
		scenario.SedimentBonus = sedimentBonus
	}
	if flowMultiplier != 0 && flowMultiplier != 1.0 {
		scenario.FlowRateMultiplier = flowMultiplier
	}

	bedState := make(map[string]float64)
	accumulated := make(map[string]float64)

	for _, s := range baseStations {
		startElev := s.BaseBedElevation + scenario.BedElevationStart
		bedState[s.StationID] = startElev
		accumulated[s.StationID] = scenario.BedElevationStart
	}

	return &Simulator{
		apiBaseURL:     apiBaseURL,
		interval:       interval,
		stations:       baseStations,
		bedState:       bedState,
		accumulated:    accumulated,
		startTime:      time.Now(),
		speedFactor:    speedFactor,
		scenario:       scenario,
		waterBonus:     waterBonus,
		sedimentBonus:  sedimentBonus,
		flowMultiplier: flowMultiplier,
	}
}

func (s *Simulator) generateData(station StationConfig, simTime time.Time) HydrologyData {
	hourOfDay := float64(simTime.Hour())
	dayOfYear := float64(simTime.YearDay())

	sc := s.scenario

	seasonalFactor := 1.0 + 0.5*math.Sin((dayOfYear-150)*2*math.Pi/365)
	dailyFactor := 1.0 + 0.1*math.Sin((hourOfDay-6)*2*math.Pi/24)

	flowRate := station.BaseFlowRate*sc.FlowRateMultiplier*seasonalFactor*dailyFactor +
		rand.NormFloat64()*station.FlowVariation*0.1
	if flowRate < 5 {
		flowRate = 5
	}

	waterLevel := station.BaseWaterLevel + sc.WaterLevelBonus +
		(flowRate-station.BaseFlowRate*sc.FlowRateMultiplier)*0.003 +
		rand.NormFloat64()*0.1
	if waterLevel < station.BaseBedElevation+0.3 {
		waterLevel = station.BaseBedElevation + 0.3
	}

	stormFactor := 0.0
	rainfall := 0.0
	if rand.Float64() < sc.StormFrequency {
		rainfall = (rand.Float64()*50 + sc.RainfallBonus)
		if rainfall < 0 {
			rainfall = 0
		}
		stormFactor = rainfall / 50 * 0.5
	}

	sedimentBase := (station.BaseSediment + sc.SedimentBonus) * sc.SedimentMultiplier * seasonalFactor * (1 + stormFactor)
	sedimentConcentration := sedimentBase + rand.NormFloat64()*0.2
	if sedimentConcentration < 0.01 {
		sedimentConcentration = 0.01
	}

	timeElapsed := simTime.Sub(s.startTime).Hours() * s.speedFactor
	netDepositionRate := station.SedimentTrend * sc.DepositionRate
	deposition := netDepositionRate * timeElapsed * (sedimentConcentration / station.BaseSediment)
	erosion := 0.0
	if flowRate > station.BaseFlowRate*sc.FlowRateMultiplier*1.5 {
		erosion = (flowRate - station.BaseFlowRate*sc.FlowRateMultiplier*1.5) * 0.00001
	}

	s.accumulated[station.StationID] += deposition - erosion

	newBedElevation := station.BaseBedElevation + s.accumulated[station.StationID]

	bedrockLimit := station.BaseBedElevation - 1.0
	if newBedElevation < bedrockLimit {
		newBedElevation = bedrockLimit
	}

	maxLimit := 733.0
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

	url := fmt.Sprintf("%s/hydrology/data", s.apiBaseURL)
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

			log.Printf("[%s] Sent %d/%d | Scenario: %s | Sim: %s | WL: %.2fm | Sed: %.3f | Bed: %.3f",
				time.Now().Format("15:04:05"),
				totalSent, len(s.stations),
				s.scenario.Name,
				simTime.Format("2006-01-02 15:04"),
				s.getAvgWaterLevel(),
				s.getAvgSediment(),
				s.getAvgBedElevation(),
			)
		}
	}
}

func (s *Simulator) getAvgWaterLevel() float64 {
	sum := 0.0
	count := 0
	for _, st := range s.stations {
		elev := st.BaseWaterLevel + s.scenario.WaterLevelBonus
		sum += elev
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (s *Simulator) getAvgSediment() float64 {
	sum := 0.0
	count := 0
	for _, st := range s.stations {
		sed := (st.BaseSediment + s.scenario.SedimentBonus) * s.scenario.SedimentMultiplier
		sum += sed
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (s *Simulator) getAvgBedElevation() float64 {
	sum := 0.0
	count := 0
	for _, v := range s.bedState {
		sum += v
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
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
	apiBaseURL := flag.String("api", "http://localhost:8080/api/v1", "API server base URL")
	interval := flag.Duration("interval", 1*time.Second, "Simulation tick interval")
	speedFactor := flag.Float64("speed", 1.0, "Simulation speed factor (hours per tick)")
	historicalDays := flag.Int("historical", 0, "Generate N days of historical data first")
	scenario := flag.String("scenario", "normal", "Scenario preset (normal, flood, drought, maintenance, high_sediment, erosion)")
	waterLevelBonus := flag.Float64("water-level-bonus", 0, "Water level bonus in meters (overrides scenario)")
	sedimentBonus := flag.Float64("sediment-bonus", 0, "Sediment concentration bonus in kg/m^3 (overrides scenario)")
	flowMultiplier := flag.Float64("flow-multiplier", 1.0, "Flow rate multiplier (overrides scenario)")
	listScenariosFlag := flag.Bool("list-scenarios", false, "List all available scenarios and exit")
	flag.Parse()

	if *listScenariosFlag {
		listScenarios()
		os.Exit(0)
	}

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.Println("============================================")
	log.Println("都江堰水文模拟器启动")
	log.Println("============================================")
	log.Printf("API Server: %s", *apiBaseURL)
	log.Printf("Interval: %s", *interval)
	log.Printf("Speed Factor: %.1fx (%.1f hours per tick)", *speedFactor, *speedFactor)
	log.Printf("Historical Data: %d days", *historicalDays)
	log.Printf("Scenario: %s", *scenario)
	if *waterLevelBonus != 0 {
		log.Printf("Water Level Bonus: %.2fm", *waterLevelBonus)
	}
	if *sedimentBonus != 0 {
		log.Printf("Sediment Bonus: %.3f kg/m^3", *sedimentBonus)
	}
	if *flowMultiplier != 1.0 {
		log.Printf("Flow Multiplier: %.2fx", *flowMultiplier)
	}
	log.Printf("Stations: %d", len(baseStations))
	log.Println("============================================")

	if s, ok := scenarioPresets[*scenario]; ok {
		log.Printf("Scenario description: %s", s.Description)
	}
	log.Println("============================================")

	rand.Seed(time.Now().UnixNano())

	simulator := NewSimulator(*apiBaseURL, *interval, *speedFactor,
		*scenario, *waterLevelBonus, *sedimentBonus, *flowMultiplier)

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
