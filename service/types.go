package service

import "time"

const (
	MaxProduction = "maxProduction"
	MinEnergy     = "minEnergy"
)

type ImportDataResult struct {
	ImportedRows int `json:"importedRows"`
}

type (
	ShiftStat struct {
		ShiftName       string    `json:"shiftName"`
		BeginTime       time.Time `json:"beginTime"`
		EndTime         time.Time `json:"endTime"`
		WorkDuration    float64   `json:"workDuration"`
		TotalProduction float64   `json:"totalProduction"`
		TotalEnergy     float64   `json:"totalEnergy"`
	}
	Key struct {
		Date  string
		Shift int
	}
)

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
		Min      int `json:"min"`
		Max      int `json:"max"`
		Average  int `json:"average"`
		Variance int `json:"variance"`
	}
)

type ReportParams struct {
	ShipName  string      `json:"ship_name"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Shifts    []ShiftStat `json:"shifts"`
}
