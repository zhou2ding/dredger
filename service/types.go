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
		ShiftName       string         `json:"shiftName"`
		Parameters      ParameterStats `json:"parameters"`
		TotalEnergy     float64        `json:"-"`
		TotalProduction float64        `json:"-"`
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
		Min      float64 `json:"min"`
		Max      float64 `json:"max"`
		Average  float64 `json:"average"`
		Variance float64 `json:"variance"`
	}
)

type ColumnInfo struct {
	ColumnName        string `json:"columnName"`
	ColumnChineseName string `json:"columnChineseName"`
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
