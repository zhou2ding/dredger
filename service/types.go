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

type DemoID int

const (
	Demo1 DemoID = 1
	Demo2 DemoID = 2
	Demo3 DemoID = 3
	Demo4 DemoID = 4
	Demo5 DemoID = 5
	Demo6 DemoID = 6
)

type DemoParams struct {
	GeoPath   string // 保存后的文件名，后端负责落盘；也允许前端传自定义名
	BrdPath   string
	DesignXYZ string
	MudXYZ    string
	RefZ      float64
	GridXY    float64
	GridZ     float64
	// Demo3/4
	CX     float64
	CY     float64
	Length float64
	Width  float64
	Depth  float64
	Height float64
	// Demo6
	X1        float64
	Y1        float64
	X2        float64
	Y2        float64
	Threshold float64
}

type GeneratedFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Mod  int64  `json:"mod"` // UnixMilli
	Ext  string `json:"ext"`
}

type ExecutionLogEntry struct {
	Timestamp int64           `json:"timestamp"`
	Files     []GeneratedFile `json:"files"`
}

type PlaybackData struct {
	Timestamps             []int64   `json:"timestamps"`
	ActualVacuum           []float64 `json:"actualVacuum"`
	EstimatedVacuum        []float64 `json:"estimatedVacuum"`
	FlowRate               []float64 `json:"flowRate"`
	Concentration          []float64 `json:"concentration"`
	ProductionRate         []float64 `json:"productionRate"`
	LadderDepth            []float64 `json:"ladderDepth"`
	CutterRpm              []float64 `json:"cutterRpm"`
	SubmergedPumpRpm       []float64 `json:"submergedPumpRpm"`
	MudPump1Rpm            []float64 `json:"mudPump1Rpm"`
	MudPump2Rpm            []float64 `json:"mudPump2Rpm"`
	SubmergedPumpDischarge []float64 `json:"submergedPumpDischarge"`
	MudPump1Discharge      []float64 `json:"mudPump1Discharge"`
	MudPump2Discharge      []float64 `json:"mudPump2Discharge"`
	GpsSpeed               []float64 `json:"gpsSpeed"`
}
