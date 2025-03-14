package service

import (
	"dredger/model"
	"math"
	"time"
)

func shiftName(shift int) string {
	switch shift {
	case 1:
		return "0-6"
	case 2:
		return "0-12"
	case 3:
		return "12-18"
	default:
		return "18-24"
	}
}

func durationMinutes(minTime, maxTime time.Time, records []model.DredgerDatum) float64 {
	for i, r := range records {
		t := time.UnixMilli(r.RecordTime)
		if i == 0 || t.Before(minTime) {
			minTime = t
		}
		if i == 0 || t.After(maxTime) {
			maxTime = t
		}
	}
	return maxTime.Sub(minTime).Minutes()
}

// 核心参数统计
func calculateParameters(records []model.DredgerDatum) ParameterStats {
	var (
		horizontalSpeeds = make([]float64, len(records))
		carriageTravels  = make([]float64, len(records))
		cutterDepths     = make([]float64, len(records))
		spumpRpms        = make([]float64, len(records))
		concentrations   = make([]float64, len(records))
		flows            = make([]float64, len(records))
		warning          string
	)

	for i, r := range records {
		horizontalSpeeds[i] = r.TransverseSpeed
		carriageTravels[i] = r.TrolleyTravel
		cutterDepths[i] = r.CutterDepth
		spumpRpms[i] = r.UnderwaterPumpSpeed
		concentrations[i] = r.Concentration
		flows[i] = r.FlowRate
		if r.OutputRate > 0 && r.TransverseSpeed == 0 {
			warning = "检测到横移速度传感器异常"
		}
	}

	return ParameterStats{
		HorizontalSpeed: HorizontalSpeed{
			Parameter: calculateStats(horizontalSpeeds),
			Warning:   warning,
		},
		CarriageTravel: calculateStats(carriageTravels),
		CutterDepth:    calculateStats(cutterDepths),
		SPumpRpm:       calculateStats(spumpRpms),
		Concentration:  calculateStats(concentrations),
		Flow:           calculateStats(flows),
	}
}

// 统计计算通用函数
func calculateStats(data []float64) Parameter {
	var sum, sumSquares float64
	minVal, maxVal := data[0], data[0]
	for _, v := range data {
		sum += v
		sumSquares += v * v
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	n := float64(len(data))
	mean := sum / n
	variance := (sumSquares / n) - (mean * mean)

	return Parameter{
		Min:      int(math.Round(minVal)),
		Max:      int(math.Round(maxVal)),
		Average:  int(math.Round(mean)),
		Variance: int(math.Round(variance)),
	}
}

// 能耗计算
func calculateEnergy(stats ParameterStats, duration float64) float64 {
	P1 := stats.HorizontalSpeed.Average
	P2 := stats.SPumpRpm.Average
	P3 := stats.Concentration.Average
	Q := stats.Flow.Average

	pw1 := 0.8 * float64(Q) * (float64(P2 - P1))
	pw2 := 0.8 * float64(Q) * (float64(P3 - P2))
	return (pw1 + pw2) * (duration / 60)
}

// 最优班次更新逻辑
func updateOptimalShift(optimal *OptimalShift, shift int, value float64, stats ParameterStats, metricType string) {
	switch metricType {
	case "production":
		if value > optimal.TotalProduction {
			optimal.TotalProduction = value
			optimal.ShiftName = shiftName(shift)
			optimal.Parameters = stats
		}
	case "energy":
		if value < optimal.TotalEnergy || optimal.TotalEnergy == 0 {
			optimal.TotalEnergy = value
			optimal.ShiftName = shiftName(shift)
			optimal.Parameters = stats
		}
	}
}
