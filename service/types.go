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
		HorizontalSpeed HorizontalSpeed `json:"horizontalSpeed"`
		CarriageTravel  Parameter       `json:"carriageTravel"`
		CutterDepth     Parameter       `json:"cutterDepth"`
		SPumpRpm        Parameter       `json:"sPumpRpm"`
		Concentration   Parameter       `json:"concentration"`
		Flow            Parameter       `json:"flow"`
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
