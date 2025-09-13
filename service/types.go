package service

import "time"

const (
	MaxProduction = "maxProduction"
	MinEnergy     = "minEnergy"
)

type ImportDataResult struct {
	ImportedRows int `json:"importedRows"`
}

type ShiftStat struct {
	ShiftName       string    `json:"shiftName"`
	BeginTime       time.Time `json:"beginTime"`
	EndTime         time.Time `json:"endTime"`
	WorkDuration    float64   `json:"workDuration"`
	TotalProduction float64   `json:"totalProduction"`
	TotalEnergy     float64   `json:"totalEnergy"`
}

type ParameterStat struct {
	Mean     float64  `json:"mean"`
	Variance float64  `json:"variance"`
	Warnings []string `json:"warnings,omitempty"`
}

type (
	OptimalShift struct {
		MaxProductionShift *ShiftWorkParams `json:"maxProductionShift"`
		MinEnergyShift     *ShiftWorkParams `json:"minEnergyShift"`
		TotalEnergy        float64          `json:"-"`
		TotalProduction    float64          `json:"-"`
	}
	ShiftWorkParams struct {
		ShiftName  string         `json:"shiftName"`
		Parameters ParameterStats `json:"parameters"`
	}
	ParameterStats struct {
		Flow                         Parameter       `json:"flow"`
		Concentration                Parameter       `json:"concentration"`
		SPumpRpm                     Parameter       `json:"sPumpRpm"`
		CutterDepth                  Parameter       `json:"cutterDepth"`
		CarriageTravel               Parameter       `json:"carriageTravel"`
		HorizontalSpeed              HorizontalSpeed `json:"horizontalSpeed"`
		BoosterPumpDischargePressure Parameter       `json:"boosterPumpDischargePressure"`
		VacuumDegree                 Parameter       `json:"vacuumDegree"`
	}
	HorizontalSpeed struct {
		Parameter
		Warning string `json:"warning"`
	}
	Parameter struct {
		Min                float64 `json:"min"`
		Max                float64 `json:"max"`
		Average            float64 `json:"average"`
		Variance           float64 `json:"variance"`
		MaxProductionParam float64 `json:"maxProductionParam"`
	}
)

type ColumnInfo struct {
	ColumnName        string `json:"columnName"`
	ColumnChineseName string `json:"columnChineseName"`
	ColumnUnit        string `json:"columnUnit"`
}

type (
	ShiftPie struct {
		ShiftName string   `json:"shiftName"`
		WorkData  *PieData `json:"workData"`
	}
	PieData struct {
		TotalProduction float64 `json:"totalProduction"`
		TotalEnergy     float64 `json:"totalEnergy"`
		WorkDuration    float64 `json:"workDuration"`
	}
)

type ColumnData struct {
	Timestamp string `json:"timestamp"`
	Value     any    `json:"value"`
}

type GlobalTimeRange struct {
	ShipName     string `json:"shipName"`
	StartDate    int64  `json:"-"`
	EndDate      int64  `json:"-"`
	StartDateStr string `json:"startDate"`
	EndDateStr   string `json:"endDate"`
}

type TheoryOptimalParamsDTO struct {
	ID                           int64     `json:"id"`
	CreatedAt                    time.Time `json:"createdAt"`
	UpdatedAt                    time.Time `json:"updatedAt"`
	ShipName                     string    `json:"shipName"`
	Flow                         float64   `json:"flow"`
	Concentration                float64   `json:"concentration"`
	SPumpRpm                     float64   `json:"sPumpRpm"`
	CutterDepth                  float64   `json:"cutterDepth"`
	CarriageTravel               float64   `json:"carriageTravel"`
	HorizontalSpeed              float64   `json:"horizontalSpeed"`
	BoosterPumpDischargePressure float64   `json:"boosterPumpDischargePressure"`
	VacuumDegree                 float64   `json:"vacuumDegree"`
}

type ExecutionParams struct {
	Action                  string
	GeologyDbFile           string
	CalculationBoundaryFile string
	DesignDepthFile         string
	MudlineFile             string
	ReferenceZ              float64
	GridDistanceXY          float64
	GridDistanceZ           float64
	Threshold               float64
	PileX                   float64
	PileY                   float64
	ProfileX1               float64
	ProfileY1               float64
	ProfileX2               float64
	ProfileY2               float64
	SpecifiedX              float64
	SpecifiedY              float64
	SpecifiedLength         float64
	SpecifiedWidth          float64
}

type SolidResult map[string]any
