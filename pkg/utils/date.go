package utils

import (
	"time"
)

func ValidateDate(date time.Time) bool {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	ninetyDaysAgo := today.AddDate(0, 0, -90)

	return !date.Before(ninetyDaysAgo)
}

func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func FormatDate(date time.Time) string {
	return date.Format("2006-01-02")
}
