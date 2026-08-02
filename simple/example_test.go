package simple_test

import (
	"fmt"

	firewx "alpineworks.io/firewx"
	"alpineworks.io/firewx/simple"
)

// Example shows two of the single-equation indices. Each pure function takes the
// measurement in the units of its published equation.
func Example() {
	// Fosberg uses degrees Fahrenheit, percent, and miles per hour.
	ffwi := simple.FosbergIndex(85, 20, 15)

	// Angström uses degrees Celsius and percent. A low value means more danger.
	ang := simple.AngstromIndex(25, 40)

	fmt.Printf("Fosberg=%.2f\n", ffwi)
	fmt.Printf("Angstrom=%.1f (%s)\n", ang, ang.Class())
	// Output:
	// Fosberg=37.53
	// Angstrom=2.2 (high)
}

// ExampleKBDIState_Step shows a cumulative index. The driver carries the drought
// index forward from day to day.
func ExampleKBDIState_Step() {
	// Start a station with 50 inches of mean annual rainfall. The index starts
	// at zero (a saturated soil).
	s := simple.NewKBDIState(50)

	// One hot, dry day raises the index.
	s.Step(90, 0) // 90 F, no rain

	fmt.Printf("KBDI=%.2f (%s)\n", s.Index, s.Index.Class())
	// Output: KBDI=24.92 (low)
}

// ExampleFosbergFromObs shows the FromObs helper. It takes an observation in SI
// units and returns an absent value if an input is absent.
func ExampleFosbergFromObs() {
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(29.4)), // about 85 F
		RelativeHumidity: firewx.Some(firewx.Percent(20)),
		// No wind, so the index is absent.
	}

	if _, ok := simple.FosbergFromObs(o).Get(); ok {
		fmt.Println("present")
	} else {
		fmt.Println("absent without wind")
	}
	// Output: absent without wind
}
