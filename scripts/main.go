package main

import (
	"dredger/model"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

const batchSize = 400

func main() {
	host := flag.String("h", "", "mysql地址")
	port := flag.String("p", "", "mysql端口")
	user := flag.String("u", "", "mysql账号")
	password := flag.String("a", "", "mysql密码")
	fileDir := flag.String("d", "", "excel文件所在的目录")
	flag.Parse()

	if *host == "" || *port == "" || *password == "" {
		flag.Usage()
		return
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/dredger?charset=utf8mb4&parseTime=True&loc=Local", *user, *password, *host, *port)

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Silent,
				Colorful:      false,
			},
		),
	})
	if err != nil {
		fmt.Printf("连接mysql失败: %v\n", err)
		return
	}

	files, err := os.ReadDir(*fileDir)
	if err != nil {
		fmt.Printf("读取目录失败: %v\n", err)
		return
	}

	totalImported := 0

	for _, file := range files {
		now := time.Now()
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".xlsx") {
			continue
		}

		filePath := filepath.Join(*fileDir, file.Name())
		baseName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		shipName, startDate, endData, err := parseFileName(baseName)
		if err != nil {
			fmt.Printf("文件 %s 无法解析船名，跳过\n", filePath)
			continue
		}

		f, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("打开文件 %s 失败: %v\n", filePath, err)
			continue
		}

		var imported int
		if strings.Contains(shipName, "华安龙") {
			imported, err = importDataHualong(f, shipName, startDate, endData)
		} else {
			imported, err = importData(f, shipName, startDate, endData)

		}
		f.Close()
		if err != nil {
			fmt.Printf("导入文件 %s 失败: %v\n", filePath, err)
		} else {
			fmt.Printf("成功导入文件 %s，%d 条记录，耗时 %.2fs\n", filePath, imported, time.Since(now).Seconds())
			totalImported += imported
		}
	}

	fmt.Printf("\n总计导入 %d 条记录\n", totalImported)
}

func importData(file io.Reader, shipName string, startDate, endDate int64) (int, error) {
	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		fmt.Printf("open excel file error: %v\n", err)
		return 0, err
	}

	rows, err := xlsx.GetRows(xlsx.GetSheetName(0))
	if err != nil {
		return 0, err
	}

	if len(rows) < 2 {
		return 0, errors.New("文件内容为空")
	}

	tx := db.Begin()
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
		return 0, err
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

	for rowNum, row := range rows[1:] {
		if len(row) < len(fieldNames) {
			fmt.Printf("第 %d 行列数不足（%d/%d），跳过", rowNum+2, len(row), len(fieldNames))
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
					fmt.Printf("第 %d 行字段 %s 转换失败: %v\n", rowNum+2, fieldName, err)
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
					fmt.Printf("第 %d 行字段 %s 转换失败: %v\n", rowNum+2, fieldName, err)
					valid = false
				}
			case reflect.Int32, reflect.Int64:
				if num, err := strconv.ParseInt(cellVal, 10, 64); err == nil {
					field.SetInt(num)
				} else {
					fmt.Printf("第 %d 行字段 %s 转换失败: %v\n", rowNum+2, fieldName, err)
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
			fmt.Printf("第 %d 行数据格式错误，已跳过", rowNum+2)
		}

		if len(batch) >= batchSize {
			if err = tx.Create(&batch).Error; err != nil {
				tx.Rollback()
				return imported, fmt.Errorf("插入第 %d 批次时出错: %v\n", imported/batchSize+1, err)
			}
			imported += len(batch)
			batch = nil
		}
	}

	if len(batch) > 0 {
		if err = tx.Create(&batch).Error; err != nil {
			tx.Rollback()
			return imported, fmt.Errorf("插入最后批次时出错: %v\n", err)
		}
		imported += len(batch)
	}

	if err = tx.Commit().Error; err != nil {
		return imported, fmt.Errorf("事务提交失败: %v\n", err)
	}

	return imported, nil
}

func importDataHualong(file io.Reader, shipName string, startDate, endDate int64) (int, error) {
	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		fmt.Printf("open excel file error: %v\n", err)
		return 0, err
	}

	rows, err := xlsx.GetRows(xlsx.GetSheetName(0))
	if err != nil {
		return 0, err
	}

	if len(rows) < 2 {
		return 0, errors.New("文件内容为空")
	}

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 插入时间标记
	dataDates := []model.DataDate{
		{ShipName: shipName, Date: startDate},
		{ShipName: shipName, Date: endDate},
	}
	if err = tx.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&dataDates).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	var (
		imported   int
		fieldNames []string
		batch      []model.DredgerDataHl
	)

	refType := reflect.TypeOf(model.DredgerDataHl{})
	for i := 2; i < refType.NumField(); i++ {
		fieldNames = append(fieldNames, refType.Field(i).Name)
	}

	for rowNum, row := range rows[1:] {
		if len(row) < len(fieldNames) {
			fmt.Printf("第 %d 行列数不足（%d/%d），跳过\n", rowNum+2, len(row), len(fieldNames))
			continue
		}
		data := model.DredgerDataHl{ShipName: shipName}
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
				t, err := time.ParseInLocation(time.DateTime, cellVal, time.Local)
				if err != nil {
					fmt.Printf("第 %d 行字段 %s 转换失败: %v\n", rowNum+2, fieldName, err)
					valid = false
				} else {
					field.SetInt(t.UnixMilli())
				}
				continue
			}

			switch field.Kind() {
			case reflect.Float64:
				if num, err := strconv.ParseFloat(cellVal, 64); err == nil {
					field.SetFloat(num)
				}
			case reflect.Int64:
				if num, err := strconv.ParseInt(cellVal, 10, 64); err == nil {
					field.SetInt(num)
				}
			case reflect.String:
				field.SetString(cellVal)
			}
		}

		if valid {
			batch = append(batch, data)
		}

		if len(batch) >= batchSize {
			if err := tx.Create(&batch).Error; err != nil {
				tx.Rollback()
				return imported, fmt.Errorf("插入第 %d 批次时出错: %v\n", imported/batchSize+1, err)
			}
			imported += len(batch)
			batch = nil
		}
	}

	if len(batch) > 0 {
		if err := tx.Create(&batch).Error; err != nil {
			tx.Rollback()
			return imported, fmt.Errorf("插入最后批次时出错: %v\n", err)
		}
		imported += len(batch)
	}

	if err := tx.Commit().Error; err != nil {
		return imported, fmt.Errorf("事务提交失败: %v\n", err)
	}

	return imported, nil
}

func parseFileName(fileName string) (shipName string, start, end int64, err error) {
	re := regexp.MustCompile(`^([\p{Han}]+)(\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2})至(\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2})`)
	matches := re.FindStringSubmatch(fileName)
	if len(matches) != 4 {
		return "", 0, 0, errors.New("文件名不合法")
	}

	startTime, err := parseTime(matches[2])
	if err != nil {
		return "", 0, 0, err
	}

	endTime, err := parseTime(matches[3])
	if err != nil {
		return "", 0, 0, err
	}

	return matches[1], startTime, endTime, nil
}

func parseTime(tsStr string) (int64, error) {
	t, err := time.ParseInLocation("2006-01-02-15-04-05", tsStr, time.Local)
	if err != nil {
		return 0, err
	}
	truncated := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)

	return truncated.UnixMilli(), nil
}
