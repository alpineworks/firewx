package simple

// schemaVersion is the schema version for each stateful driver in this package.
// Increase it when the meaning of a field changes. A larger version lets the
// code identify an old persisted state and change it correctly. Without the
// version, the code reads the old state as the new state and gives wrong output.
const schemaVersion = 1

// DangerClass is a fire danger category.
//
// Each index has its own scale and its own limits. DangerClass gives a common
// set of categories. A caller can compare two indices with it. The Class method
// of each index shows the limit for each category. A caller who wants different
// limits can use the raw value to make a different category.
type DangerClass int

const (
	ClassLow DangerClass = iota
	ClassModerate
	ClassHigh
	ClassVeryHigh
	ClassExtreme
)

func (c DangerClass) String() string {
	switch c {
	case ClassLow:
		return "low"
	case ClassModerate:
		return "moderate"
	case ClassHigh:
		return "high"
	case ClassVeryHigh:
		return "very high"
	case ClassExtreme:
		return "extreme"
	default:
		return "unknown"
	}
}
