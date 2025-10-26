package service

import (
	"dredger/model"
	"math"
	"path/filepath"
	"strings"
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

// 计算华安龙的 "按天" 累计施工时长 (分钟)
func calculateAccumulatedDurationHl(records []*model.DredgerDataHl) float64 {
	// 1. 按 "天" (日期字符串) 分组
	recordsByDay := make(map[string][]*model.DredgerDataHl)
	for _, r := range records {
		dayKey := time.UnixMilli(r.RecordTime).Format("2006-01-02")
		recordsByDay[dayKey] = append(recordsByDay[dayKey], r)
	}

	var totalDuration time.Duration

	// 2. 遍历每一天的数据
	for _, dayRecords := range recordsByDay {
		if len(dayRecords) < 2 { // 当天数据不足2条，无法计算时长
			continue
		}

		// 3. 找到这一天的 minTime 和 maxTime
		var dayMinTime, dayMaxTime time.Time
		for i, r := range dayRecords {
			t := time.UnixMilli(r.RecordTime)
			if i == 0 || t.Before(dayMinTime) {
				dayMinTime = t
			}
			if i == 0 || t.After(dayMaxTime) {
				dayMaxTime = t
			}
		}

		// 4. 累加这一天的时长
		totalDuration += dayMaxTime.Sub(dayMinTime)
	}

	// 5. 返回总分钟数
	return totalDuration.Minutes()
}

// 计算敏龙的 "按天" 累计施工时长 (分钟)
func calculateAccumulatedDuration(records []*model.DredgerDatum) float64 {
	// 1. 按 "天" (日期字符串) 分组
	recordsByDay := make(map[string][]*model.DredgerDatum)
	for _, r := range records {
		dayKey := time.UnixMilli(r.RecordTime).Format("2006-01-02")
		recordsByDay[dayKey] = append(recordsByDay[dayKey], r)
	}

	var totalDuration time.Duration

	// 2. 遍历每一天的数据
	for _, dayRecords := range recordsByDay {
		if len(dayRecords) < 2 { // 当天数据不足2条，无法计算时长
			continue
		}

		// 3. 找到这一天的 minTime 和 maxTime
		var dayMinTime, dayMaxTime time.Time
		for i, r := range dayRecords {
			t := time.UnixMilli(r.RecordTime)
			if i == 0 || t.Before(dayMinTime) {
				dayMinTime = t
			}
			if i == 0 || t.After(dayMaxTime) {
				dayMaxTime = t
			}
		}

		// 4. 累加这一天的时长
		totalDuration += dayMaxTime.Sub(dayMinTime)
	}

	// 5. 返回总分钟数
	return totalDuration.Minutes()
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

func calParams(records []*model.DredgerDatum) (ParameterStats, int64) {
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

	cfg := GetCfg(records[0].ShipName)

	maxOutputRate := -1.0
	maxIndex := 0

	for i, r := range records {
		if r.CurrentShiftOutputRate > maxOutputRate {
			maxOutputRate = r.CurrentShiftOutputRate
			maxIndex = i
		}
		carriageTravels[i] = r.TrolleyTravel
		cutterDepths[i] = -r.CutterDepth
		spumpRpms[i] = r.UnderwaterPumpSpeed
		concentrations[i] = r.Concentration
		flows[i] = r.FlowRate
		boosterPumpDischargePressures[i] = r.BoosterPumpDischargePressure
		vacuumDegrees[i] = calcVacuumKPa(r, cfg)
		if r.CurrentShiftOutputRate > 0 && r.TransverseSpeed == 0 {
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
				horizontalSpeeds[i] = math.Abs(r.TransverseSpeed)
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
	vacuumDegree.MaxProductionParam = round(vacuumDegrees[maxIndex])

	var optimalTime int64
	if len(records) > maxIndex {
		optimalTime = records[maxIndex].RecordTime
	}
	return ParameterStats{
		HorizontalSpeed:              horizontalSpeed,
		CarriageTravel:               carriageTravel,
		CutterDepth:                  cutterDepth,
		SPumpRpm:                     sPumpRpm,
		Concentration:                concentration,
		Flow:                         flow,
		BoosterPumpDischargePressure: boosterPumpDischargePressure,
		VacuumDegree:                 vacuumDegree,
	}, optimalTime
}

func calParamsHl(records []*model.DredgerDataHl) (ParameterStats, int64) {
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
	cfg := GetCfg(records[0].ShipName)

	for i, r := range records {
		if r.HourlyOutputRate > maxOutputRate {
			maxOutputRate = r.HourlyOutputRate
			maxIndex = i
		}
		carriageTravels[i] = r.TrolleyTravel
		cutterDepths[i] = -r.BridgeDepth
		spumpRpms[i] = r.UnderwaterPumpSpeed
		concentrations[i] = r.Concentration
		flows[i] = r.FlowRate
		var pumpPressure float64
		if r.MudPump2DischargePressure != 0 {
			pumpPressure = r.MudPump2DischargePressure
		} else if r.MudPump1DischargePressure != 0 {
			pumpPressure = r.MudPump1DischargePressure
		} else if r.UnderwaterPumpDischargePressure != 0 {
			pumpPressure = r.UnderwaterPumpDischargePressure
		}
		underwaterPumpDischargePressures[i] = pumpPressure
		vacuumDegrees[i] = CalcVacuumKPaFromHL(r, cfg)
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
				horizontalSpeeds[i] = math.Abs(r.TransverseSpeed)
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
	vacuumDegree.MaxProductionParam = round(vacuumDegrees[maxIndex])

	var optimalTime int64
	if len(records) > maxIndex {
		optimalTime = records[maxIndex].RecordTime
	}

	return ParameterStats{
		HorizontalSpeed:              horizontalSpeed,
		CarriageTravel:               carriageTravel,
		CutterDepth:                  cutterDepth,
		SPumpRpm:                     sPumpRpm,
		Concentration:                concentration,
		Flow:                         flow,
		BoosterPumpDischargePressure: underwaterPumpDischargePressure,
		VacuumDegree:                 vacuumDegree,
	}, optimalTime
}

// 统计计算通用函数
func calculateStats(data []float64) Parameter {
	valid := make([]float64, 0, len(data))
	for _, v := range data {
		if !math.IsNaN(v) {
			valid = append(valid, v)
		}
	}
	if len(valid) == 0 {
		return Parameter{}
	}
	var sum, sumSquares float64
	minVal, maxVal := valid[0], valid[0]
	for _, v := range valid {
		sum += v
		sumSquares += v * v
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	n := float64(len(valid))
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

// 去掉收集到的 Windows 字符串路径上可能的引号（前端/复制粘贴容易带）
func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	s = strings.TrimPrefix(s, `'`)
	s = strings.TrimSuffix(s, `'`)
	return s
}

// 把 p 规范化为绝对路径：
// - 如果 p 本来就是绝对路径，直接 Clean 后返回
// - 如果是相对路径，则认为它应该在 dataDir 下面（也就是 ./pys/data），Join 后返回绝对路径
func makeAbsUnder(p string, dataDir string) string {
	if p == "" {
		return ""
	}
	p = stripQuotes(p)
	// Windows 下判断绝对路径：有盘符或 UNC（\\server\share）
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	// 相对路径的场景（例如 ".\\pys\\data\\test1.mdb" 或 "pys\\data\\test1.mdb"）
	// 统一接到 dataDir 下
	return filepath.Clean(filepath.Join(dataDir, p))
}

// 统计“敏龙”(DredgerDatum) 某个班组的平均真空度（kPa）；忽略 NaN/Inf
func averageVacuumDatum(records []*model.DredgerDatum, cfg ShipHydraulicsConfig) (avg float64, ok bool) {
	var sum float64
	var n int
	for _, r := range records {
		v := calcVacuumKPa(r, cfg)
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

// CalcVacuumKPaFromHL 把华安龙记录按字段“映射”为 DredgerDatum 再复用 calcVacuumKPa（字段名按你库实际改）
// 假设 Hl 记录具备：WaterDensity/Density/FieldSlurryDensity/FlowVelocity/FlowRate/MudPipeDiameter
// 以及用于几何的：BridgeDepth(当作吸口深度)/EarDraft/LeftEarDraft/RightEarDraft/EarToBottomDistance
func CalcVacuumKPaFromHL(r *model.DredgerDataHl, cfg ShipHydraulicsConfig) float64 {
	d := &model.DredgerDatum{
		ShipName:            r.ShipName,
		WaterDensity:        r.WaterDensity,
		Density:             r.Density,
		FieldSlurryDensity:  r.FieldSlurryDensity,
		FlowVelocity:        r.FlowVelocity,
		FlowRate:            r.FlowRate,
		CutterDepth:         r.BridgeDepth, // Hl 下用 BridgeDepth 作为吸口深度
		EarDraft:            r.EarDraft,
		LeftEarDraft:        r.LeftEarDraft,
		RightEarDraft:       r.RightEarDraft,
		MudPipeDiameter:     0.7,  // 泥管直径使用敏龙的数据
		EarToBottomDistance: 12.9, // 耳轴到船底距离由敏龙的耳轴吃水➗耳轴到船底的距离计算出来的比例，✖️华龙的耳轴吃水
	}
	return calcVacuumKPa(d, cfg)
}

// 统计“华安龙”(DredgerDataHl) 某个班组的平均真空度（kPa）；忽略 NaN/Inf
func averageVacuumHL(records []*model.DredgerDataHl, cfg ShipHydraulicsConfig) (avg float64, ok bool) {
	var sum float64
	var n int
	for _, r := range records {
		v := CalcVacuumKPaFromHL(r, cfg)
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

func findSoilType(x, y, z float64, regions []model.SoilRegion) string {
	for _, region := range regions {
		// cutter_y (传入的 y) 对应 soil_regions 的 x 坐标
		// cutter_x (传入的 x) 对应 soil_regions 的 y 坐标
		if y >= region.XMin && y < region.XMax &&
			x >= region.YMin && x < region.YMax &&
			z >= region.ZMin && z < region.ZMax {
			return region.SoilType
		}
	}
	return "未知土质" // 如果没有找到匹配的区域
}
