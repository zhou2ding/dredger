package service

import (
	"context"
	"dredger/pkg/logger"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm/clause"

	"dredger/model"

	"gorm.io/gorm"
)

const batchSize = 400

type Service struct {
	db *gorm.DB

	demoBase string                         // ./pys
	dataDir  string                         // ./pys/data
	demoDirs map[DemoID]string              // 1..6 => ./pys/demoN
	seen     map[DemoID]map[string]struct{} // 已知文件名
	mu       sync.Mutex
}

func exeBaseDir() string {
	exe, err := os.Executable()
	if err != nil {
		d, _ := os.Getwd()
		return d
	}
	return filepath.Dir(exe)
}

func findProjectRootWith(subdir string) string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, subdir)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func NewService(db *gorm.DB) *Service {
	base := exeBaseDir()
	pysBase := filepath.Join(base, "pys")
	if _, err := os.Stat(pysBase); err != nil {
		if root := findProjectRootWith("pys"); root != "" {
			pysBase = filepath.Join(root, "pys")
		}
	}
	dataDir := filepath.Join(pysBase, "data")

	s := &Service{
		db:       db,
		demoBase: pysBase, // 绝对
		dataDir:  dataDir, // 绝对 ✅
		demoDirs: map[DemoID]string{
			Demo1: filepath.Join(pysBase, "demo1"),
			Demo2: filepath.Join(pysBase, "demo2"),
			Demo3: filepath.Join(pysBase, "demo3"),
			Demo4: filepath.Join(pysBase, "demo4"),
			Demo5: filepath.Join(pysBase, "demo5"),
			Demo6: filepath.Join(pysBase, "demo6"),
		},
		seen: make(map[DemoID]map[string]struct{}),
	}
	s.initDemoSeen()
	return s
}

func (s *Service) initDemoSeen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, dir := range s.demoDirs {
		files, _ := os.ReadDir(dir)
		m := make(map[string]struct{}, len(files))
		for _, f := range files {
			if !f.IsDir() {
				m[f.Name()] = struct{}{}
			}
		}
		s.seen[id] = m
	}
}

func (s *Service) ImportData(file io.Reader, shipName string, cover bool, startDate, endDate int64) (*ImportDataResult, error) {
	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		logger.Logger.Errorf("open excel file error: %v", err)
		return nil, err
	}

	rows, err := xlsx.GetRows(xlsx.GetSheetName(0))
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, errors.New("文件内容为空")
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	dataDates := []model.DataDate{
		{ShipName: shipName, Date: startDate},
		{ShipName: shipName, Date: endDate},
	}
	if err = tx.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&dataDates).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	var importedCount int

	if strings.Contains(shipName, "华安龙") {
		importedCount, err = executeImport[model.DredgerDataHl](tx, rows, shipName, cover)
	} else if strings.Contains(shipName, "敏龙") {
		importedCount, err = executeImport[model.DredgerDatum](tx, rows, shipName, cover)
	} else {
		return nil, fmt.Errorf("船名[%s]有误，请检查Excel文件后重试", shipName)
	}

	if err != nil {
		tx.Rollback()
		return &ImportDataResult{importedCount}, err
	}

	if err = tx.Commit().Error; err != nil {
		return &ImportDataResult{importedCount}, fmt.Errorf("事务提交失败: %v", err)
	}

	return &ImportDataResult{importedCount}, nil
}

func executeImport[T any](tx *gorm.DB, rows [][]string, shipName string, cover bool) (int, error) {
	var imported int
	var fieldNames []string

	// 使用反射获取模型 T 的字段名，跳过前两个字段（通常是 ID 和 ShipName）
	modelType := reflect.TypeOf(*new(T))
	for i := 2; i < modelType.NumField(); i++ {
		fieldNames = append(fieldNames, modelType.Field(i).Name)
	}

	// 如果 cover 为 true，则删除将要被覆盖的旧数据
	if cover {
		var deleteTimestamps []int64
		for _, row := range rows[1:] {
			if len(row) > 1 {
				// 从Excel的第二列解析时间戳
				if t, err := time.ParseInLocation(time.DateTime, row[1], time.Local); err == nil {
					deleteTimestamps = append(deleteTimestamps, t.UnixMilli())
				}
			}
		}
		if len(deleteTimestamps) > 0 {
			// 使用模型T来指定要删除的表
			err := tx.Where("record_time IN (?) AND ship_name = ?", deleteTimestamps, shipName).Delete(new(T)).Error
			if err != nil {
				logger.Logger.Errorf("覆盖数据时，删除旧数据失败: %v", err)
				return 0, err
			}
		}
	}

	// 初始化用于批量插入的切片
	batch := make([]*T, 0, batchSize)

	for rowNum, row := range rows[1:] {
		if len(row) < len(fieldNames) {
			logger.Logger.Warnf("第 %d 行的列数不足（%d/%d），已跳过", rowNum+2, len(row), len(fieldNames))
			continue
		}

		// 创建模型T的新实例
		dataInstance := reflect.New(modelType).Elem()
		dataInstance.FieldByName("ShipName").SetString(shipName)

		validRow := true
		for i, fieldName := range fieldNames {
			cellVal := row[i+1] // 从Excel第二列开始取值
			field := dataInstance.FieldByName(fieldName)

			if !field.CanSet() {
				logger.Logger.Warnf("第 %d 行的字段 %s 无法设置，已跳过", rowNum+2, fieldName)
				validRow = false
				break
			}

			// 时间字段（RecordTime）特殊处理
			if i == 0 {
				if timestamp, err := time.ParseInLocation(time.DateTime, cellVal, time.Local); err != nil {
					logger.Logger.Warnf("第 %d 行的时间字段 %s 转换失败: %v", rowNum+2, fieldName, err)
					validRow = false
				} else {
					field.SetInt(timestamp.UnixMilli())
				}
				continue
			}

			// 根据字段类型进行转换和赋值
			switch field.Kind() {
			case reflect.Float64, reflect.Float32:
				if num, err := strconv.ParseFloat(cellVal, 64); err == nil {
					field.SetFloat(num)
				} else if cellVal != "" {
					logger.Logger.Warnf("第 %d 行的浮点数字段 %s 转换失败: %v", rowNum+2, fieldName, err)
					validRow = false
				}
			case reflect.Int32, reflect.Int64:
				if num, err := strconv.ParseInt(cellVal, 10, 64); err == nil {
					field.SetInt(num)
				} else if cellVal != "" {
					logger.Logger.Warnf("第 %d 行的整数字段 %s 转换失败: %v", rowNum+2, fieldName, err)
					validRow = false
				}
			case reflect.String:
				field.SetString(cellVal)
			default:
				logger.Logger.Warnf("第 %d 行的字段 %s 是不支持的类型 %s，已跳过", rowNum+2, fieldName, field.Kind())
				validRow = false
			}

			if !validRow {
				break
			}
		}

		if validRow {
			// 将转换后的数据指针添加到批处理切片中
			batch = append(batch, dataInstance.Addr().Interface().(*T))
		} else {
			logger.Logger.Warnf("第 %d 行数据格式有误，已跳过", rowNum+2)
		}

		// 当批处理切片达到指定大小时，执行插入并清空切片
		if len(batch) >= batchSize {
			if err := tx.Create(&batch).Error; err != nil {
				return imported, fmt.Errorf("插入第 %d 批次时出错: %v", (imported/batchSize)+1, err)
			}
			imported += len(batch)
			batch = make([]*T, 0, batchSize) // 重置切片
		}
	}

	// 插入最后一批剩余的数据
	if len(batch) > 0 {
		if err := tx.Create(&batch).Error; err != nil {
			return imported, fmt.Errorf("插入最后批次时出错: %v", err)
		}
		imported += len(batch)
	}

	return imported, nil
}

func (s *Service) GetShiftStats(shipName string, startTime, endTime int64) ([]*ShiftStat, error) {
	var stats []*ShiftStat
	var err error

	if strings.Contains(shipName, "华安龙") {
		var records []*model.DredgerDataHl
		columns := []string{
			"ship_name", "record_time", "hourly_output_rate",
			"underwater_pump_power", "mud_pump_1_power", "mud_pump_2_power",
		}
		err = s.db.Select(columns).
			Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Find(&records).Error
		if err != nil {
			logger.Logger.Errorf("[华安龙]查询班组统计数据失败: %v", err)
			return nil, err
		}

		groups := make(map[int][]*model.DredgerDataHl)
		for _, record := range records {
			hour := time.UnixMilli(record.RecordTime).Hour()
			switch {
			case hour >= 0 && hour < 6:
				groups[1] = append(groups[1], record)
			case hour >= 6 && hour < 12:
				groups[2] = append(groups[2], record)
			case hour >= 12 && hour < 18:
				groups[3] = append(groups[3], record)
			default:
				groups[4] = append(groups[4], record)
			}
		}

		for shift := 1; shift <= 4; shift++ {
			shiftRecords, exists := groups[shift]
			if !exists || len(shiftRecords) == 0 {
				continue
			}

			var minTime, maxTime time.Time
			maxTime, minTime = durationMinutesHl(minTime, maxTime, shiftRecords)
			duration := maxTime.Sub(minTime).Minutes()
			if duration <= 0 {
				continue
			}

			var totalOutputRate float64
			for _, r := range shiftRecords {
				totalOutputRate += r.HourlyOutputRate
			}
			avgOutputRate := totalOutputRate / float64(len(shiftRecords))
			totalProduction := avgOutputRate * (duration / 60)

			var totalPower float64
			for _, r := range shiftRecords {
				totalPower += r.UnderwaterPumpPower + r.MudPump1Power + r.MudPump2Power
			}
			avgPower := totalPower / float64(len(shiftRecords))
			totalEnergyConsumption := avgPower * (duration / 60)
			unitEnergyConsumption := 0.0
			if totalProduction > 0 {
				unitEnergyConsumption = totalEnergyConsumption / totalProduction
			}

			stats = append(stats, &ShiftStat{
				ShiftName:       shiftName(shift),
				BeginTime:       minTime,
				EndTime:         maxTime,
				WorkDuration:    duration,
				TotalProduction: round(totalProduction),
				TotalEnergy:     round(unitEnergyConsumption),
			})
		}
	} else if strings.Contains(shipName, "敏龙") {
		var records []*model.DredgerDatum
		columns := []string{
			"ship_name", "record_time", "output_rate", "underwater_pump_suction_vacuum",
			"intermediate_pressure", "booster_pump_discharge_pressure", "flow_rate",
		}
		err = s.db.Select(columns).
			Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Find(&records).Error
		if err != nil {
			logger.Logger.Errorf("[敏龙]查询班组统计数据失败: %v", err)
			return nil, err
		}

		groups := make(map[int][]*model.DredgerDatum)
		for _, record := range records {
			hour := time.UnixMilli(record.RecordTime).Hour()
			switch {
			case hour >= 0 && hour < 6:
				groups[1] = append(groups[1], record)
			case hour >= 6 && hour < 12:
				groups[2] = append(groups[2], record)
			case hour >= 12 && hour < 18:
				groups[3] = append(groups[3], record)
			default:
				groups[4] = append(groups[4], record)
			}
		}

		for shift := 1; shift <= 4; shift++ {
			shiftRecords, exists := groups[shift]
			if !exists || len(shiftRecords) == 0 {
				continue
			}

			var minTime, maxTime time.Time
			maxTime, minTime = durationMinutes(minTime, maxTime, shiftRecords)
			duration := maxTime.Sub(minTime).Minutes()
			if duration <= 0 {
				continue
			}

			var totalOutputRate float64
			for _, r := range shiftRecords {
				totalOutputRate += r.OutputRate
			}
			avgOutputRate := totalOutputRate / float64(len(shiftRecords))
			totalProduction := avgOutputRate * (duration / 60)

			var totalPower float64
			for _, r := range shiftRecords {
				P1 := r.UnderwaterPumpSuctionVacuum
				P2 := r.IntermediatePressure
				P3 := r.BoosterPumpDischargePressure
				Q := r.FlowRate
				pw1 := 0.8 * Q * (P2 - P1)
				pw2 := 0.8 * Q * (P3 - P2)
				pw := pw1 + pw2
				totalPower += pw
			}
			avgPower := totalPower / float64(len(shiftRecords))
			totalEnergyConsumption := avgPower * (duration / 60)
			unitEnergyConsumption := 0.0
			if totalProduction > 0 {
				unitEnergyConsumption = totalEnergyConsumption / totalProduction
			}

			stats = append(stats, &ShiftStat{
				ShiftName:       shiftName(shift),
				BeginTime:       minTime,
				EndTime:         maxTime,
				WorkDuration:    duration,
				TotalProduction: round(totalProduction),
				TotalEnergy:     round(unitEnergyConsumption),
			})
		}
	} else {
		return nil, fmt.Errorf("船名[%s]暂不支持此统计", shipName)
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].BeginTime.Equal(stats[j].BeginTime) {
			return stats[i].ShiftName < stats[j].ShiftName
		}
		return stats[i].BeginTime.Before(stats[j].BeginTime)
	})

	return stats, nil
}

func (s *Service) GetOptimalShift(shipName string, startTime, endTime int64) (*OptimalShift, error) {
	optimalShift := OptimalShift{
		MinEnergyShift: &ShiftWorkParams{},
	}
	var err error

	if strings.Contains(shipName, "华安龙") {
		var records []*model.DredgerDataHl
		err = s.db.Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Find(&records).Error
		if err != nil {
			logger.Logger.Errorf("[华安龙]查询班组数据失败: %v", err)
			return nil, err
		}

		groups := make(map[int][]*model.DredgerDataHl)
		for _, record := range records {
			hour := time.UnixMilli(record.RecordTime).Hour()
			switch {
			case hour >= 0 && hour < 6:
				groups[1] = append(groups[1], record)
			case hour >= 6 && hour < 12:
				groups[2] = append(groups[2], record)
			case hour >= 12 && hour < 18:
				groups[3] = append(groups[3], record)
			default:
				groups[4] = append(groups[4], record)
			}
		}
		optimalShift.MinEnergyShift.Parameters.BoosterPumpDischargePressure.Max = -1
		for shift := 1; shift <= 4; shift++ {
			shiftRecords, exists := groups[shift]
			if !exists || len(shiftRecords) == 0 {
				continue
			}

			var minTime, maxTime time.Time
			maxTime, minTime = durationMinutesHl(minTime, maxTime, shiftRecords)
			duration := maxTime.Sub(minTime).Minutes()
			if duration <= 0 {
				continue
			}

			var totalOutputRate float64
			for _, r := range shiftRecords {
				totalOutputRate += r.HourlyOutputRate
			}
			avgOutputRate := totalOutputRate / float64(len(shiftRecords))
			totalProduction := avgOutputRate * (duration / 60)

			if totalProduction > optimalShift.TotalProduction {
				optimalShift.TotalProduction = round(totalProduction)
				optimalShift.MaxProductionShift = &ShiftWorkParams{
					ShiftName:  shiftName(shift),
					Parameters: calParamsHl(shiftRecords),
				}
			}

			var totalPower float64
			for _, r := range shiftRecords {
				totalPower += r.UnderwaterPumpPower + r.MudPump1Power + r.MudPump2Power
			}
			avgPower := totalPower / float64(len(shiftRecords))
			totalEnergy := avgPower * (duration / 60)

			if optimalShift.MinEnergyShift.Parameters.BoosterPumpDischargePressure.Max == -1 || totalEnergy < optimalShift.TotalEnergy {
				optimalShift.TotalEnergy = round(totalEnergy)
				optimalShift.MinEnergyShift = &ShiftWorkParams{
					ShiftName:  shiftName(shift),
					Parameters: calParamsHl(shiftRecords),
				}
			}
		}

	} else if strings.Contains(shipName, "敏龙") {
		var records []*model.DredgerDatum
		err = s.db.Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Find(&records).Error
		if err != nil {
			logger.Logger.Errorf("[敏龙]查询班组数据失败: %v", err)
			return nil, err
		}

		groups := make(map[int][]*model.DredgerDatum)
		for _, record := range records {
			hour := time.UnixMilli(record.RecordTime).Hour()
			switch {
			case hour >= 0 && hour < 6:
				groups[1] = append(groups[1], record)
			case hour >= 6 && hour < 12:
				groups[2] = append(groups[2], record)
			case hour >= 12 && hour < 18:
				groups[3] = append(groups[3], record)
			default:
				groups[4] = append(groups[4], record)
			}
		}

		optimalShift.MinEnergyShift.Parameters.BoosterPumpDischargePressure.Max = -1
		for shift := 1; shift <= 4; shift++ {
			shiftRecords, exists := groups[shift]
			if !exists || len(shiftRecords) == 0 {
				continue
			}

			var minTime, maxTime time.Time
			maxTime, minTime = durationMinutes(minTime, maxTime, shiftRecords)
			duration := maxTime.Sub(minTime).Minutes()
			if duration <= 0 {
				continue
			}

			var totalOutputRate float64
			for _, r := range shiftRecords {
				totalOutputRate += r.OutputRate
			}
			avgOutputRate := totalOutputRate / float64(len(shiftRecords))
			totalProduction := avgOutputRate * (duration / 60)

			if totalProduction > optimalShift.TotalProduction {
				optimalShift.TotalProduction = round(totalProduction)
				optimalShift.MaxProductionShift = &ShiftWorkParams{
					ShiftName:  shiftName(shift),
					Parameters: calParams(shiftRecords),
				}
			}

			var totalPower float64
			for _, r := range shiftRecords {
				P1 := r.UnderwaterPumpSuctionVacuum
				P2 := r.IntermediatePressure
				P3 := r.BoosterPumpDischargePressure
				Q := r.FlowRate
				pw1 := 0.8 * Q * (P2 - P1)
				pw2 := 0.8 * Q * (P3 - P2)
				totalPower += (pw1 + pw2)
			}
			avgPower := totalPower / float64(len(shiftRecords))
			totalEnergy := avgPower * (duration / 60)

			if optimalShift.MinEnergyShift.Parameters.BoosterPumpDischargePressure.Max == -1 || totalEnergy < optimalShift.TotalEnergy {
				optimalShift.TotalEnergy = round(totalEnergy)
				optimalShift.MinEnergyShift = &ShiftWorkParams{
					ShiftName:  shiftName(shift),
					Parameters: calParams(shiftRecords),
				}
			}
		}
	} else {
		return nil, fmt.Errorf("船名[%s]暂不支持此统计", shipName)
	}

	return &optimalShift, nil
}

func (s *Service) GetShipList() ([]string, error) {
	var ships1, ships2 []string
	var allShips []string
	shipMap := make(map[string]bool)

	err := s.db.Model(&model.DredgerDatum{}).Distinct().Pluck("ship_name", &ships1).Error
	if err != nil {
		logger.Logger.Errorf("查询船名列表(dredger_data)出错: %v", err)
		return nil, err
	}
	for _, ship := range ships1 {
		if !shipMap[ship] {
			shipMap[ship] = true
			allShips = append(allShips, ship)
		}
	}

	err = s.db.Model(&model.DredgerDataHl{}).Distinct().Pluck("ship_name", &ships2).Error
	if err != nil {
		logger.Logger.Errorf("查询船名列表(dredger_data_hl)出错: %v", err)
		return nil, err
	}
	for _, ship := range ships2 {
		if !shipMap[ship] {
			shipMap[ship] = true
			allShips = append(allShips, ship)
		}
	}

	sort.Strings(allShips)
	return allShips, nil
}

func (s *Service) GetColumns(shipName string) []*ColumnInfo {
	var refType reflect.Type

	if strings.Contains(shipName, "华安龙") {
		refType = reflect.TypeOf(model.DredgerDataHl{})
	} else if strings.Contains(shipName, "敏龙") {
		refType = reflect.TypeOf(model.DredgerDatum{})
	} else {
		return nil
	}

	excludes := map[string]bool{
		"ID":         true,
		"ShipName":   true,
		"RecordTime": true,
	}

	var columns []*ColumnInfo
	for i := 0; i < refType.NumField(); i++ {
		field := refType.Field(i)

		if !excludes[field.Name] {
			tag := field.Tag.Get("gorm")
			parts := strings.Split(tag, ";")
			column := strings.TrimPrefix(parts[0], "column:")
			var columnCN string
			for _, part := range parts {
				if strings.HasPrefix(part, "comment:") {
					columnCN = strings.TrimPrefix(part, "comment:")
					break
				}
			}
			columns = append(columns, &ColumnInfo{
				ColumnName:        column,
				ColumnChineseName: columnCN,
				ColumnUnit:        field.Tag.Get("unit"),
			})
		}
	}

	return columns
}

func (s *Service) GetShiftPie(shipName string, startTime, endTime int64) ([]*ShiftPie, error) {
	var pies []*ShiftPie
	var err error

	if strings.Contains(shipName, "华安龙") {
		var records []*model.DredgerDataHl
		err = s.db.Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Find(&records).Error
		if err != nil {
			logger.Logger.Errorf("[华安龙]查询班组饼图数据失败: %v", err)
			return nil, err
		}

		groups := make(map[int][]*model.DredgerDataHl)
		for _, record := range records {
			hour := time.UnixMilli(record.RecordTime).Hour()
			switch {
			case hour >= 0 && hour < 6:
				groups[1] = append(groups[1], record)
			case hour >= 6 && hour < 12:
				groups[2] = append(groups[2], record)
			case hour >= 12 && hour < 18:
				groups[3] = append(groups[3], record)
			default:
				groups[4] = append(groups[4], record)
			}
		}

		for shift := 1; shift <= 4; shift++ {
			shiftRecords, exists := groups[shift]
			if !exists || len(shiftRecords) == 0 {
				continue
			}

			var minTime, maxTime time.Time
			maxTime, minTime = durationMinutesHl(minTime, maxTime, shiftRecords)
			duration := maxTime.Sub(minTime).Minutes()
			if duration <= 0 {
				continue
			}

			var totalOutputRate float64
			for _, r := range shiftRecords {
				totalOutputRate += r.HourlyOutputRate
			}
			avgOutputRate := totalOutputRate / float64(len(shiftRecords))
			totalProduction := avgOutputRate * (duration / 60)

			var totalPower float64
			for _, r := range shiftRecords {
				totalPower += r.UnderwaterPumpPower + r.MudPump1Power + r.MudPump2Power
			}
			avgPower := totalPower / float64(len(shiftRecords))
			totalEnergy := avgPower * (duration / 60)

			pies = append(pies, &ShiftPie{
				ShiftName: shiftName(shift),
				WorkData: &PieData{
					TotalProduction: round(totalProduction),
					TotalEnergy:     round(totalEnergy),
					WorkDuration:    duration,
				},
			})
		}
	} else if strings.Contains(shipName, "敏龙") {
		var records []*model.DredgerDatum
		err = s.db.Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Find(&records).Error
		if err != nil {
			logger.Logger.Errorf("[敏龙]查询班组饼图数据失败: %v", err)
			return nil, err
		}

		groups := make(map[int][]*model.DredgerDatum)
		for _, record := range records {
			hour := time.UnixMilli(record.RecordTime).Hour()
			switch {
			case hour >= 0 && hour < 6:
				groups[1] = append(groups[1], record)
			case hour >= 6 && hour < 12:
				groups[2] = append(groups[2], record)
			case hour >= 12 && hour < 18:
				groups[3] = append(groups[3], record)
			default:
				groups[4] = append(groups[4], record)
			}
		}

		for shift := 1; shift <= 4; shift++ {
			shiftRecords, exists := groups[shift]
			if !exists || len(shiftRecords) == 0 {
				continue
			}

			var minTime, maxTime time.Time
			maxTime, minTime = durationMinutes(minTime, maxTime, shiftRecords)
			duration := maxTime.Sub(minTime).Minutes()
			if duration <= 0 {
				continue
			}

			var totalOutputRate float64
			for _, r := range shiftRecords {
				totalOutputRate += r.OutputRate
			}
			avgOutputRate := totalOutputRate / float64(len(shiftRecords))
			totalProduction := avgOutputRate * (duration / 60)

			var totalPower float64
			for _, r := range shiftRecords {
				P1 := r.UnderwaterPumpSuctionVacuum
				P2 := r.IntermediatePressure
				P3 := r.BoosterPumpDischargePressure
				Q := r.FlowRate

				pw1 := 0.8 * Q * (P2 - P1)
				pw2 := 0.8 * Q * (P3 - P2)
				totalPower += (pw1 + pw2)
			}
			avgPower := totalPower / float64(len(shiftRecords))
			totalEnergy := avgPower * (duration / 60)

			pies = append(pies, &ShiftPie{
				ShiftName: shiftName(shift),
				WorkData: &PieData{
					TotalProduction: round(totalProduction),
					TotalEnergy:     round(totalEnergy),
					WorkDuration:    duration,
				},
			})
		}
	} else {
		return nil, fmt.Errorf("船名[%s]暂不支持此统计", shipName)
	}

	return pies, nil
}

func (s *Service) GetColumnDataList(columnName, shipName string, startTime, endTime int64) ([]*ColumnData, error) {
	var tableName string
	if strings.Contains(shipName, "华安龙") {
		hl := model.DredgerDataHl{}
		tableName = hl.TableName()
	} else if strings.Contains(shipName, "敏龙") {
		ml := model.DredgerDatum{}
		tableName = ml.TableName()
	} else {
		return nil, fmt.Errorf("船名[%s]暂不支持此统计", shipName)
	}

	var records []map[string]interface{}
	err := s.db.Table(tableName).
		Select(columnName, "record_time").
		Where("ship_name = ?", shipName).
		Where("record_time BETWEEN ? AND ?", startTime, endTime).Scan(&records).Error
	if err != nil {
		logger.Logger.Errorf("查询 %s 历史数据失败: %v", columnName, err)
		return nil, err
	}

	var dataList []*ColumnData
	for _, record := range records {
		t := time.UnixMilli(record["record_time"].(int64)).Format(time.DateTime)
		v := record[columnName]
		var roundVal float64
		if val, ok := v.(float64); ok {
			roundVal = round(val)
		}
		data := &ColumnData{Timestamp: t}
		if roundVal != 0 {
			data.Value = roundVal
		} else {
			data.Value = v
		}
		dataList = append(dataList, data)
	}

	return dataList, nil
}

func (s *Service) GetGlobalTimeRange() ([]*GlobalTimeRange, error) {
	var records []*GlobalTimeRange
	err := s.db.Model(&model.DataDate{}).
		Select("ship_name, MIN(date) as start_date, MAX(date) as end_date").
		Group("ship_name").
		Scan(&records).Error
	if err != nil {
		logger.Logger.Errorf("查询全局时间范围失败: %v", err)
		return nil, err
	}

	for _, record := range records {
		record.StartDateStr = time.UnixMilli(record.StartDate).Format(time.DateOnly)
		record.EndDateStr = time.UnixMilli(record.EndDate).Format(time.DateOnly)
	}

	return records, nil
}

func (s *Service) GetNonEmptyTimeRange(shipName string, startDate, endDate int64) ([]int64, error) {
	var records []int64
	err := s.db.Model(&model.DataDate{}).
		Where("ship_name = ? AND date BETWEEN ? AND ?", shipName, startDate, endDate).
		Order("date ASC").
		Pluck("date", &records).Error
	if err != nil {
		logger.Logger.Errorf("查询有数据的时间范围失败: %v", err)
		return nil, err
	}
	return records, nil
}

func (s *Service) SetTheoryOptimalParams(params *model.TheoryOptimalParam) error {
	// 使用 OnConflict 子句处理唯一键冲突（基于 uk_ship_name）
	// 当 ship_name 冲突时，更新所有指定的列
	err := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ship_name"}}, // 冲突判断的列
		DoUpdates: clause.AssignmentColumns([]string{ // 需要更新的列
			"flow", "concentration", "s_pump_rpm", "cutter_depth", "carriage_travel",
			"horizontal_speed", "booster_pump_discharge_pressure", "vacuum_degree",
		}),
	}).Create(params).Error

	if err != nil {
		logger.Logger.Errorf("保存或更新理论最优参数失败: %v", err)
		return err
	}

	return nil
}

func (s *Service) GetTheoryOptimalParams(shipName string) (*TheoryOptimalParamsDTO, error) {
	var params model.TheoryOptimalParam
	err := s.db.Where("ship_name = ?", shipName).First(&params).Error

	// 如果没有找到记录，这不是一个服务器内部错误，所以我们返回 nil 数据和 nil 错误
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 特别处理“未找到”的情况
		}
		// 其他数据库错误则正常返回
		logger.Logger.Errorf("查询理论最优参数失败: %v", err)
		return nil, err
	}

	dto := &TheoryOptimalParamsDTO{
		ID:                           params.ID,
		CreatedAt:                    params.CreatedAt,
		UpdatedAt:                    params.UpdatedAt,
		ShipName:                     params.ShipName,
		Flow:                         params.Flow,
		Concentration:                params.Concentration,
		SPumpRpm:                     params.SPumpRpm,
		CutterDepth:                  params.CutterDepth,
		CarriageTravel:               params.CarriageTravel,
		HorizontalSpeed:              params.HorizontalSpeed,
		BoosterPumpDischargePressure: params.BoosterPumpDischargePressure,
		VacuumDegree:                 params.VacuumDegree,
	}

	return dto, nil
}

func (s *Service) GetAllShiftParameters(shipName string, startTime, endTime int64) ([]*ShiftWorkParams, error) {
	var allShiftParams []*ShiftWorkParams
	var err error

	if strings.Contains(shipName, "华安龙") {
		var records []*model.DredgerDataHl
		err = s.db.Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Find(&records).Error
		if err != nil {
			logger.Logger.Errorf("[华安龙]查询所有班组参数数据失败: %v", err)
			return nil, err
		}

		// 按班组（小时）进行分组
		groups := make(map[int][]*model.DredgerDataHl)
		for _, record := range records {
			hour := time.UnixMilli(record.RecordTime).Hour()
			switch {
			case hour >= 0 && hour < 6:
				groups[1] = append(groups[1], record)
			case hour >= 6 && hour < 12:
				groups[2] = append(groups[2], record)
			case hour >= 12 && hour < 18:
				groups[3] = append(groups[3], record)
			default:
				groups[4] = append(groups[4], record)
			}
		}

		// 为每个班组计算参数
		for shift := 1; shift <= 4; shift++ {
			shiftRecords, exists := groups[shift]
			if !exists || len(shiftRecords) == 0 {
				continue
			}

			params := &ShiftWorkParams{
				ShiftName:  shiftName(shift),
				Parameters: calParamsHl(shiftRecords), // 调用 tool.go 中的现有函数
			}
			allShiftParams = append(allShiftParams, params)
		}

	} else if strings.Contains(shipName, "敏龙") {
		var records []*model.DredgerDatum
		err = s.db.Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Find(&records).Error
		if err != nil {
			logger.Logger.Errorf("[敏龙]查询所有班组参数数据失败: %v", err)
			return nil, err
		}

		groups := make(map[int][]*model.DredgerDatum)
		for _, record := range records {
			hour := time.UnixMilli(record.RecordTime).Hour()
			switch {
			case hour >= 0 && hour < 6:
				groups[1] = append(groups[1], record)
			case hour >= 6 && hour < 12:
				groups[2] = append(groups[2], record)
			case hour >= 12 && hour < 18:
				groups[3] = append(groups[3], record)
			default:
				groups[4] = append(groups[4], record)
			}
		}

		for shift := 1; shift <= 4; shift++ {
			shiftRecords, exists := groups[shift]
			if !exists || len(shiftRecords) == 0 {
				continue
			}

			params := &ShiftWorkParams{
				ShiftName:  shiftName(shift),
				Parameters: calParams(shiftRecords), // 调用 tool.go 中的现有函数
			}
			allShiftParams = append(allShiftParams, params)
		}
	} else {
		return nil, fmt.Errorf("船名[%s]暂不支持此统计", shipName)
	}

	return allShiftParams, nil
}

func (s *Service) ExecuteSolidProgram(params ExecutionParams) (SolidResult, error) {
	executable := "python"
	script := "solid.py"
	args := []string{script}

	// 使用反射来动态构建命令行参数，这部分逻辑不变。
	v := reflect.ValueOf(params)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name
		argName := fmt.Sprintf("--%c%s", fieldName[0]+32, fieldName[1:])

		isZero := false
		switch field.Kind() {
		case reflect.String:
			if field.String() == "" {
				isZero = true
			}
		case reflect.Float64:
			if field.Float() == 0.0 {
				isZero = true
			}
		default:
			return nil, errors.New("invalid field type")
		}

		if !isZero {
			args = append(args, argName)
			if field.Kind() == reflect.String {
				args = append(args, field.String())
			} else {
				args = append(args, strconv.FormatFloat(field.Float(), 'f', -1, 64))
			}
		}
	}

	cmd := exec.Command(executable, args...)
	fmt.Printf("Executing command: %s\n", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing solid.exe: %v\nOutput: %s", err, string(output))
	}

	var result SolidResult
	if err = json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from script: %v\nRaw output: %s", err, string(output))
	}

	return result, nil
}

func (s *Service) saveUploadsToData(form *multipart.Form) (map[string]string, error) {
	// key: 前端字段名 (geo_path/brd_path/design_xyz/mud_xyz)
	// val: 保存后的绝对/相对路径（我们用相对路径：./pys/data/xxx.ext）
	out := make(map[string]string)
	if form == nil || len(form.File) == 0 {
		return out, nil
	}
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return nil, err
	}
	for field, fhs := range form.File {
		if len(fhs) == 0 {
			continue
		}
		fh := fhs[0]
		src, err := fh.Open()
		if err != nil {
			return nil, err
		}

		dstPath := filepath.Join(s.dataDir, filepath.Base(fh.Filename))
		dst, err := os.Create(dstPath)
		if err != nil {
			return nil, err
		}
		if _, err = io.Copy(dst, src); err != nil {
			dst.Close()
			return nil, err
		}
		out[field] = dstPath
		dst.Close()
		src.Close()
	}
	return out, nil
}

func (s *Service) RunDemo(ctx context.Context, id DemoID, params *DemoParams, form *multipart.Form) ([]GeneratedFile, error) {
	// 1) 保存上传的文件到 ./pys/data
	saved, err := s.saveUploadsToData(form) // 将 geo_path/brd_path/... 替换为保存后的路径
	if err != nil {
		return nil, err
	}

	// 将保存后的路径回填到 params；如果前端本来就传了相对路径，也以保存结果为准
	if v, ok := saved["geo_path"]; ok {
		params.GeoPath = v
	}
	if v, ok := saved["brd_path"]; ok {
		params.BrdPath = v
	}
	if v, ok := saved["design_xyz"]; ok {
		params.DesignXYZ = v
	}
	if v, ok := saved["mud_xyz"]; ok {
		params.MudXYZ = v
	}

	// 2) 记录执行前已有文件
	dir, ok := s.demoDirs[id]
	if !ok {
		return nil, fmt.Errorf("unknown demo id: %d", id)
	}
	before, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	beforeSet := make(map[string]struct{}, len(before))
	for _, f := range before {
		if !f.IsDir() {
			beforeSet[f.Name()] = struct{}{}
		}
	}

	// 3) 组装命令
	exeName := fmt.Sprintf("Python_Demo_%d.exe", id)
	exePath := filepath.Join(dir, exeName)

	abs, _ := filepath.Abs(exePath)
	pwd, _ := os.Getwd()
	log.Printf("[DEBUG] cwd=%s exePath=%s abs=%s", pwd, exePath, abs)
	if _, err := os.Stat(abs); err != nil {
		log.Printf("[DEBUG] os.Stat(abs) error: %v", err)
	}

	// 统一把四个文件参数转成绝对路径（优先使用 saveUploadsToData 回填的路径）
	// 兼容：即便前端没上传文件、只给了字符串路径，也会被规范成 data 目录下的绝对路径
	params.GeoPath = makeAbsUnder(params.GeoPath, s.dataDir)
	params.BrdPath = makeAbsUnder(params.BrdPath, s.dataDir)
	params.DesignXYZ = makeAbsUnder(params.DesignXYZ, s.dataDir)
	params.MudXYZ = makeAbsUnder(params.MudXYZ, s.dataDir)

	// 可选：执行前做存在性检查，能更早报错定位
	for _, p := range []struct{ label, path string }{
		{"--geo_path", params.GeoPath},
		{"--brd_path", params.BrdPath},
		{"--design_xyz", params.DesignXYZ},
		{"--mud_xyz", params.MudXYZ},
	} {
		if p.path != "" {
			if _, err := os.Stat(p.path); err != nil {
				return nil, fmt.Errorf("input file %s not found: %s (%v)", p.label, p.path, err)
			}
		}
	}

	args := []string{
		"--geo_path", params.GeoPath,
		"--brd_path", params.BrdPath,
		"--design_xyz", params.DesignXYZ,
		"--mud_xyz", params.MudXYZ,
		"--ref_z", strconv.FormatFloat(params.RefZ, 'f', -1, 64),
		"--grid_xy", strconv.FormatFloat(params.GridXY, 'f', -1, 64),
		"--grid_z", strconv.FormatFloat(params.GridZ, 'f', -1, 64),
	}
	switch id {
	case Demo3, Demo4:
		args = append(args,
			"--cx", fmt.Sprintf("%v", params.CX),
			"--cy", fmt.Sprintf("%v", params.CY),
			"--length", fmt.Sprintf("%v", params.Length),
			"--width", fmt.Sprintf("%v", params.Width),
			"--depth", fmt.Sprintf("%v", params.Depth),
			"--height", fmt.Sprintf("%v", params.Height),
		)
	case Demo5:
		args = append(args,
			"--cx", fmt.Sprintf("%v", params.CX),
			"--cy", fmt.Sprintf("%v", params.CY),
		)
	case Demo6:
		args = append(args,
			"--x1", fmt.Sprintf("%v", params.X1),
			"--y1", fmt.Sprintf("%v", params.Y1),
			"--x2", fmt.Sprintf("%v", params.X2),
			"--y2", fmt.Sprintf("%v", params.Y2),
			"--threshold", fmt.Sprintf("%v", params.Threshold),
		)
	}

	// 4) 执行（阻塞至结束，前端显示“转圈”即可）
	log.Printf("[DEBUG] exePath=%s ats=%v", abs, args)
	cmd := exec.CommandContext(ctx, abs, args...)
	// 重要：设置工作目录为对应 demo 目录，保证相对路径/依赖就近
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run %s failed: %v\nOutput: %s", exeName, err, string(output))
	}

	// 5) 扫描新增文件
	after, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var news []GeneratedFile
	for _, f := range after {
		if f.IsDir() {
			continue
		}
		if _, existed := beforeSet[f.Name()]; existed {
			continue
		}
		info, _ := f.Info()
		news = append(news, GeneratedFile{
			Name: f.Name(),
			Path: filepath.Join(dir, f.Name()),
			Size: info.Size(),
			Mod:  info.ModTime().UnixMilli(),
			Ext:  strings.ToLower(filepath.Ext(f.Name())),
		})
	}

	// 6) 更新“已知文件”集合（内存）
	s.mu.Lock()
	for _, nf := range news {
		if s.seen[id] == nil {
			s.seen[id] = make(map[string]struct{})
		}
		s.seen[id][nf.Name] = struct{}{}
	}
	s.mu.Unlock()

	if len(news) > 0 {
		logFilePath := filepath.Join(dir, "execution_log.json")

		// 读取现有日志
		var logs []ExecutionLogEntry
		logFile, err := os.ReadFile(logFilePath)
		if err == nil {
			_ = json.Unmarshal(logFile, &logs)
		}

		// 添加新条目
		newEntry := ExecutionLogEntry{
			Timestamp: time.Now().UnixMilli(),
			Files:     news,
		}
		logs = append(logs, newEntry)

		// 写回文件
		updatedLogs, err := json.MarshalIndent(logs, "", "  ")
		if err == nil {
			_ = os.WriteFile(logFilePath, updatedLogs, 0644)
		} else {
			logger.Logger.Warnf("写入执行日志 %s 失败: %v", logFilePath, err)
		}
	}

	return news, nil
}

func (s *Service) GetLatestResult(id DemoID) (*ExecutionLogEntry, error) {
	dir, ok := s.demoDirs[id]
	if !ok {
		return nil, fmt.Errorf("unknown demo id: %d", id)
	}

	logFilePath := filepath.Join(dir, "execution_log.json")

	var logs []ExecutionLogEntry
	logFile, err := os.ReadFile(logFilePath)
	// 如果日志文件不存在，说明从未成功执行过，这不是一个错误
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(logFile, &logs); err != nil {
		return nil, fmt.Errorf("解析日志文件 %s 失败: %w", logFilePath, err)
	}

	if len(logs) == 0 {
		return nil, nil // 日志为空
	}

	// 按时间戳降序排序，找到最新的
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp > logs[j].Timestamp
	})

	return &logs[0], nil
}

func (s *Service) GetLatestResults(ids []DemoID) (map[DemoID]*ExecutionLogEntry, error) {
	results := make(map[DemoID]*ExecutionLogEntry)
	for _, id := range ids {
		entry, err := s.GetLatestResult(id)
		if err != nil {
			// 如果只是某个demo的日志不存在，可以忽略
			logger.Logger.Warnf("查询 demo %d 最新结果失败: %v", id, err)
			continue
		}
		if entry != nil {
			results[id] = entry
		}
	}
	return results, nil
}

func (s *Service) OpenLocation(filePath string) error {
	// 安全校验：再次确认路径在允许的pys目录下
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("无效的路径: %s", filePath)
	}

	root, _ := filepath.Abs(s.demoBase) // s.demoBase 是 "./pys"
	if !strings.HasPrefix(absPath, root) {
		return fmt.Errorf("路径不允许访问: %s", filePath)
	}

	// 获取文件所在的目录
	dir := filepath.Dir(absPath)

	// 在Windows上，使用 explorer 命令打开目录
	cmd := exec.Command("explorer", dir)
	if err := cmd.Start(); err != nil { // 使用 Start() 而不是 Run()，避免阻塞后端服务
		logger.Logger.Errorf("打开目录 %s 失败: %v", dir, err)
		return fmt.Errorf("无法打开目录: %v", err)
	}

	return nil
}
