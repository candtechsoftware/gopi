package util

import "fmt"

func FormatChange(value float64) string {
	if value > 0 {
		return fmt.Sprintf("+%.2f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

func FormatFloat(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func CalculatePercentageChange(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return ((current - previous) / previous) * 100
}
