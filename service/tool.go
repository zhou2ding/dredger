package service

import (
	"dredger/model"
	"math"
	"time"
)

const calHorizontalSpeedTimeDuration = 3 * 60 * 1000

func shiftName(shift int) string {
	switch shift {
	case 1:
		return "0-6"
	case 2:
		return "6-12"
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

func durationMinutesHl(minTime, maxTime time.Time, records []*model.DredgerDataHl) (time.Time, time.Time) {
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

func calParams(records []*model.DredgerDatum) ParameterStats {
	var (
		horizontalSpeeds              = make([]float64, len(records))
		carriageTravels               = make([]float64, len(records))
		cutterDepths                  = make([]float64, len(records))
		spumpRpms                     = make([]float64, len(records))
		concentrations                = make([]float64, len(records))
		flows                         = make([]float64, len(records))
		boosterPumpDischargePressures = make([]float64, len(records))
		vacuumDegrees                 = make([]float64, len(records))
		warning                       string
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
		boosterPumpDischargePressures[i] = r.BoosterPumpDischargePressure
		if r.OutputRate > 0 && r.TransverseSpeed == 0 {
			currentTime := r.RecordTime
			targetTime := currentTime + calHorizontalSpeedTimeDuration // 5分钟后的时间戳
			var nextRecord *model.DredgerDatum

			// 查找 5 分钟后的记录
			for j := i + 1; j < len(records); j++ {
				if records[j].RecordTime >= targetTime {
					nextRecord = records[j]
					break
				}
			}

			// 若无 3 分钟后的记录，使用最后一条记录
			if nextRecord == nil && len(records) > i+1 {
				nextRecord = records[len(records)-1]
			}

			if nextRecord != nil {
				// 计算两点间距离
				x1, y1 := r.CutterX, r.CutterY
				x2, y2 := nextRecord.CutterX, nextRecord.CutterY
				distance := math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))

				// 计算时间差（单位：秒）
				timeDiff := float64(nextRecord.RecordTime-currentTime) / 1000.0 / 60
				if timeDiff > 3 {
					timeDiff = 3 // 限制为 3 分钟
				}

				// 计算横移速度
				transverseSpeed := distance / timeDiff
				horizontalSpeeds[i] = transverseSpeed
				warning = "横移速度为0，已通过绞刀位置重新计算"
			} else {
				horizontalSpeeds[i] = r.TransverseSpeed
				warning = "存在产量非0，但是横移速度为0的数据，且无法计算，请检查传感器状态"
			}
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
	boosterPumpDischargePressure := calculateStats(boosterPumpDischargePressures)
	vacuumDegree := calculateStats(vacuumDegrees)

	horizontalSpeed.MaxProductionParam = round(horizontalSpeeds[maxIndex])
	carriageTravel.MaxProductionParam = round(carriageTravels[maxIndex])
	cutterDepth.MaxProductionParam = round(cutterDepths[maxIndex])
	sPumpRpm.MaxProductionParam = round(spumpRpms[maxIndex])
	concentration.MaxProductionParam = round(concentrations[maxIndex])
	flow.MaxProductionParam = round(flows[maxIndex])
	boosterPumpDischargePressure.MaxProductionParam = round(boosterPumpDischargePressures[maxIndex])

	return ParameterStats{
		HorizontalSpeed:              horizontalSpeed,
		CarriageTravel:               carriageTravel,
		CutterDepth:                  cutterDepth,
		SPumpRpm:                     sPumpRpm,
		Concentration:                concentration,
		Flow:                         flow,
		BoosterPumpDischargePressure: boosterPumpDischargePressure,
		VacuumDegree:                 vacuumDegree,
	}
}

func calParamsHl(records []*model.DredgerDataHl) ParameterStats {
	var (
		horizontalSpeeds                 = make([]float64, len(records))
		carriageTravels                  = make([]float64, len(records))
		cutterDepths                     = make([]float64, len(records))
		spumpRpms                        = make([]float64, len(records))
		concentrations                   = make([]float64, len(records))
		flows                            = make([]float64, len(records))
		underwaterPumpDischargePressures = make([]float64, len(records))
		vacuumDegrees                    = make([]float64, len(records))
		warning                          string
	)

	maxOutputRate := -1.0
	maxIndex := 0

	for i, r := range records {
		if r.HourlyOutputRate > maxOutputRate {
			maxOutputRate = r.HourlyOutputRate
			maxIndex = i
		}
		horizontalSpeeds[i] = r.TransverseSpeed
		carriageTravels[i] = r.TrolleyTravel
		cutterDepths[i] = r.BridgeDepth
		spumpRpms[i] = r.UnderwaterPumpSpeed
		concentrations[i] = r.Concentration
		flows[i] = r.FlowRate
		underwaterPumpDischargePressures[i] = r.UnderwaterPumpDischargePressure
		if r.HourlyOutputRate > 0 && r.TransverseSpeed == 0 {
			currentTime := r.RecordTime
			targetTime := currentTime + calHorizontalSpeedTimeDuration
			var nextRecord *model.DredgerDataHl

			for j := i + 1; j < len(records); j++ {
				if records[j].RecordTime >= targetTime {
					nextRecord = records[j]
					break
				}
			}

			if nextRecord == nil && len(records) > i+1 {
				nextRecord = records[len(records)-1]
			}

			if nextRecord != nil {
				x1, y1 := r.CutterX, r.CutterY
				x2, y2 := nextRecord.CutterX, nextRecord.CutterY
				distance := math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
				timeDiff := float64(nextRecord.RecordTime-currentTime) / 1000.0 / 60
				if timeDiff > 3 {
					timeDiff = 3
				}
				transverseSpeed := distance / timeDiff
				horizontalSpeeds[i] = transverseSpeed
				warning = "横移速度为0，已通过绞刀位置重新计算"
			} else {
				horizontalSpeeds[i] = r.TransverseSpeed
				warning = "存在产量非0，但是横移速度为0的数据，且无法计算，请检查传感器状态"
			}
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
	vacuumDegree := calculateStats(vacuumDegrees)
	underwaterPumpDischargePressure := calculateStats(underwaterPumpDischargePressures)

	horizontalSpeed.MaxProductionParam = round(horizontalSpeeds[maxIndex])
	carriageTravel.MaxProductionParam = round(carriageTravels[maxIndex])
	cutterDepth.MaxProductionParam = round(cutterDepths[maxIndex])
	sPumpRpm.MaxProductionParam = round(spumpRpms[maxIndex])
	concentration.MaxProductionParam = round(concentrations[maxIndex])
	flow.MaxProductionParam = round(flows[maxIndex])
	underwaterPumpDischargePressure.MaxProductionParam = round(flows[maxIndex])

	return ParameterStats{
		HorizontalSpeed:              horizontalSpeed,
		CarriageTravel:               carriageTravel,
		CutterDepth:                  cutterDepth,
		SPumpRpm:                     sPumpRpm,
		Concentration:                concentration,
		Flow:                         flow,
		BoosterPumpDischargePressure: underwaterPumpDischargePressure,
		VacuumDegree:                 vacuumDegree,
	}
}

// 统计计算通用函数
func calculateStats(data []float64) Parameter {
	if len(data) == 0 {
		return Parameter{}
	}
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
		Min:      round(minVal),
		Max:      round(maxVal),
		Average:  round(mean),
		Variance: round(variance),
	}
}

func round(x float64) float64 {
	return math.Round(x*100) / 100
}
