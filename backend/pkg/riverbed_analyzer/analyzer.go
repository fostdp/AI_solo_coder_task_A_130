package riverbed_analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"dujiangyan-system/pkg/models"
	"dujiangyan-system/pkg/msg"
	"gonum.org/v1/gonum/stat"
)

type WeirConfig struct {
	Name              string   `json:"name"`
	WeirType          string   `json:"weir_type"`
	CrestElevation    float64  `json:"crest_elevation"`
	WeirLength        float64  `json:"weir_length"`
	DischargeCoeff    float64  `json:"discharge_coeff"`
	SubmergenceRatio  float64  `json:"submergence_ratio"`
	BackwaterFactor   float64  `json:"backwater_factor"`
	SedimentTrapEff   float64  `json:"sediment_trap_eff"`
	ReleaseThreshold  float64  `json:"release_threshold"`
	UpstreamStations  []string `json:"upstream_stations"`
}

type SedimentTransportConfig struct {
	K              float64 `json:"K"`
	ExponentFlow   float64 `json:"exponent_flow"`
	ExponentSlope  float64 `json:"exponent_slope"`
	Porosity       float64 `json:"porosity"`
	BulkDensity    float64 `json:"bulk_density"`
	TimeStepYears  float64 `json:"time_step_years"`
}

type PredictionConfig struct {
	DefaultYears        int     `json:"default_years"`
	MinHistoryMonths    int     `json:"min_history_months"`
	SeasonalCycleYears  int     `json:"seasonal_cycle_years"`
	SeasonalAmplitude   float64 `json:"seasonal_amplitude"`
	ConfidenceDecay     float64 `json:"confidence_decay_per_year"`
	MinConfidence       float64 `json:"min_confidence"`
	BaseConfidence      float64 `json:"base_confidence"`
}

type EvolutionRateConfig struct {
	DepositionRatioUpstream float64 `json:"deposition_ratio_upstream"`
	BedrockOffset           float64 `json:"bedrock_offset"`
}

type SedimentConfig struct {
	SedimentTransport SedimentTransportConfig `json:"sediment_transport"`
	Weirs             []WeirConfig            `json:"weirs"`
	Prediction        PredictionConfig        `json:"prediction"`
	EvolutionRate     EvolutionRateConfig     `json:"evolution_rate"`
}

type SedimentTransportModel struct {
	config SedimentConfig
}

type BedEvolutionResult struct {
	Year               int       `json:"year"`
	PredictionDate     time.Time `json:"prediction_date"`
	PredictedElevation float64   `json:"predicted_elevation"`
	Deposition         float64   `json:"deposition"`
	Erosion            float64   `json:"erosion"`
	NetChange          float64   `json:"net_change"`
	Confidence         float64   `json:"confidence"`
}

func LoadSedimentConfig(path string) (*SedimentConfig, error) {
	if path == "" {
		path = "config/sediment_params.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: sediment config not found at %s, using defaults", path)
		return DefaultSedimentConfig(), nil
	}
	var config SedimentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid sediment config: %w", err)
	}
	return &config, nil
}

func DefaultSedimentConfig() *SedimentConfig {
	return &SedimentConfig{
		SedimentTransport: SedimentTransportConfig{
			K: 0.01, ExponentFlow: 2.0, ExponentSlope: 1.5,
			Porosity: 0.4, BulkDensity: 2650.0, TimeStepYears: 1.0,
		},
		Weirs: []WeirConfig{
			{Name: "飞沙堰", WeirType: "overflow", CrestElevation: 728.0, WeirLength: 120.0,
				DischargeCoeff: 0.45, BackwaterFactor: 0.15, SedimentTrapEff: 0.35,
				ReleaseThreshold: 730.0, UpstreamStations: []string{"NEIJ-001", "NEIJ-002", "WAIJ-001", "WAIJ-002", "FSSY-001"}},
			{Name: "宝瓶口", WeirType: "orifice", CrestElevation: 727.0, WeirLength: 20.0,
				DischargeCoeff: 0.62, BackwaterFactor: 0.25, SedimentTrapEff: 0.20,
				ReleaseThreshold: 731.0, UpstreamStations: []string{"NEIJ-001", "NEIJ-002", "NEIJ-003", "FSSY-001", "FSSY-002"}},
			{Name: "人字堤", WeirType: "overflow", CrestElevation: 727.5, WeirLength: 80.0,
				DischargeCoeff: 0.40, BackwaterFactor: 0.10, SedimentTrapEff: 0.15,
				ReleaseThreshold: 729.5, UpstreamStations: []string{"NEIJ-001", "NEIJ-002", "NEIJ-003", "RJK-001"}},
		},
		Prediction: PredictionConfig{
			DefaultYears: 10, MinHistoryMonths: 24, SeasonalCycleYears: 10,
			SeasonalAmplitude: 0.3, ConfidenceDecay: 0.03, MinConfidence: 0.6, BaseConfidence: 0.95,
		},
		EvolutionRate: EvolutionRateConfig{
			DepositionRatioUpstream: 1.1, BedrockOffset: 0.5,
		},
	}
}

func NewSedimentTransportModel(config *SedimentConfig) *SedimentTransportModel {
	if config == nil {
		config = DefaultSedimentConfig()
	}
	return &SedimentTransportModel{config: *config}
}

func (m *SedimentTransportModel) CalculateSedimentTransport(flowRate, slope, sedimentConc float64) float64 {
	st := m.config.SedimentTransport
	transportCapacity := st.K * math.Pow(flowRate, st.ExponentFlow) * math.Pow(slope, st.ExponentSlope)
	actualTransport := sedimentConc * flowRate * 3600 * 24 * 365
	return math.Min(actualTransport, transportCapacity)
}

func (m *SedimentTransportModel) CalculateBedChange(sedimentIn, sedimentOut, channelWidth, channelLength float64) float64 {
	st := m.config.SedimentTransport
	sedimentDelta := sedimentIn - sedimentOut
	bulkVolume := sedimentDelta / (st.BulkDensity * (1 - st.Porosity))
	area := channelWidth * channelLength
	return bulkVolume / area
}

func (w *WeirConfig) CalculateWeirOverflow(headwaterLevel float64) float64 {
	if headwaterLevel <= w.CrestElevation {
		return 0
	}
	head := headwaterLevel - w.CrestElevation
	if w.SubmergenceRatio > 0.8 {
		head *= math.Pow(1-w.SubmergenceRatio, 0.5)
	}
	return w.DischargeCoeff * w.WeirLength * math.Pow(head, 1.5) * 9.81
}

func (w *WeirConfig) CalculateBackwaterEffect(upstreamLevel float64) float64 {
	return math.Max(0, w.BackwaterFactor*math.Pow(upstreamLevel-w.CrestElevation, 0.5))
}

func (w *WeirConfig) CalculateSedimentRetention(sedimentLoad float64) float64 {
	return sedimentLoad * w.SedimentTrapEff
}

func (m *SedimentTransportModel) isStationUpstreamOfWeir(stationID string, weir *WeirConfig) bool {
	for _, s := range weir.UpstreamStations {
		if s == stationID {
			return true
		}
	}
	return false
}

func (m *SedimentTransportModel) ApplyWeirBoundaryConditions(flowRate, waterLevel, sedimentLoad float64, stationID string) (float64, float64) {
	adjustedFlowRate := flowRate
	adjustedSediment := sedimentLoad

	for i := range m.config.Weirs {
		weir := &m.config.Weirs[i]
		if !m.isStationUpstreamOfWeir(stationID, weir) {
			continue
		}

		backwaterRise := weir.CalculateBackwaterEffect(waterLevel)
		adjustedFlowRate *= (1 + backwaterRise*0.01)

		if waterLevel > weir.ReleaseThreshold {
			overflowQ := weir.CalculateWeirOverflow(waterLevel)
			flowRatio := 1.0 - math.Min(overflowQ/adjustedFlowRate, 0.3)
			adjustedFlowRate *= flowRatio
			adjustedSediment *= (1.0 - weir.SedimentTrapEff*0.3)
		}

		if waterLevel > weir.CrestElevation && waterLevel <= weir.ReleaseThreshold {
			trapped := weir.CalculateSedimentRetention(adjustedSediment)
			adjustedSediment -= trapped * 0.1
		}
	}

	return math.Max(adjustedFlowRate, 10.0), math.Max(adjustedSediment, 0.01)
}

type RiverbedAnalyzer struct {
	ctx    context.Context
	bus    chan<- msg.BusMessage
	config *SedimentConfig
}

func NewRiverbedAnalyzer(ctx context.Context, bus chan<- msg.BusMessage, config *SedimentConfig) *RiverbedAnalyzer {
	if config == nil {
		config = DefaultSedimentConfig()
	}
	return &RiverbedAnalyzer{ctx: ctx, bus: bus, config: config}
}

func (ra *RiverbedAnalyzer) PredictBedEvolution(stationID string, years int) ([]BedEvolutionResult, error) {
	model := NewSedimentTransportModel(ra.config)

	hydrologyData, err := models.GetHydrologyData(
		ra.ctx, stationID,
		time.Now().AddDate(-2, 0, 0), time.Now(), 10000,
	)
	if err != nil {
		return nil, err
	}

	if len(hydrologyData) < ra.config.Prediction.MinHistoryMonths {
		log.Printf("Not enough history for %s, using defaults", stationID)
		return ra.generateDefaultPrediction(stationID, years)
	}

	stations, _ := models.GetMonitoringStations(ra.ctx)
	var currentStation *models.MonitoringStation
	for _, s := range stations {
		if s.StationID == stationID {
			currentStation = &s
			break
		}
	}

	var flowRates, sedimentConcs, bedElevations []float64
	for _, d := range hydrologyData {
		flowRates = append(flowRates, d.FlowRate)
		sedimentConcs = append(sedimentConcs, d.SedimentConcentration)
		bedElevations = append(bedElevations, d.BedElevation)
	}

	meanFlow := stat.Mean(flowRates, nil)
	stdFlow := stat.StdDev(flowRates, nil)
	meanSediment := stat.Mean(sedimentConcs, nil)

	initialBedElevation := bedElevations[len(bedElevations)-1]
	if currentStation != nil {
		initialBedElevation = math.Max(initialBedElevation, currentStation.BedrockElevation+1.0)
	}

	slope := 0.001
	channelWidth := 50.0
	channelLength := 1000.0

	var results []BedEvolutionResult
	currentElevation := initialBedElevation
	predCfg := ra.config.Prediction

	for year := 1; year <= years; year++ {
		annualFlow := meanFlow + (math.Sin(float64(year)*0.5) * stdFlow * 0.3)
		annualSediment := meanSediment + (math.Sin(float64(year)*0.3+1.0) * stat.StdDev(sedimentConcs, nil) * 0.5)
		seasonalFactor := 1.0 + predCfg.SeasonalAmplitude*math.Sin(float64(year)*2*math.Pi/float64(predCfg.SeasonalCycleYears))
		annualFlow *= seasonalFactor
		annualSediment *= seasonalFactor

		adjustedFlow, adjustedSediment := model.ApplyWeirBoundaryConditions(
			annualFlow, 728.5+math.Sin(float64(year)*0.5)*0.3, annualSediment, stationID,
		)

		sedimentTransported := model.CalculateSedimentTransport(adjustedFlow, slope, adjustedSediment)
		upstreamSediment := adjustedSediment * adjustedFlow * 3600 * 24 * 365 * ra.config.EvolutionRate.DepositionRatioUpstream
		downstreamSediment := sedimentTransported

		var annualDeposition, annualErosion float64
		if upstreamSediment > downstreamSediment {
			annualDeposition = model.CalculateBedChange(upstreamSediment, downstreamSediment, channelWidth, channelLength)
			currentElevation += annualDeposition
		} else {
			annualErosion = model.CalculateBedChange(upstreamSediment, downstreamSediment, channelWidth, channelLength)
			currentElevation += annualErosion
		}

		if currentStation != nil {
			bedrockLimit := currentStation.BedrockElevation + ra.config.EvolutionRate.BedrockOffset
			if currentElevation < bedrockLimit {
				currentElevation = bedrockLimit
			}
		}

		confidence := predCfg.BaseConfidence - float64(year)*predCfg.ConfidenceDecay
		if confidence < predCfg.MinConfidence {
			confidence = predCfg.MinConfidence
		}

		predictionDate := time.Now().AddDate(year, 0, 0)
		result := BedEvolutionResult{
			Year: year, PredictionDate: predictionDate,
			PredictedElevation: currentElevation,
			Deposition: math.Max(0, annualDeposition),
			Erosion:    math.Abs(math.Min(0, annualErosion)),
			NetChange:  annualDeposition - math.Abs(math.Min(0, annualErosion)),
			Confidence: confidence,
		}
		results = append(results, result)

		prediction := &models.BedEvolutionPrediction{
			StationID: stationID, PredictionDate: predictionDate,
			ForecastHorizonMonths: year * 12, PredictedBedElevation: currentElevation,
			PredictedSedimentDeposition: math.Max(0, annualDeposition),
			PredictedErosion: math.Abs(math.Min(0, annualErosion)),
			ModelVersion: "v2.0-weir-boundary", Confidence: confidence,
		}
		if err := models.InsertBedEvolutionPrediction(ra.ctx, prediction); err != nil {
			log.Printf("Failed to insert prediction: %v", err)
		}
	}

	ra.bus <- msg.BusMessage{
		Type: msg.TypePredictResult, Timestamp: time.Now(),
		Payload: msg.PredictResultPayload{
			StationID: stationID, Years: years,
			FinalElevation: currentElevation, ModelVersion: "v2.0-weir-boundary",
		},
	}

	return results, nil
}

func (ra *RiverbedAnalyzer) generateDefaultPrediction(stationID string, years int) ([]BedEvolutionResult, error) {
	model := NewSedimentTransportModel(ra.config)
	var results []BedEvolutionResult
	currentElevation := 728.5

	for year := 1; year <= years; year++ {
		depositionRate := 0.08 + 0.02*math.Sin(float64(year)*0.4)
		erosionRate := 0.03 + 0.01*math.Sin(float64(year)*0.6)

		waterLevel := 728.5 + math.Sin(float64(year)*0.5)*0.3
		flowRate := 250.0 + math.Sin(float64(year)*0.4)*50
		sedimentLoad := depositionRate * 2650 * 50 * 1000

		_, adjustedSediment := model.ApplyWeirBoundaryConditions(flowRate, waterLevel, sedimentLoad, stationID)
		sedimentRatio := adjustedSediment / math.Max(sedimentLoad, 1)
		depositionRate *= sedimentRatio
		erosionRate *= (2.0 - sedimentRatio)

		netChange := depositionRate - erosionRate
		currentElevation += netChange

		confidence := ra.config.Prediction.BaseConfidence - float64(year)*ra.config.Prediction.ConfidenceDecay
		if confidence < ra.config.Prediction.MinConfidence {
			confidence = ra.config.Prediction.MinConfidence
		}

		results = append(results, BedEvolutionResult{
			Year: year, PredictionDate: time.Now().AddDate(year, 0, 0),
			PredictedElevation: currentElevation,
			Deposition: depositionRate, Erosion: erosionRate,
			NetChange: netChange, Confidence: confidence,
		})
	}
	return results, nil
}
