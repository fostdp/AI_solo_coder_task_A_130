package simulation

import (
	"context"
	"log"
	"math"
	"time"

	"dujiangyan-system/pkg/models"
	"gonum.org/v1/gonum/stat"
)

type SedimentTransportModel struct {
	K              float64
	ExponentFlow   float64
	ExponentSlope  float64
	Porosity       float64
	BulkDensity    float64
	TimeStepYears  float64
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

func NewSedimentTransportModel() *SedimentTransportModel {
	return &SedimentTransportModel{
		K:             0.01,
		ExponentFlow:  2.0,
		ExponentSlope: 1.5,
		Porosity:      0.4,
		BulkDensity:   2650.0,
		TimeStepYears: 1.0,
	}
}

func (m *SedimentTransportModel) CalculateSedimentTransport(flowRate, slope, sedimentConc float64) float64 {
	transportCapacity := m.K * math.Pow(flowRate, m.ExponentFlow) * math.Pow(slope, m.ExponentSlope)
	actualTransport := sedimentConc * flowRate * 3600 * 24 * 365
	return math.Min(actualTransport, transportCapacity)
}

func (m *SedimentTransportModel) CalculateBedChange(
	sedimentIn, sedimentOut, channelWidth, channelLength float64) float64 {
	sedimentDelta := sedimentIn - sedimentOut
	bulkVolume := sedimentDelta / (m.BulkDensity * (1 - m.Porosity))
	area := channelWidth * channelLength
	elevationChange := bulkVolume / area
	return elevationChange
}

func PredictBedEvolution(
	ctx context.Context,
	stationID string,
	years int,
) ([]BedEvolutionResult, error) {

	model := NewSedimentTransportModel()

	hydrologyData, err := models.GetHydrologyData(
		ctx, stationID,
		time.Now().AddDate(-2, 0, 0),
		time.Now(),
		10000,
	)
	if err != nil {
		return nil, err
	}

	if len(hydrologyData) < 24 {
		log.Printf("Not enough historical data for station %s, using default parameters", stationID)
		return generateDefaultPrediction(ctx, stationID, years)
	}

	station, err := models.GetMonitoringStations(ctx)
	if err != nil {
		return nil, err
	}

	var currentStation *models.MonitoringStation
	for _, s := range station {
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
	stdSediment := stat.StdDev(sedimentConcs, nil)

	initialBedElevation := bedElevations[len(bedElevations)-1]
	if currentStation != nil {
		initialBedElevation = math.Max(initialBedElevation, currentStation.BedrockElevation+1.0)
	}

	slope := 0.001
	channelWidth := 50.0
	channelLength := 1000.0

	var results []BedEvolutionResult
	currentElevation := initialBedElevation

	for year := 1; year <= years; year++ {
		annualFlow := meanFlow + (math.Sin(float64(year)*0.5) * stdFlow * 0.3)
		annualSediment := meanSediment + (math.Sin(float64(year)*0.3+1.0) * stdSediment * 0.5)

		seasonalFactor := 1.0 + 0.3*math.Sin(float64(year)*2*math.Pi/10)
		annualFlow *= seasonalFactor
		annualSediment *= seasonalFactor

		sedimentTransported := model.CalculateSedimentTransport(annualFlow, slope, annualSediment)

		upstreamSediment := annualSediment * annualFlow * 3600 * 24 * 365 * 1.1
		downstreamSediment := sedimentTransported

		annualDeposition := 0.0
		annualErosion := 0.0

		if upstreamSediment > downstreamSediment {
			annualDeposition = model.CalculateBedChange(
				upstreamSediment, downstreamSediment, channelWidth, channelLength)
			currentElevation += annualDeposition
		} else {
			annualErosion = model.CalculateBedChange(
				upstreamSediment, downstreamSediment, channelWidth, channelLength)
			currentElevation += annualErosion
		}

		bedrockLimit := currentStation.BedrockElevation + 0.5
		if currentElevation < bedrockLimit {
			excessErosion := bedrockLimit - currentElevation
			currentElevation = bedrockLimit
			annualErosion += excessErosion
		}

		confidence := 0.95 - float64(year)*0.03
		if confidence < 0.6 {
			confidence = 0.6
		}

		predictionDate := time.Now().AddDate(year, 0, 0)
		result := BedEvolutionResult{
			Year:               year,
			PredictionDate:     predictionDate,
			PredictedElevation: currentElevation,
			Deposition:         math.Max(0, annualDeposition),
			Erosion:            math.Abs(math.Min(0, annualErosion)),
			NetChange:          annualDeposition - math.Abs(annualErosion),
			Confidence:         confidence,
		}
		results = append(results, result)

		prediction := &models.BedEvolutionPrediction{
			StationID:                 stationID,
			PredictionDate:            predictionDate,
			ForecastHorizonMonths:     year * 12,
			PredictedBedElevation:     currentElevation,
			PredictedSedimentDeposition: math.Max(0, annualDeposition),
			PredictedErosion:          math.Abs(math.Min(0, annualErosion)),
			ModelVersion:              "v1.0",
			Confidence:                confidence,
		}

		if err := models.InsertBedEvolutionPrediction(ctx, prediction); err != nil {
			log.Printf("Failed to insert prediction for year %d: %v", year, err)
		}
	}

	return results, nil
}

func generateDefaultPrediction(
	ctx context.Context,
	stationID string,
	years int,
) ([]BedEvolutionResult, error) {
	stations, err := models.GetMonitoringStations(ctx)
	if err != nil {
		return nil, err
	}

	var bedrockElevation float64 = 726.0
	for _, s := range stations {
		if s.StationID == stationID {
			bedrockElevation = s.BedrockElevation
			break
		}
	}

	var results []BedEvolutionResult
	currentElevation := bedrockElevation + 2.0

	for year := 1; year <= years; year++ {
		depositionRate := 0.08 + 0.02*math.Sin(float64(year)*0.4)
		erosionRate := 0.03 + 0.01*math.Sin(float64(year)*0.6)

		annualDeposition := depositionRate
		annualErosion := erosionRate
		netChange := annualDeposition - annualErosion

		currentElevation += netChange

		bedrockLimit := bedrockElevation + 0.5
		if currentElevation < bedrockLimit {
			currentElevation = bedrockLimit
		}

		confidence := 0.9 - float64(year)*0.025
		if confidence < 0.6 {
			confidence = 0.6
		}

		predictionDate := time.Now().AddDate(year, 0, 0)
		result := BedEvolutionResult{
			Year:               year,
			PredictionDate:     predictionDate,
			PredictedElevation: currentElevation,
			Deposition:         annualDeposition,
			Erosion:            annualErosion,
			NetChange:          netChange,
			Confidence:         confidence,
		}
		results = append(results, result)

		prediction := &models.BedEvolutionPrediction{
			StationID:                   stationID,
			PredictionDate:              predictionDate,
			ForecastHorizonMonths:       year * 12,
			PredictedBedElevation:       currentElevation,
			PredictedSedimentDeposition: annualDeposition,
			PredictedErosion:            annualErosion,
			ModelVersion:                "v1.0-default",
			Confidence:                  confidence,
		}

		if err := models.InsertBedEvolutionPrediction(ctx, prediction); err != nil {
			log.Printf("Failed to insert default prediction: %v", err)
		}
	}

	return results, nil
}

func GenerateDEMGrid(
	ctx context.Context,
	centerX, centerY, gridSize, resolution float64,
	baseElevation float64,
) [][]models.DEMGrid {
	gridCells := int(gridSize / resolution)
	grid := make([][]models.DEMGrid, gridCells)

	for i := 0; i < gridCells; i++ {
		grid[i] = make([]models.DEMGrid, gridCells)
		for j := 0; j < gridCells; j++ {
			dx := (float64(i) - float64(gridCells)/2) * resolution
			dy := (float64(j) - float64(gridCells)/2) * resolution

			distanceFromCenter := math.Sqrt(dx*dx + dy*dy)
			channelFactor := math.Exp(-math.Pow(dx/20, 2))
			undulation := 0.3 * math.Sin(dx*0.1) * math.Cos(dy*0.1)
			slopeEffect := dy * 0.002

			elevation := baseElevation + undulation + slopeEffect
			waterDepth := 2.5*channelFactor + 0.5*math.Exp(-math.Pow(distanceFromCenter/50, 2))

			grid[i][j] = models.DEMGrid{
				GridX:      i,
				GridY:      j,
				Elevation:  elevation,
				WaterDepth: math.Max(0, waterDepth),
			}
		}
	}

	return grid
}

func CalculateEvolutionRate(data []models.HydrologyData) (float64, float64, float64) {
	if len(data) < 2 {
		return 0, 0, 0
	}

	var deposition, erosion float64
	var totalChange float64

	for i := 1; i < len(data); i++ {
		change := data[i].BedElevation - data[i-1].BedElevation
		totalChange += change
		if change > 0 {
			deposition += change
		} else {
			erosion += math.Abs(change)
		}
	}

	hours := data[0].Time.Sub(data[len(data)-1].Time).Hours()
	if hours < 1 {
		hours = 1
	}

	annualFactor := 365.0 * 24.0 / hours

	return deposition * annualFactor, erosion * annualFactor, totalChange * annualFactor
}
