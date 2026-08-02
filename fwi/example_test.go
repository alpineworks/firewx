package fwi_test

import (
	"fmt"
	"time"

	"alpineworks.io/firewx/fwi"
)

// Example shows the normal use of the package: a driver carries the three
// moisture codes forward from day to day.
func Example() {
	// A fire season starts with the standard spring start-up codes.
	s := fwi.NewState()

	// Give the driver one noon observation. It returns the full daily output and
	// carries the codes into its own state.
	out := s.Step(fwi.Weather{
		Temperature: 17, // degrees Celsius
		Humidity:    42, // percent
		Wind:        25, // kilometres per hour
		Rain:        0,  // millimetres in the last 24 hours
		Month:       time.April,
		Latitude:    40, // decimal degrees, positive in the north
	})

	fmt.Printf("FFMC=%.1f DMC=%.1f DC=%.1f FWI=%.1f\n", out.FFMC, out.DMC, out.DC, out.FWI)
	// Output:
	// FFMC=87.6 DMC=8.5 DC=19.0 FWI=10.0
}

// ExampleState_Step shows the carryover: each day uses the previous day's codes,
// so the driver must receive an uninterrupted daily stream.
func ExampleState_Step() {
	s := fwi.NewState()

	// Day one is warm and dry, so the codes rise.
	day1 := s.Step(fwi.Weather{Temperature: 17, Humidity: 42, Wind: 25, Month: time.April, Latitude: 40})

	// Day two adds rain, so the Fine Fuel Moisture Code falls.
	day2 := s.Step(fwi.Weather{Temperature: 20, Humidity: 21, Wind: 25, Rain: 2.4, Month: time.April, Latitude: 40})

	fmt.Printf("day 1 FFMC=%.1f\n", day1.FFMC)
	fmt.Printf("day 2 FFMC=%.1f\n", day2.FFMC)
	// Output:
	// day 1 FFMC=87.6
	// day 2 FFMC=86.2
}

// ExampleFineFuelMoistureCode shows the pure function for one moisture code. The
// caller gives yesterday's code and today's noon weather.
func ExampleFineFuelMoistureCode() {
	// Yesterday's code was 85. Today is warm, dry and windy.
	today := fwi.FineFuelMoistureCode(85, 17, 42, 25, 0)

	fmt.Printf("%.1f\n", today)
	// Output: 87.6
}
