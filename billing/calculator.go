package billing

import (
	"math"
)

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func PlanAndDurationToPrice(plan string, duration string) float64 {
	price := float64(0.00)
	if duration == "monthly" {
		switch plan {
		case "Personal": // now "Personal"
			price = 9.99 * 1
		case "Consultant": // now "Consultant"
			price = 18.99 * 1
		case "Business": // now "Business"
			price = 35.99 * 1
		case "Growing Business": // now "Growing Business"
			price = 54.99 * 1
		}
	} else {
		switch plan {
		case "Personal": // now "Personal"
			price = 7.99 * 12
		case "Consultant": // now "Consultant"
			price = 15.99 * 12
		case "Business": // now "Business"
			price = 29.99 * 12
		case "Growing Business": // now "Growing Business"
			price = 43.99 * 12
		}
	}

	return toFixed(price, 2)
}
