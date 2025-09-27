package service

import (
	"dredger/model"
	"math"
)

type ShipHydraulicsConfig struct {
	PatmPa                   float64 // 大气压，Pa（默认 101325）
	G                        float64 // 重力加速度，m/s^2（默认 9.80665）
	PipeInnerDiameterM       float64 // 吸入管内径 m（若数据表已有，则此项可为空）
	SuctionPipeLengthM       float64 // 直管长度 m（来自 Word/设备台账）
	LocalEqLengthM           float64 // 局部件当量长度 m（来自 Word 表格折算）
	FrictionFactorClearWater float64 // 清水沿程阻力系数 f_cw（或直接填泥浆用 f）
	UseDensityRatio          bool    // 是否用 (rho_m/rho_w) 放大 f_cw 得到泥浆 f
	PumpAboveBottomM         float64 // 泵中心线高于船底的高度 m（来自布置）
	FlowRateUnit             string  // "m3/h" 或 "m3/s"
	DensityUnit              string  // "kg/m3" / "t/m3" / "g/cm3"
	VacuumOutUnit            string  // "kPa"（默认）
}

// TODO：把这些数值替换为你 Word“图片表格”的实际值
var shipCfg = map[string]ShipHydraulicsConfig{
	"敏龙": {
		PatmPa:                   101325,
		G:                        9.80665,
		PipeInnerDiameterM:       0,      // 若数据表 MudPipeDiameter 有值，可留 0
		SuctionPipeLengthM:       0,      // ← Word 表格
		LocalEqLengthM:           0,      // ← Word 表格
		FrictionFactorClearWater: 0.02,   // 例：按规范/表取值，或置 0 后用泥浆 f 直填
		UseDensityRatio:          true,   // 按 Word 思路：清水 f × 密度比
		PumpAboveBottomM:         0,      // ← 布置尺寸（图片表）
		FlowRateUnit:             "m3/h", // 你库里的 FlowRate 常见是 m3/h
		DensityUnit:              "",     // 若库里是 kg/m3 可留空；若是 g/cm3/t/m3 请填
		VacuumOutUnit:            "kPa",
	},
	"华安龙": {
		PatmPa: 101325,
		G:      9.80665,
		// 其余同上，按该船实测/台账填写
	},
}

func getCfg(ship string) ShipHydraulicsConfig {
	if c, ok := shipCfg[ship]; ok {
		return c
	}
	// 默认配置（尽量保守）
	return ShipHydraulicsConfig{
		PatmPa:        101325,
		G:             9.80665,
		FlowRateUnit:  "m3/h",
		VacuumOutUnit: "kPa",
	}
}

func densityToKgM3(v float64, unit string) float64 {
	switch unit {
	case "kg/m3":
		return v
	case "t/m3", "g/cm3":
		return v * 1000
	default:
		// 简易自检：0~5 认为是相对密度(g/cm3)，乘1000
		if v > 0 && v < 5 {
			return v * 1000
		}
		return v
	}
}

func flowRateToM3s(v float64, unit string) float64 {
	if unit == "m3/h" {
		return v / 3600.0
	}
	return v // 认为已是 m3/s
}

func pipeD(r *model.DredgerDatum, cfg ShipHydraulicsConfig) float64 {
	if cfg.PipeInnerDiameterM > 0 {
		return cfg.PipeInnerDiameterM
	}
	// 数据表里有 MudPipeDiameter（通常 mm），尝试转 m
	if r.MudPipeDiameter > 5 { // 粗判断：>5 可能是 mm
		return r.MudPipeDiameter / 1000.0
	}
	return r.MudPipeDiameter
}

func flowVelocityVs(r *model.DredgerDatum, cfg ShipHydraulicsConfig, D float64) float64 {
	if r.FlowVelocity > 0 {
		return r.FlowVelocity
	}
	if D <= 0 || r.FlowRate <= 0 {
		return math.NaN()
	}
	A := math.Pi * D * D / 4.0
	Q := flowRateToM3s(r.FlowRate, cfg.FlowRateUnit)
	if Q <= 0 || A <= 0 {
		return math.NaN()
	}
	return Q / A
}

func suctionDepthHsPipe(r *model.DredgerDatum) float64 {
	// 约定：CutterDepth 为相对水面的深度（向下为正）
	// 若你的定义不同，请在这里按 BridgeWaterDepth 做一次换算
	if r.CutterDepth > 0 {
		return r.CutterDepth
	}
	return math.NaN()
}

func pumpDepthHsPump(r *model.DredgerDatum, cfg ShipHydraulicsConfig) float64 {
	ear := r.EarDraft
	if ear == 0 {
		// 取左右平均
		if r.LeftEarDraft > 0 && r.RightEarDraft > 0 {
			ear = (r.LeftEarDraft + r.RightEarDraft) / 2.0
		}
	}
	if ear == 0 || r.EarToBottomDistance == 0 || cfg.PumpAboveBottomM <= 0 {
		return math.NaN()
	}
	// 水面→船底深度 = 耳轴吃水 + 耳轴到底
	// 泵深度 = (水面→船底深度) - (泵高于船底)
	bottomDepth := ear + r.EarToBottomDistance
	return bottomDepth - cfg.PumpAboveBottomM
}

func frictionFactorMud(cfg ShipHydraulicsConfig, rhoW, rhoM float64) float64 {
	f := cfg.FrictionFactorClearWater
	if cfg.UseDensityRatio && rhoW > 0 {
		f = f * (rhoM / rhoW)
	}
	return f
}

// 返回：按 Word 公式得到的“真空度”，默认 kPa
func calcVacuumKPa(r *model.DredgerDatum, cfg ShipHydraulicsConfig) float64 {
	g := cfg.G
	if g == 0 {
		g = 9.80665
	}
	Patm := cfg.PatmPa
	if Patm == 0 {
		Patm = 101325
	}
	rhoW := r.WaterDensity
	rhoW = densityToKgM3(rhoW, cfg.DensityUnit)
	if rhoW <= 0 {
		rhoW = 1000
	}

	rhoM := r.Density
	if rhoM <= 0 {
		rhoM = r.FieldSlurryDensity
	}
	rhoM = densityToKgM3(rhoM, cfg.DensityUnit)
	if rhoM <= 0 {
		return math.NaN()
	}

	D := pipeD(r, cfg)
	L := cfg.SuctionPipeLengthM + cfg.LocalEqLengthM
	hsPipe := suctionDepthHsPipe(r)
	hsPump := pumpDepthHsPump(r, cfg)
	Vs := flowVelocityVs(r, cfg, D)

	if D <= 0 || L <= 0 || math.IsNaN(hsPipe) || math.IsNaN(hsPump) || math.IsNaN(Vs) {
		return math.NaN()
	}
	fMud := frictionFactorMud(cfg, rhoW, rhoM)

	staticPipe := rhoW * g * hsPipe
	staticLevel := rhoM * g * (hsPipe - hsPump)
	friction := rhoM * fMud * (L / D) * (Vs * Vs / 2.0)
	kinetic := rhoM * (Vs * Vs / 2.0)

	PvacPa := Patm + staticPipe - staticLevel - friction - kinetic

	// 输出单位：kPa
	return PvacPa / 1000.0
}
