package firewx_test

import (
	"fmt"

	firewx "alpineworks.io/firewx"
)

// Example shows the shared observation type. An observation carries SI units,
// and each derived value is absent when an input is absent.
func Example() {
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(20)),
		RelativeHumidity: firewx.Some(firewx.Percent(50)),
	}

	// The dew point comes from the temperature and the humidity.
	dew := o.DewPoint().Must()

	// The vapour pressure deficit comes from the same two inputs.
	vpd := o.VaporPressureDeficit().Must()

	fmt.Printf("dew point=%.1f C\n", dew)
	fmt.Printf("VPD=%.2f kPa\n", vpd)
	// Output:
	// dew point=9.3 C
	// VPD=1.17 kPa
}

// ExampleOpt shows the optional value type. A present value holds data; an
// absent value holds nothing. This replaces the use of NaN for missing data.
func ExampleOpt() {
	present := firewx.Some(firewx.Celsius(21.5))
	absent := firewx.None[firewx.Celsius]()

	if v, ok := present.Get(); ok {
		fmt.Printf("present: %.1f\n", v)
	}
	fmt.Printf("absent is valid: %t\n", absent.Valid())
	// Output:
	// present: 21.5
	// absent is valid: false
}

// ExampleCelsius_Fahrenheit shows a unit conversion. Each physical quantity has
// a named type, so the compiler stops a unit mistake.
func ExampleCelsius_Fahrenheit() {
	boiling := firewx.Celsius(100)
	fmt.Printf("%.0f F\n", boiling.Fahrenheit())
	// Output: 212 F
}

// ExampleAdjustWindHeight shows the wind-height correction. It raises a wind
// speed from a low sensor to the 10 m reference the Canadian system assumes.
func ExampleAdjustWindHeight() {
	// A 3 m sensor over suburban roughness reads 4 m/s. Correct it up to 10 m.
	corrected := firewx.AdjustWindHeight(4, 3, firewx.HeightFWI, firewx.RoughnessSuburban)
	fmt.Printf("%.1f m/s\n", corrected)
	// Output: 6.7 m/s
}
