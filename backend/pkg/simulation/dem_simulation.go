package simulation

import (
	"context"
	"log"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"dujiangyan-system/pkg/models"
)

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
	return &SpatialHashGrid{
		cellSize: cellSize,
		cells:    make(map[int64][]int),
	}
}

func (g *SpatialHashGrid) cellKey(cx, cy, cz int) int64 {
	prime1 := int64(73856093)
	prime2 := int64(19349663)
	prime3 := int64(83492791)
	return int64(cx)*prime1 ^ int64(cy)*prime2 ^ int64(cz)*prime3
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
	Gravity      float64
	Restitution  float64
	Friction     float64
	Viscosity    float64
	TimeStep     float64
	Stones       []StoneParticle
	Cages        []BambooCage
	Machas       []MachaStructure
	spatialGrid  *SpatialHashGrid
	workers      int
}

func NewDEMSimulation(simID int64) *DEMSimulation {
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	return &DEMSimulation{
		SimulationID: simID,
		Gravity:      -9.81,
		Restitution:  0.3,
		Friction:     0.6,
		Viscosity:    0.98,
		TimeStep:     0.01,
		spatialGrid:  newSpatialHashGrid(2.0),
		workers:      workers,
	}
}

func (s *DEMSimulation) AddStone(x, y, z, radius float64, fixed bool) int {
	stone := StoneParticle{
		ID:        len(s.Stones),
		PositionX: x,
		PositionY: y,
		PositionZ: z,
		VelocityX: 0,
		VelocityY: 0,
		VelocityZ: 0,
		Radius:    radius,
		Mass:      (4.0 / 3.0) * math.Pi * math.Pow(radius, 3) * 2650,
		Fixed:     fixed,
	}
	s.Stones = append(s.Stones, stone)
	return stone.ID
}

func (s *DEMSimulation) CreateBambooCage(
	cageID string, x, y, z, diameter, length float64, stoneCount int,
) *BambooCage {
	cage := BambooCage{
		CageID:    cageID,
		PositionX: x,
		PositionY: y,
		PositionZ: z,
		Diameter:  diameter,
		Length:    length,
		Porosity:  0.35,
		Stability: 0.0,
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

	s.spatialGrid.cellSize = math.Max(diameter*1.5, 2.0)

	s.Cages = append(s.Cages, cage)
	return &s.Cages[len(s.Cages)-1]
}

func (s *DEMSimulation) CreateMacha(
	x, y, z, height, angle float64, logCount int,
) *MachaStructure {
	macha := MachaStructure{
		ID:              len(s.Machas),
		PositionX:       x,
		PositionY:       y,
		PositionZ:       z,
		Height:          height,
		Angle:           angle,
		LogCount:        logCount,
		BindingStrength: 0.8,
		Efficiency:      0.0,
	}

	s.Machas = append(s.Machas, macha)
	return &s.Machas[len(s.Machas)-1]
}

func (s *DEMSimulation) detectCollision(i, j int) (bool, float64, [3]float64) {
	si := &s.Stones[i]
	sj := &s.Stones[j]

	dx := sj.PositionX - si.PositionX
	dy := sj.PositionY - si.PositionY
	dz := sj.PositionZ - si.PositionZ
	distanceSq := dx*dx + dy*dy + dz*dz
	minDist := si.Radius + sj.Radius

	if distanceSq < minDist*minDist && distanceSq > 0 {
		distance := math.Sqrt(distanceSq)
		overlap := minDist - distance
		nx := dx / distance
		ny := dy / distance
		nz := dz / distance
		return true, overlap, [3]float64{nx, ny, nz}
	}

	return false, 0, [3]float64{}
}

func (s *DEMSimulation) resolveCollision(i, j int, overlap float64, normal [3]float64) {
	si := &s.Stones[i]
	sj := &s.Stones[j]

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

	impulse := -(1 + s.Restitution) * velAlongNormal / (1/si.Mass + 1/sj.Mass)

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

				s.Stones[i].VelocityZ += s.Gravity * s.TimeStep

				s.Stones[i].VelocityX *= s.Viscosity
				s.Stones[i].VelocityY *= s.Viscosity
				s.Stones[i].VelocityZ *= s.Viscosity

				s.Stones[i].PositionX += s.Stones[i].VelocityX * s.TimeStep
				s.Stones[i].PositionY += s.Stones[i].VelocityY * s.TimeStep
				s.Stones[i].PositionZ += s.Stones[i].VelocityZ * s.TimeStep

				if s.Stones[i].PositionZ < s.Stones[i].Radius {
					s.Stones[i].PositionZ = s.Stones[i].Radius
					if s.Stones[i].VelocityZ < 0 {
						s.Stones[i].VelocityZ = -s.Stones[i].VelocityZ * s.Restitution
					}

					frictionForce := s.Friction * s.Gravity * s.Stones[i].Mass * s.TimeStep
					speed := math.Sqrt(s.Stones[i].VelocityX*s.Stones[i].VelocityX +
						s.Stones[i].VelocityY*s.Stones[i].VelocityY)
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

	var actualCollisions []collisionPair
	for _, pair := range potentialPairs {
		colliding, overlap, normal := s.detectCollision(pair.i, pair.j)
		if colliding {
			actualCollisions = append(actualCollisions, collisionPair{
				i: pair.i, j: pair.j, overlap: overlap, normal: normal,
			})
		}
	}

	return actualCollisions
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
		go func(startIdx, endIdx int) {
			defer wg.Done()
			for k := startIdx; k < endIdx; k++ {
				c := collisions[k]
				s.resolveCollision(c.i, c.j, c.overlap, c.normal)
			}
		}(start, end)
	}

	wg.Wait()
}

func (s *DEMSimulation) Step() {
	s.applyForcesParallel()

	collisions := s.detectCollisionsSpatial()

	s.resolveCollisionsParallel(collisions)

	for ci := range s.Cages {
		s.updateCageStability(ci)
	}
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
		ke := 0.5 * stone.Mass * (stone.VelocityX*stone.VelocityX +
			stone.VelocityY*stone.VelocityY + stone.VelocityZ*stone.VelocityZ)
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

func (s *DEMSimulation) Run(steps int) {
	for step := 0; step < steps; step++ {
		s.Step()
	}
}

func SimulateBambooCagePlacement(
	ctx context.Context,
	simulationID int64,
	location string,
	cageCount int,
) ([]BambooCage, error) {

	sim := NewDEMSimulation(simulationID)

	baseX := 0.0
	baseY := 0.0
	baseZ := 726.5

	cages := make([]BambooCage, 0, cageCount)

	for i := 0; i < cageCount; i++ {
		row := i / 5
		col := i % 5

		x := baseX + float64(col)*1.2 - 2.4
		y := baseY + float64(row)*1.2 - 2.4
		z := baseZ

		stoneCount := 80 + rand.Intn(40)
		diameter := 0.8 + rand.Float64()*0.4
		length := 2.0 + rand.Float64()*1.0

		cageID := ""
		switch location {
		case "neijiang":
			cageID = "NJ-CAGE-"
		case "waijiang":
			cageID = "WJ-CAGE-"
		default:
			cageID = "CAGE-"
		}
		cageID += string(rune('A'+row)) + string(rune('0'+col))

		cage := sim.CreateBambooCage(cageID, x, y, z, diameter, length, stoneCount)
		sim.Run(500)

		cages = append(cages, *cage)

		depositionHeight := 0.0
		for _, stone := range cage.Stones {
			if stone.PositionZ > depositionHeight {
				depositionHeight = stone.PositionZ
			}
		}

		cageData := &models.BambooCageData{
			SimulationID:         int(simulationID),
			CageID:               cage.CageID,
			PositionX:            cage.PositionX,
			PositionY:            cage.PositionY,
			PositionZ:            cage.PositionZ,
			StoneCount:           stoneCount,
			CageDiameter:         cage.Diameter,
			CageLength:           cage.Length,
			Porosity:             cage.Porosity,
			StabilityCoefficient: cage.Stability,
			DepositionHeight:     depositionHeight - baseZ,
		}

		if err := models.InsertBambooCageData(ctx, cageData); err != nil {
			log.Printf("Failed to insert bamboo cage data: %v", err)
		}
	}

	return cages, nil
}

func SimulateMachaInterception(
	ctx context.Context,
	simulationID int64,
	location string,
	machaCount int,
	initialFlowRate, initialWaterLevel float64,
) ([]MachaStructure, []models.MachaInterceptionData, error) {

	sim := NewDEMSimulation(simulationID)

	baseX := 0.0
	baseY := 0.0
	baseZ := 726.5

	var machas []MachaStructure
	var interceptionData []models.MachaInterceptionData

	currentFlowRate := initialFlowRate
	currentWaterLevel := initialWaterLevel

	for step := 0; step < machaCount; step++ {
		angle := 15.0 + float64(step)*2.0
		x := baseX + float64(step)*3.0
		y := baseY
		z := baseZ
		height := 4.0 + rand.Float64()*1.0
		logCount := 6 + rand.Intn(4)

		macha := sim.CreateMacha(x, y, z, height, angle, logCount)

		efficiency := 0.05 + 0.02*float64(step)
		if efficiency > 0.85 {
			efficiency = 0.85
		}
		macha.Efficiency = efficiency

		flowReduction := currentFlowRate * efficiency
		newFlowRate := currentFlowRate - flowReduction

		waterLevelRise := (initialWaterLevel - currentWaterLevel) * 0.1 * efficiency
		newWaterLevel := currentWaterLevel + waterLevelRise

		interceptionRecord := models.MachaInterceptionData{
			Time:                   time.Now().Add(time.Duration(step) * time.Hour),
			SimulationID:           int(simulationID),
			PositionX:              x,
			PositionY:              y,
			WaterLevelBefore:       currentWaterLevel,
			WaterLevelAfter:        newWaterLevel,
			FlowRateBefore:         currentFlowRate,
			FlowRateAfter:          newFlowRate,
			InterceptionEfficiency: efficiency * 100,
			MachaCount:             step + 1,
		}

		if err := models.InsertMachaInterceptionData(ctx, &interceptionRecord); err != nil {
			log.Printf("Failed to insert macha interception data: %v", err)
		}

		machas = append(machas, *macha)
		interceptionData = append(interceptionData, interceptionRecord)

		currentFlowRate = newFlowRate
		currentWaterLevel = newWaterLevel

		sim.Run(100)
	}

	return machas, interceptionData, nil
}
