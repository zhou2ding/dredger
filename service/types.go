package service

import "time"

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

type ParameterStats struct {
	SwingSpeed     ParameterStat `json:"swing_speed"`
	CarriageTravel ParameterStat `json:"carriage_travel"`
	CutterDepth    ParameterStat `json:"cutter_depth"`
	PumpRPM        ParameterStat `json:"pump_rpm"`
	Concentration  ParameterStat `json:"concentration"`
	FlowRate       ParameterStat `json:"flow_rate"`
}

type OptimalAnalysis struct {
	OptimalShift string         `json:"optimal_shift"`
	Parameters   ParameterStats `json:"parameters"`
}

type ReportParams struct {
	ShipName  string      `json:"ship_name"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Shifts    []ShiftStat `json:"shifts"`
}
