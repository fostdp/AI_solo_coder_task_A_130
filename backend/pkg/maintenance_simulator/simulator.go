package maintenance_simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	"dujiangyan-system/pkg/models"
	"dujiangyan-system/pkg/msg"
)

type DEMConfig struct {
	Gravity              float64 `json:"gravity"`
	Restitution          float64 `json:"restitution"`
	Friction             float64 `json:"friction"`
	Viscosity            float64 `json:"viscosity"`
	TimeStep             float64 `json:"time_step"`
	SpatialHashCellSize  float64 `json:"spatial_hash_cell_size"`
	MaxWorkers           int     `json:"max_workers"`
	CollisionImpulseScale float64 `json:"collision_impulse_scale"`
}

type BambooCageConfig struct {
	DefaultDiameter    float64 `json:"default_diameter"`
	DefaultLength      float64 `json:"default_length"`
	Porosity           float64 `json:"porosity"`
	StoneCountPerCage  int     `json:"stone_count_per_cage"`
	StoneRadiusMin     float64 `json:"stone_radius_range.0"`
	StoneRadiusMax     float64 `json:"stone_radius_range.1"`
	StoneDensity       float64 `json:"stone_density"`
	CageSpacing        float64 `json:"cage_spacing"`
	CagesPerRow        int     `json:"cages_per_row"`
	ConvergenceSteps   int     `json:"convergence_steps"`
}

type MachaConfig struct {
	DefaultHeight       float64 `json:"default_height"`
	DefaultAngle        float64 `json:"default_angle"`
	AngleIncrement      float64 `json:"angle_increment"`
	LogCountMin         int     `json:"log_count_range.0"`
	LogCountMax         int     `json:"log_count_range.1"`
	BindingStrength     float64 `json:"binding_strength"`
	BaseEfficiency      float64 `json:"base_efficiency"`
	EfficiencyIncrement float64 `json:"efficiency_increment"`
	MaxEfficiency       float64 `json:"max_efficiency"`
	Spacing             float64 `json:"spacing"`
}

type CraftConfig struct {
	DEM         DEMConfig         `json:"dem"`
	BambooCage  BambooCageConfig  `json:"bamboo_cage"`
	Macha       MachaConfig       `json:"macha"`
}

type StoneParticle struct {
	ID        int     `json:"id"`
	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
	PositionZ float64 `json:"position_z"`
	VelocityX float64 `json:"velocity_x"`
	VelocityY float64 `json:"velocity_y"`
	VelocityZ float64 `json:"velocity_z"`
	Radius    float64 `json:"radius"`
	Mass      float64 `json:"mass"`
	Fixed     bool    `json:"fixed"`
}

type BambooCage struct {
	CageID    string          `json:"cage_id"`
	PositionX float64         `json:"position_x"`
	PositionY float64         `json:"position_y"`
	PositionZ float64         `json:"position_z"`
	Diameter  float64         `json:"diameter"`
	Length    float64         `json:"length"`
	Porosity  float64         `json:"porosity"`
	Stones    []StoneParticle `json:"stones"`
	Stability float64         `json:"stability"`
}

type MachaStructure struct {
	ID              int     `json:"id"`
	PositionX       float64 `json:"position_x"`
	PositionY       float64 `json:"position_y"`
	PositionZ       float64 `json:"position_z"`
	Height          float64 `json:"height"`
	Angle           float64 `json:"angle"`
	LogCount        int     `json:"log_count"`
	BindingStrength float64 `json:"binding_strength"`
	Efficiency      float64 `json:"efficiency"`
}

type collisionPair struct {
	i, j   int
	overlap float64
	normal [3]float64
}

type SpatialHashGrid struct {
	cellSize float64
	cells    map[int64][]int
}

func newSpatialHashGrid(cellSize float64) *SpatialHashGrid {
	return &SpatialHashGrid{cellSize: cellSize, cells: make(map[int64][]int)}
}

func (g *SpatialHashGrid) cellKey(cx, cy, cz int) int64 {
	return int64(cx)*73856093 ^ int64(cy)*19349663 ^ int64(cz)*83492791
}

func (g *SpatialHashGrid) clear() {
	for k := range g.cells {
		delete(g.cells, k)
	}
}

func (g *SpatialHashGrid) insert(index int, px, py, pz, radius float64) {
	minCX := int(math.Floor((px - radius) / g.cellSize))
	minCY := int(math.Floor((py - radius) / g.cellSize))
	minCZ := int(math.Floor((pz - radius) / g.cellSize))
	maxCX := int(math.Floor((px + radius) / g.cellSize))
	maxCY := int(math.Floor((py + radius) / g.cellSize))
	maxCZ := int(math.Floor((pz + radius) / g.cellSize))
	for cx := minCX; cx <= maxCX; cx++ {
		for cy := minCY; cy <= maxCY; cy++ {
			for cz := minCZ; cz <= maxCZ; cz++ {
				key := g.cellKey(cx, cy, cz)
				g.cells[key] = append(g.cells[key], index)
			}
		}
	}
}

func (g *SpatialHashGrid) queryPotentialCollisions() []collisionPair {
	seen := make(map[[2]int]bool)
	var pairs []collisionPair
	for _, indices := range g.cells {
		n := len(indices)
		if n < 2 {
			continue
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				a, b := indices[i], indices[j]
				if a > b {
					a, b = b, a
				}
				key := [2]int{a, b}
				if seen[key] {
					continue
				}
				seen[key] = true
				pairs = append(pairs, collisionPair{i: a, j: b})
			}
		}
	}
	return pairs
}

type DEMSimulation struct {
	SimulationID int64
	config       CraftConfig
	Stones       []StoneParticle
	Cages        []BambooCage
	Machas       []MachaStructure
	spatialGrid  *SpatialHashGrid
	workers      int
}

func LoadCraftConfig(path string) (*CraftConfig, error) {
	if path == "" {
		path = "config/craft_params.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: craft config not found at %s, using defaults", path)
		return DefaultCraftConfig(), nil
	}
	var config CraftConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid craft config: %w", err)
	}
	return &config, nil
}

func DefaultCraftConfig() *CraftConfig {
	return &CraftConfig{
		DEM: DEMConfig{
			Gravity: -9.81, Restitution: 0.3, Friction: 0.6, Viscosity: 0.98,
			TimeStep: 0.01, SpatialHashCellSize: 2.0, MaxWorkers: 8,
		},
		BambooCage: BambooCageConfig{
			DefaultDiameter: 0.9, DefaultLength: 2.4, Porosity: 0.35,
			StoneCountPerCage: 100, StoneRadiusMin: 0.04, StoneRadiusMax: 0.12,
			StoneDensity: 2650, CageSpacing: 1.2, CagesPerRow: 5, ConvergenceSteps: 500,
		},
		Macha: MachaConfig{
			DefaultHeight: 4.5, DefaultAngle: 15.0, AngleIncrement: 2.0,
			LogCountMin: 6, LogCountMax: 10, BindingStrength: 0.8,
			BaseEfficiency: 0.05, EfficiencyIncrement: 0.02,
			MaxEfficiency: 0.85, Spacing: 3.0,
		},
	}
}

func NewDEMSimulation(simID int64, config *CraftConfig) *DEMSimulation {
	workers := config.DEM.MaxWorkers
	if workers < 1 {
		workers = runtime.NumCPU()
	}
	if workers > 16 {
		workers = 16
	}
	return &DEMSimulation{
		SimulationID: simID,
		config:       *config,
		workers:      workers,
		spatialGrid:  newSpatialHashGrid(config.DEM.SpatialHashCellSize),
	}
}

func (s *DEMSimulation) AddStone(x, y, z, radius float64, fixed bool) int {
	stone := StoneParticle{
		ID: len(s.Stones), PositionX: x, PositionY: y, PositionZ: z,
		VelocityX: 0, VelocityY: 0, VelocityZ: 0,
		Radius: radius, Mass: (4.0 / 3.0) * math.Pi * math.Pow(radius, 3) * s.config.BambooCage.StoneDensity,
		Fixed: fixed,
	}
	s.Stones = append(s.Stones, stone)
	return stone.ID
}

func (s *DEMSimulation) CreateBambooCage(cageID string, x, y, z, diameter, length float64, stoneCount int) *BambooCage {
	cfg := s.config.BambooCage
	cage := BambooCage{
		CageID: cageID, PositionX: x, PositionY: y, PositionZ: z,
		Diameter: diameter, Length: length, Porosity: cfg.Porosity, Stability: 0.0,
	}

	stoneVolume := math.Pi * math.Pow(diameter/2, 2) * length * (1 - cage.Porosity)
	avgStoneVolume := stoneVolume / float64(stoneCount)
	avgStoneRadius := math.Pow(3*avgStoneVolume/(4*math.Pi), 1.0/3.0)

	for i := 0; i < stoneCount; i++ {
		sx := x + (rand.Float64()-0.5)*diameter*0.8
		sy := y + (rand.Float64()-0.5)*length*0.8
		sz := z + diameter*0.5 + rand.Float64()*diameter*0.3
		sr := avgStoneRadius * (0.7 + rand.Float64()*0.6)
		stoneID := s.AddStone(sx, sy, sz, sr, false)
		cage.Stones = append(cage.Stones, s.Stones[stoneID])
	}

	s.Cages = append(s.Cages, cage)
	return &s.Cages[len(s.Cages)-1]
}

func (s *DEMSimulation) CreateMacha(x, y, z, height, angle float64, logCount int) *MachaStructure {
	macha := MachaStructure{
		ID: len(s.Machas), PositionX: x, PositionY: y, PositionZ: z,
		Height: height, Angle: angle, LogCount: logCount,
		BindingStrength: s.config.Macha.BindingStrength, Efficiency: 0.0,
	}
	s.Machas = append(s.Machas, macha)
	return &s.Machas[len(s.Machas)-1]
}

func (s *DEMSimulation) detectCollision(i, j int) (bool, float64, [3]float64) {
	si, sj := &s.Stones[i], &s.Stones[j]
	dx := sj.PositionX - si.PositionX
	dy := sj.PositionY - si.PositionY
	dz := sj.PositionZ - si.PositionZ
	distanceSq := dx*dx + dy*dy + dz*dz
	minDist := si.Radius + sj.Radius
	if distanceSq < minDist*minDist && distanceSq > 0 {
		distance := math.Sqrt(distanceSq)
		return true, minDist - distance, [3]float64{dx / distance, dy / distance, dz / distance}
	}
	return false, 0, [3]float64{}
}

func (s *DEMSimulation) resolveCollision(i, j int, overlap float64, normal [3]float64) {
	si, sj := &s.Stones[i], &s.Stones[j]
	if !si.Fixed {
		si.PositionX -= normal[0] * overlap * 0.5
		si.PositionY -= normal[1] * overlap * 0.5
		si.PositionZ -= normal[2] * overlap * 0.5
	}
	if !sj.Fixed {
		sj.PositionX += normal[0] * overlap * 0.5
		sj.PositionY += normal[1] * overlap * 0.5
		sj.PositionZ += normal[2] * overlap * 0.5
	}
	rvx := sj.VelocityX - si.VelocityX
	rvy := sj.VelocityY - si.VelocityY
	rvz := sj.VelocityZ - si.VelocityZ
	velAlongNormal := rvx*normal[0] + rvy*normal[1] + rvz*normal[2]
	if velAlongNormal > 0 {
		return
	}
	impulse := -(1 + s.config.DEM.Restitution) * velAlongNormal / (1/si.Mass + 1/sj.Mass)
	if !si.Fixed {
		si.VelocityX -= impulse * normal[0] / si.Mass
		si.VelocityY -= impulse * normal[1] / si.Mass
		si.VelocityZ -= impulse * normal[2] / si.Mass
	}
	if !sj.Fixed {
		sj.VelocityX += impulse * normal[0] / sj.Mass
		sj.VelocityY += impulse * normal[1] / sj.Mass
		sj.VelocityZ += impulse * normal[2] / sj.Mass
	}
}

func (s *DEMSimulation) applyForcesParallel() {
	n := len(s.Stones)
	if n == 0 {
		return
	}
	chunkSize := (n + s.workers - 1) / s.workers
	var wg sync.WaitGroup
	for w := 0; w < s.workers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > n {
			end = n
		}
		if start >= end {
			continue
		}
		wg.Add(1)
		go func(startIdx, endIdx int) {
			defer wg.Done()
			for i := startIdx; i < endIdx; i++ {
				if s.Stones[i].Fixed {
					continue
				}
				s.Stones[i].VelocityZ += s.config.DEM.Gravity * s.config.DEM.TimeStep
				s.Stones[i].VelocityX *= s.config.DEM.Viscosity
				s.Stones[i].VelocityY *= s.config.DEM.Viscosity
				s.Stones[i].VelocityZ *= s.config.DEM.Viscosity
				s.Stones[i].PositionX += s.Stones[i].VelocityX * s.config.DEM.TimeStep
				s.Stones[i].PositionY += s.Stones[i].VelocityY * s.config.DEM.TimeStep
				s.Stones[i].PositionZ += s.Stones[i].VelocityZ * s.config.DEM.TimeStep
				if s.Stones[i].PositionZ < s.Stones[i].Radius {
					s.Stones[i].PositionZ = s.Stones[i].Radius
					if s.Stones[i].VelocityZ < 0 {
						s.Stones[i].VelocityZ = -s.Stones[i].VelocityZ * s.config.DEM.Restitution
					}
					frictionForce := s.config.DEM.Friction * s.config.DEM.Gravity * s.Stones[i].Mass * s.config.DEM.TimeStep
					speed := math.Sqrt(s.Stones[i].VelocityX*s.Stones[i].VelocityX + s.Stones[i].VelocityY*s.Stones[i].VelocityY)
					if speed > 0 {
						s.Stones[i].VelocityX -= (s.Stones[i].VelocityX / speed) * math.Min(frictionForce/s.Stones[i].Mass, speed)
						s.Stones[i].VelocityY -= (s.Stones[i].VelocityY / speed) * math.Min(frictionForce/s.Stones[i].Mass, speed)
					}
				}
			}
		}(start, end)
	}
	wg.Wait()
}

func (s *DEMSimulation) detectCollisionsSpatial() []collisionPair {
	s.spatialGrid.clear()
	for i := range s.Stones {
		st := &s.Stones[i]
		s.spatialGrid.insert(i, st.PositionX, st.PositionY, st.PositionZ, st.Radius)
	}
	potentialPairs := s.spatialGrid.queryPotentialCollisions()
	var actual []collisionPair
	for _, pair := range potentialPairs {
		if colliding, overlap, normal := s.detectCollision(pair.i, pair.j); colliding {
			actual = append(actual, collisionPair{i: pair.i, j: pair.j, overlap: overlap, normal: normal})
		}
	}
	return actual
}

func (s *DEMSimulation) resolveCollisionsParallel(collisions []collisionPair) {
	if len(collisions) == 0 {
		return
	}
	groupCount := s.workers
	if groupCount > len(collisions) {
		groupCount = len(collisions)
	}
	chunkSize := (len(collisions) + groupCount - 1) / groupCount
	var wg sync.WaitGroup
	for g := 0; g < groupCount; g++ {
		start := g * chunkSize
		end := start + chunkSize
		if end > len(collisions) {
			end = len(collisions)
		}
		if start >= end {
			continue
		}
		wg.Add(1)
		go func(si, ei int) {
			defer wg.Done()
			for k := si; k < ei; k++ {
				c := collisions[k]
				s.resolveCollision(c.i, c.j, c.overlap, c.normal)
			}
		}(start, end)
	}
	wg.Wait()
}

func (s *DEMSimulation) updateCageStability(cageIndex int) {
	cage := &s.Cages[cageIndex]
	if len(cage.Stones) == 0 {
		cage.Stability = 0
		return
	}
	var totalKE, maxHeight, minHeight float64
	var contactCount int
	for _, stone := range cage.Stones {
		ke := 0.5 * stone.Mass * (stone.VelocityX*stone.VelocityX + stone.VelocityY*stone.VelocityY + stone.VelocityZ*stone.VelocityZ)
		totalKE += ke
		if stone.PositionZ > maxHeight {
			maxHeight = stone.PositionZ
		}
		if stone.PositionZ-stone.Radius < 0.1 {
			contactCount++
		}
	}
	minHeight = cage.PositionZ
	heightRatio := minHeight / math.Max(maxHeight, 1)
	contactRatio := float64(contactCount) / float64(len(cage.Stones))
	kineticFactor := math.Exp(-totalKE * 0.001)
	cage.Stability = (heightRatio*0.3 + contactRatio*0.4 + kineticFactor*0.3) * 100
}

func (s *DEMSimulation) Step() {
	s.applyForcesParallel()
	collisions := s.detectCollisionsSpatial()
	s.resolveCollisionsParallel(collisions)
	for ci := range s.Cages {
		s.updateCageStability(ci)
	}
}

func (s *DEMSimulation) Run(steps int) {
	for step := 0; step < steps; step++ {
		s.Step()
	}
}

type MaintenanceSimulator struct {
	ctx    context.Context
	bus    chan<- msg.BusMessage
	config *CraftConfig
}

func NewMaintenanceSimulator(ctx context.Context, bus chan<- msg.BusMessage, config *CraftConfig) *MaintenanceSimulator {
	if config == nil {
		config = DefaultCraftConfig()
	}
	return &MaintenanceSimulator{ctx: ctx, bus: bus, config: config}
}

func (ms *MaintenanceSimulator) RunBambooCageSimulation(req msg.SimRequestPayload) ([]BambooCage, error) {
	sim := NewDEMSimulation(req.SimID, ms.config)
	cfg := ms.config.BambooCage
	baseZ := 726.5
	cages := make([]BambooCage, 0, req.CageCount)

	for i := 0; i < req.CageCount; i++ {
		row := i / cfg.CagesPerRow
		col := i % cfg.CagesPerRow
		x := (float64(col)-float64(cfg.CagesPerRow)/2)*cfg.CageSpacing - 2.4
		y := (float64(row)-1)*cfg.CageSpacing - 2.4
		z := baseZ

		cageID := ""
		switch req.Location {
		case "neijiang":
			cageID = "NJ-CAGE-"
		case "waijiang":
			cageID = "WJ-CAGE-"
		default:
			cageID = "CAGE-"
		}
		cageID += string(rune('A'+row)) + string(rune('0'+col))

		cage := sim.CreateBambooCage(cageID, x, y, z, cfg.DefaultDiameter, cfg.DefaultLength, cfg.StoneCountPerCage)
		sim.Run(cfg.ConvergenceSteps)
		cages = append(cages, *cage)

		depositionHeight := 0.0
		for _, stone := range cage.Stones {
			if stone.PositionZ > depositionHeight {
				depositionHeight = stone.PositionZ
			}
		}

		cageData := &models.BambooCageData{
			SimulationID: int(req.SimID), CageID: cage.CageID,
			PositionX: cage.PositionX, PositionY: cage.PositionY, PositionZ: cage.PositionZ,
			StoneCount: cfg.StoneCountPerCage, CageDiameter: cage.Diameter, CageLength: cage.Length,
			Porosity: cage.Porosity, StabilityCoefficient: cage.Stability,
			DepositionHeight: depositionHeight - baseZ,
		}
		if err := models.InsertBambooCageData(ms.ctx, cageData); err != nil {
			log.Printf("Failed to insert bamboo cage data: %v", err)
		}
	}

	ms.bus <- msg.BusMessage{
		Type: msg.TypeSimResult, Timestamp: time.Now(),
		Payload: msg.SimResultPayload{SimID: req.SimID, SimType: "bamboo_cage", Result: cages},
	}

	return cages, nil
}

func (ms *MaintenanceSimulator) RunMachaInterception(req msg.SimRequestPayload) ([]MachaStructure, []models.MachaInterceptionData, error) {
	cfg := ms.config.Macha
	sim := NewDEMSimulation(req.SimID, ms.config)
	baseZ := 726.5

	var machas []MachaStructure
	var interceptionData []models.MachaInterceptionData
	currentFlowRate := req.FlowRate
	currentWaterLevel := req.WaterLevel

	for step := 0; step < req.MachaCount; step++ {
		angle := cfg.DefaultAngle + float64(step)*cfg.AngleIncrement
		x := float64(step) * cfg.Spacing
		y := 0.0
		z := baseZ
		height := cfg.DefaultHeight + rand.Float64()*1.0
		logCount := cfg.LogCountMin + rand.Intn(cfg.LogCountMax-cfg.LogCountMin+1)

		macha := sim.CreateMacha(x, y, z, height, angle, logCount)
		efficiency := cfg.BaseEfficiency + cfg.EfficiencyIncrement*float64(step)
		if efficiency > cfg.MaxEfficiency {
			efficiency = cfg.MaxEfficiency
		}
		macha.Efficiency = efficiency

		flowReduction := currentFlowRate * efficiency
		newFlowRate := currentFlowRate - flowReduction
		waterLevelRise := (req.WaterLevel - currentWaterLevel) * 0.1 * efficiency
		newWaterLevel := currentWaterLevel + waterLevelRise

		record := models.MachaInterceptionData{
			Time: time.Now().Add(time.Duration(step) * time.Hour),
			SimulationID: int(req.SimID), PositionX: x, PositionY: y,
			WaterLevelBefore: currentWaterLevel, WaterLevelAfter: newWaterLevel,
			FlowRateBefore: currentFlowRate, FlowRateAfter: newFlowRate,
			InterceptionEfficiency: efficiency * 100, MachaCount: step + 1,
		}
		if err := models.InsertMachaInterceptionData(ms.ctx, &record); err != nil {
			log.Printf("Failed to insert macha data: %v", err)
		}

		machas = append(machas, *macha)
		interceptionData = append(interceptionData, record)
		currentFlowRate = newFlowRate
		currentWaterLevel = newWaterLevel
		sim.Run(100)
	}

	ms.bus <- msg.BusMessage{
		Type: msg.TypeSimResult, Timestamp: time.Now(),
		Payload: msg.SimResultPayload{SimID: req.SimID, SimType: "macha_interception", Result: machas},
	}

	return machas, interceptionData, nil
}
