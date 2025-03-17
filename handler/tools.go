package handler

import (
	"errors"
	"regexp"
	"time"
)

func parseFileName(fileName string) (start, end int64, err error) {
	re := regexp.MustCompile(`^([\p{Han}]+)(\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2})至(\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2})`)
	matches := re.FindStringSubmatch(fileName)
	if len(matches) != 4 {
		return 0, 0, errors.New("文件名不合法")
	}

	startTime, err := parseTime(matches[2])
	if err != nil {
		return 0, 0, err
	}

	endTime, err := parseTime(matches[3])
	if err != nil {
		return 0, 0, err
	}

	return startTime, endTime, nil
}

func parseTime(tsStr string) (int64, error) {
	t, err := time.ParseInLocation("2006-01-02-15-04-05", tsStr, time.Local)
	if err != nil {
		return 0, err
	}
	truncated := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)

	return truncated.UnixMilli(), nil
}
