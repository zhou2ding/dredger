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

func durationMinutes(minTime, maxTime time.Time, records []model.DredgerDatum) float64 {
	for i, r := range records {
		t, _ := time.Parse(time.DateTime, r.RecordTime)
		if i == 0 || t.Before(minTime) {
			minTime = t
		}
		if i == 0 || t.After(maxTime) {
			maxTime = t
		}
	}
	return maxTime.Sub(minTime).Minutes()
}
