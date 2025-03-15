package service

import (
	"dredger/model"
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

func durationMinutes(minTime, maxTime time.Time, records []*model.DredgerDatum) (time.Time, time.Time) {
	for i, r := range records {
		t := time.UnixMilli(r.RecordTime)
		if i == 0 || t.Before(minTime) {
			minTime = t
		}
		if i == 0 || t.After(maxTime) {
			maxTime = t
		}
	}
	return maxTime, minTime
}

// 核心参数统计
func calParams(records []*model.DredgerDatum) ParameterStats {
	var (
		horizontalSpeeds = make([]float64, len(records))
		carriageTravels  = make([]float64, len(records))
		cutterDepths     = make([]float64, len(records))
		spumpRpms        = make([]float64, len(records))
		concentrations   = make([]float64, len(records))
		flows            = make([]float64, len(records))
		warning          string
	)

	maxOutputRate := -1.0
	maxIndex := 0

	for i, r := range records {
		if r.OutputRate > maxOutputRate {
			maxOutputRate = r.OutputRate
			maxIndex = i
		}
		horizontalSpeeds[i] = r.TransverseSpeed
		carriageTravels[i] = r.TrolleyTravel
		cutterDepths[i] = r.CutterDepth
		spumpRpms[i] = r.UnderwaterPumpSpeed
		concentrations[i] = r.Concentration
		flows[i] = r.FlowRate
		if r.OutputRate > 0 && r.TransverseSpeed == 0 {
			warning = "存在产量非0，但是横移速度为0的数据，请检查传感器状态"
		}
	}

	horizontalSpeed := HorizontalSpeed{
		Parameter: calculateStats(horizontalSpeeds),
		Warning:   warning,
	}
	carriageTravel := calculateStats(carriageTravels)
	cutterDepth := calculateStats(cutterDepths)
	sPumpRpm := calculateStats(spumpRpms)
	concentration := calculateStats(concentrations)
	flow := calculateStats(flows)

	horizontalSpeed.MaxProductionParam = horizontalSpeeds[maxIndex]
	carriageTravel.MaxProductionParam = carriageTravels[maxIndex]
	cutterDepth.MaxProductionParam = cutterDepths[maxIndex]
	sPumpRpm.MaxProductionParam = spumpRpms[maxIndex]
	concentration.MaxProductionParam = concentrations[maxIndex]
	flow.MaxProductionParam = flows[maxIndex]

	return ParameterStats{
		HorizontalSpeed: horizontalSpeed,
		CarriageTravel:  carriageTravel,
		CutterDepth:     cutterDepth,
		SPumpRpm:        sPumpRpm,
		Concentration:   concentration,
		Flow:            flow,
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
		Min:      minVal,
		Max:      maxVal,
		Average:  mean,
		Variance: variance,
	}
}
