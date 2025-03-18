package service

import (
	"dredger/dao"
	"dredger/pkg/logger"
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm/clause"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"dredger/model"
	"gorm.io/gorm"
)

const batchSize = 400

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	dao.SetDefault(db)
	return &Service{
		db: db,
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

	var (
		imported   int
		fieldNames []string
		batch      []model.DredgerDatum
	)

	refType := reflect.TypeOf(model.DredgerDatum{})
	for i := 2; i < refType.NumField(); i++ {
		fieldNames = append(fieldNames, refType.Field(i).Name)
	}

	var deletes []int64
	if cover {
		for _, row := range rows[1:] {
			del, _ := time.ParseInLocation(time.DateTime, row[1], time.Local)
			deletes = append(deletes, del.UnixMilli())
		}
		err = tx.Where("record_time IN (?)", deletes).Delete(model.DredgerDatum{}).Error
		if err != nil {
			logger.Logger.Errorf("覆盖数据时，删除旧数据失败: %v", err)
			return nil, err
		}
	}

	for rowNum, row := range rows[1:] {
		if len(row) < len(fieldNames) {
			logger.Logger.Warnf("第 %d 行列数不足（%d/%d），跳过", rowNum+2, len(row), len(fieldNames))
			continue
		}
		data := model.DredgerDatum{ShipName: shipName}
		elem := reflect.ValueOf(&data).Elem()

		valid := true
		for i, fieldName := range fieldNames {
			cellVal := row[i+1]
			field := elem.FieldByName(fieldName)
			if !field.CanSet() {
				valid = false
				break
			}
			if i == 0 {
				// 第一个字段是时间戳
				timestamp, err := time.ParseInLocation(time.DateTime, cellVal, time.Local)
				if err != nil {
					logger.Logger.Warnf("第 %d 行字段 %s 转换失败: %v", rowNum+2, fieldName, err)
					valid = false
				} else {
					field.SetInt(timestamp.UnixMilli())
				}
				continue
			}
			switch field.Kind() {
			case reflect.Float64, reflect.Float32:
				if num, err := strconv.ParseFloat(cellVal, 64); err == nil {
					field.SetFloat(num)
				} else {
					logger.Logger.Warnf("第 %d 行字段 %s 转换失败: %v", rowNum+2, fieldName, err)
					valid = false
				}
			case reflect.Int32, reflect.Int64:
				if num, err := strconv.ParseInt(cellVal, 10, 64); err == nil {
					field.SetInt(num)
				} else {
					logger.Logger.Warnf("第 %d 行字段 %s 转换失败: %v", rowNum+2, fieldName, err)
					valid = false
				}
			case reflect.String:
				field.SetString(cellVal)
			default:
				valid = false
			}
		}

		if valid {
			batch = append(batch, data)
		} else {
			logger.Logger.Warnf("第 %d 行数据格式错误，已跳过", rowNum+2)
		}

		if len(batch) >= batchSize {
			if err = tx.Create(&batch).Error; err != nil {
				tx.Rollback()
				return &ImportDataResult{imported}, fmt.Errorf("插入第 %d 批次时出错: %v", imported/batchSize+1, err)
			}
			imported += len(batch)
			batch = nil
		}
	}

	if len(batch) > 0 {
		if err = tx.Create(&batch).Error; err != nil {
			tx.Rollback()
			return &ImportDataResult{imported}, fmt.Errorf("插入最后批次时出错: %v", err)
		}
		imported += len(batch)
	}

	if err = tx.Commit().Error; err != nil {
		return &ImportDataResult{imported}, fmt.Errorf("事务提交失败: %v", err)
	}

	return &ImportDataResult{imported}, nil
}

func (s *Service) GetShiftStats(shipName string, startTime, endTime int64) ([]*ShiftStat, error) {
	// 查询符合条件的记录
	var records []*model.DredgerDatum
	err := s.db.Where("ship_name = ?", shipName).
		Where("record_time BETWEEN ? AND ?", startTime, endTime).
		Find(&records).Error
	if err != nil {
		logger.Logger.Errorf("查询班组统计数据失败: %v", err)
		return nil, err
	}

	// 分组统计
	groups := make(map[int][]*model.DredgerDatum)
	for _, record := range records {
		t := time.UnixMilli(record.RecordTime)
		hour := t.Hour()
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

	// 生成统计结果
	var stats []*ShiftStat
	for shift := 1; shift <= 4; shift++ {
		shiftRecords, exists := groups[shift]
		if !exists || len(shiftRecords) == 0 {
			continue
		}

		// 计算班次时间范围
		var minTime, maxTime time.Time
		maxTime, minTime = durationMinutes(minTime, maxTime, shiftRecords)
		duration := maxTime.Sub(minTime).Minutes()

		// 计算产量总量
		var totalOutputRate float64
		for _, r := range shiftRecords {
			totalOutputRate += r.OutputRate
		}
		avgOutputRate := totalOutputRate / float64(len(shiftRecords))
		totalProduction := avgOutputRate * (duration / 60)

		// 计算能耗
		var totalPower float64
		for _, r := range shiftRecords {
			P1 := r.UnderwaterPumpSuctionVacuum
			P2 := r.IntermediatePressure
			P3 := r.BoosterPumpDischargePressure
			Q := r.FlowRate

			pw1 := 0.8 * Q * (P2 - P1)
			pw2 := 0.8 * Q * (P3 - P2)

			totalPower += pw1 + pw2
		}
		avgPower := totalPower / float64(len(shiftRecords))
		totalEnergy := avgPower * (duration / 60)

		if totalProduction != 0 {
			totalEnergy = totalEnergy / totalProduction
		}
		stats = append(stats, &ShiftStat{
			ShiftName:       shiftName(shift),
			BeginTime:       minTime,
			EndTime:         maxTime,
			WorkDuration:    duration,
			TotalProduction: round(totalProduction),
			TotalEnergy:     round(totalEnergy),
		})
	}

	// 按日期和班次排序
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].BeginTime.Equal(stats[j].BeginTime) {
			return stats[i].ShiftName < stats[j].ShiftName
		}
		return stats[i].BeginTime.Before(stats[j].BeginTime)
	})

	return stats, nil
}

func (s *Service) GetOptimalShift(shipName string, startTime, endTime int64) (*OptimalShift, error) {
	// 查询符合条件的记录
	var records []*model.DredgerDatum
	err := s.db.Where("ship_name = ?", shipName).
		Where("record_time BETWEEN ? AND ?", startTime, endTime).
		Find(&records).Error
	if err != nil {
		logger.Logger.Errorf("查询班组数据失败: %v", err)
		return nil, err
	}

	// 分组统计
	groups := make(map[int][]*model.DredgerDatum)
	for _, record := range records {
		t := time.UnixMilli(record.RecordTime)
		hour := t.Hour()
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

	optimalShift := OptimalShift{}
	for shift := 1; shift <= 4; shift++ {
		shiftRecords, exists := groups[shift]
		if !exists || len(shiftRecords) == 0 {
			continue
		}

		// 计算班次时间范围
		var minTime, maxTime time.Time
		maxTime, minTime = durationMinutes(minTime, maxTime, shiftRecords)
		duration := maxTime.Sub(minTime).Minutes()

		// 计算产量总量
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
		// 计算能耗
		var totalEnergy float64
		for _, r := range shiftRecords {
			P1 := r.UnderwaterPumpSuctionVacuum
			P2 := r.IntermediatePressure
			P3 := r.BoosterPumpDischargePressure
			Q := r.FlowRate

			pw1 := 0.8 * Q * (P2 - P1)
			pw2 := 0.8 * Q * (P3 - P2)
			totalEnergy += (pw1 + pw2) * (duration / 60)
		}

		if optimalShift.TotalEnergy == 0 {
			optimalShift.TotalEnergy = round(totalEnergy)
			optimalShift.MinEnergyShift = &ShiftWorkParams{
				ShiftName:  shiftName(shift),
				Parameters: calParams(shiftRecords),
			}
		}
		if totalEnergy < optimalShift.TotalEnergy {
			optimalShift.TotalEnergy = round(totalEnergy)
			optimalShift.MinEnergyShift = &ShiftWorkParams{
				ShiftName:  shiftName(shift),
				Parameters: calParams(shiftRecords),
			}
		}
	}

	return &optimalShift, nil
}

func (s *Service) GetShipList() ([]string, error) {
	var records []model.DredgerDatum
	err := s.db.Distinct("ship_name").Find(&records).Error
	if err != nil {
		logger.Logger.Errorf("查询船名列表出错: %v", err)
		return nil, err
	}

	ships := make([]string, 0, len(records))
	for _, record := range records {
		ships = append(ships, record.ShipName)
	}
	return ships, nil
}

func (s *Service) GetColumns() []*ColumnInfo {
	refTypes := reflect.TypeOf(model.DredgerDatum{})
	excludes := map[string]bool{
		"ID":         true,
		"ShipName":   true,
		"RecordTime": true,
	}

	var columns []*ColumnInfo
	for i := 0; i < refTypes.NumField(); i++ {
		field := refTypes.Field(i)

		tag := field.Tag.Get("gorm")
		parts := strings.Split(tag, ";")
		column := strings.TrimPrefix(parts[0], "column:")
		var columnCN string
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "comment:") {
				columnCN = strings.TrimPrefix(part, "comment:")
				break
			}
		}
		if !excludes[field.Name] {
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
	// 查询符合条件的记录
	var records []*model.DredgerDatum
	err := s.db.Where("ship_name = ?", shipName).
		Where("record_time BETWEEN ? AND ?", startTime, endTime).
		Find(&records).Error
	if err != nil {
		logger.Logger.Errorf("查询班组统计数据失败: %v", err)
		return nil, err
	}

	// 分组统计
	groups := make(map[int][]*model.DredgerDatum)
	for _, record := range records {
		t := time.UnixMilli(record.RecordTime)
		hour := t.Hour()
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

	// 生成统计结果
	var pies []*ShiftPie
	for shift := 1; shift <= 4; shift++ {
		shiftRecords, exists := groups[shift]
		if !exists || len(shiftRecords) == 0 {
			continue
		}

		// 计算班次时间范围
		var minTime, maxTime time.Time
		maxTime, minTime = durationMinutes(minTime, maxTime, shiftRecords)
		duration := maxTime.Sub(minTime).Minutes()

		// 计算产量总量
		var totalOutputRate float64
		for _, r := range shiftRecords {
			totalOutputRate += r.OutputRate
		}
		avgOutputRate := totalOutputRate / float64(len(shiftRecords))
		totalProduction := avgOutputRate * (duration / 60)

		// 计算能耗
		var totalEnergy float64
		for _, r := range shiftRecords {
			P1 := r.UnderwaterPumpSuctionVacuum
			P2 := r.IntermediatePressure
			P3 := r.BoosterPumpDischargePressure
			Q := r.FlowRate

			pw1 := 0.8 * Q * (P2 - P1)
			pw2 := 0.8 * Q * (P3 - P2)
			totalEnergy += (pw1 + pw2) * (duration / 60)
		}

		pies = append(pies, &ShiftPie{
			ShiftName: shiftName(shift),
			WorkData: &PieData{
				TotalProduction: totalProduction,
				TotalEnergy:     totalEnergy,
				WorkDuration:    duration,
			},
		})
	}

	return pies, nil
}

func (s *Service) GetColumnDataList(columnName, shipName string, startTime, endTime int64) ([]*ColumnData, error) {
	var records []map[string]any
	err := s.db.Table(dao.DredgerDatum.TableName()).
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
		if _, ok := v.(float64); ok {
			roundVal = round(v.(float64))
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
