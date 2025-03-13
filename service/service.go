package service

import (
	"dredger/pkg/logger"
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"io"
	"log"
	"reflect"
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
	var stats []ShiftStat
	shifts := []string{"0-6", "6-12", "12-18", "18-24"}

	for _, shift := range shifts {
		startHour, endHour := parseShiftHours(shift)
		var totalDuration float64
		var totalProduction float64
		var totalEnergy float64

		// 计算施工时长
		query := s.db.Where(s.db.Where("ship_name = ?", shipName).
			Where("record_time BETWEEN ? AND ?", startTime, endTime).
			Where("HOUR(record_time) >= ?", startHour).
			Where("HOUR(record_time) < ?", endHour))

		// 计算产量
		var avgProductionRate float64
		query.Select("AVG(hourly_output_rate)").Scan(&avgProductionRate)
		totalProduction = avgProductionRate * totalDuration

		stats = append(stats, ShiftStat{
			Shift:             shift,
			Duration:          totalDuration,
			TotalProduction:   totalProduction,
			EnergyConsumption: totalEnergy,
		})
	}

	return stats, nil
}

func (s *Service) AnalyzeOptimalShift(shipName string, startTime, endTime time.Time, metric string) (OptimalAnalysis, error) {

	var optimal ShiftStat
	switch metric {
	case "max_production":
	case "min_energy":
	default:
		return OptimalAnalysis{}, errors.New("invalid metric")
	}

	// 获取详细参数
	params := s.getShiftParameters(shipName, startTime, endTime, optimal.Shift)
	return OptimalAnalysis{
		OptimalShift: optimal.Shift,
		Parameters:   params,
	}, nil
}

func (s *Service) getShiftParameters(shipName string, startTime, endTime time.Time, shift string) ParameterStats {
	startHour, endHour := parseShiftHours(shift)
	query := s.db.Where("ship_name = ?", shipName).
		Where("record_time BETWEEN ? AND ?", startTime, endTime).
		Where("HOUR(record_time) >= ?", startHour).
		Where("HOUR(record_time) < ?", endHour)

	var data []model.DredgerDatum
	query.Find(&data)

	stats := ParameterStats{}

	// 数据有效性检查
	stats.SwingSpeed.Warnings = checkSwingSpeedValidity(data)
	return stats
}

func parseShiftHours(shift string) (int, int) {
	switch shift {
	case "0-6":
		return 0, 6
	case "6-12":
		return 6, 12
	case "12-18":
		return 12, 18
	case "18-24":
		return 18, 24
	default:
		return 0, 0
	}
}

func checkSwingSpeedValidity(data []model.DredgerDatum) []string {
	var warnings []string
	for _, d := range data {
		if d.HourlyOutputRate > 0 && d.TransverseSpeed == 0 {
			warnings = append(warnings, fmt.Sprintf("时间戳 %s: 横移速度为0但产量率>0", d.RecordTime))
		}
	}
	return warnings
}
