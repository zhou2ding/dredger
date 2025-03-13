package service

import (
	"dredger/pkg/logger"
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"io"
	"log"
	"math"
	"reflect"
	"sort"
	"strconv"
	"time"

	"dredger/model"
	"gorm.io/gorm"
)

const batchSize = 400

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) ImportData(file io.Reader, shipName string) (*ImportDataResult, error) {
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

	var (
		imported   int
		fieldNames []string
		batch      []model.DredgerDatum
	)

	refType := reflect.TypeOf(model.DredgerDatum{})
	for i := 2; i < refType.NumField(); i++ {
		fieldNames = append(fieldNames, refType.Field(i).Name)
	}
	for rowNum, row := range rows[1:] {
		if len(row) < len(fieldNames) {
			logger.Logger.Warnf("警告：第 %d 行列数不足（%d/%d），跳过", rowNum+2, len(row), len(fieldNames))
			continue
		}
		data := model.DredgerDatum{ShipName: shipName}
		v := reflect.ValueOf(&data).Elem()

		valid := true
		for i, fieldName := range fieldNames {
			cellVal := row[i+1]
			field := v.FieldByName(fieldName)
			if !field.CanSet() {
				valid = false
				break
			}
			switch field.Kind() {
			case reflect.Float64, reflect.Float32:
				if num, err := strconv.ParseFloat(cellVal, 64); err == nil {
					field.SetFloat(num)
				} else {
					log.Printf("第 %d 行字段 %s 转换失败: %v", rowNum+2, fieldName, err)
					valid = false
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
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

func (s *Service) GetShiftStats(shipName string, startTime, endTime time.Time) ([]ShiftStat, error) {
	// 格式化查询时间范围
	startDate := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	stopDate := time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())
	startStr := startDate.Format(time.DateTime)
	endStr := stopDate.Format(time.DateTime)

	// 查询符合条件的记录
	var records []model.DredgerDatum
	err := s.db.Where("ship_name = ?", shipName).
		Where("record_time BETWEEN ? AND ?", startStr, endStr).
		Find(&records).Error
	if err != nil {
		logger.Logger.Errorf("查询班组统计数据失败: %v", err)
		return nil, err
	}

	// 分组统计
	groups := make(map[int][]model.DredgerDatum)
	for _, record := range records {
		t, err := time.Parse(time.DateTime, record.RecordTime)
		if err != nil {
			continue // 跳过解析失败的记录
		}
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
	var stats []ShiftStat
	for shift := 1; shift <= 4; shift++ {
		shiftRecords, exists := groups[shift]
		if !exists || len(shiftRecords) == 0 {
			continue
		}

		// 计算班次时间范围
		var minTime, maxTime time.Time
		duration := durationMinutes(minTime, maxTime, shiftRecords)

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

		stats = append(stats, ShiftStat{
			ShiftName:       shiftName(shift),
			BeginTime:       minTime,
			EndTime:         maxTime,
			WorkDuration:    duration,
			TotalProduction: math.Round(totalProduction*100) / 100,
			TotalEnergy:     math.Round(totalEnergy*100) / 100,
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

func (s *Service) AnalyzeOptimalShift(shipName string, startTime, endTime time.Time, metric string) (OptimalAnalysis, error) {
	return OptimalAnalysis{}, nil
}
