package fems

import (
	"context"
	"fmt"
	"net/url"
	"time"

	firewx "alpineworks.io/firewx"
)

// NFDROutput is one hour of computed NFDRS output for one station. Each value is
// optional, because a station may not report every field. The fuel moistures are
// a percentage; the four indices and the two drought and greenness measures are
// dimensionless.
type NFDROutput struct {
	StationID   string
	StationName string
	Time        time.Time

	// FuelModel is the NFDRS fuel model letter, for example "Y".
	FuelModel string

	OneHourFuelMoisture      firewx.Opt[firewx.Percent]
	TenHourFuelMoisture      firewx.Opt[firewx.Percent]
	HundredHourFuelMoisture  firewx.Opt[firewx.Percent]
	ThousandHourFuelMoisture firewx.Opt[firewx.Percent]

	WoodyFuelMoisture      firewx.Opt[firewx.Percent]
	HerbaceousFuelMoisture firewx.Opt[firewx.Percent]

	// KBDI is the Keetch-Byram Drought Index, 0 to 800.
	KBDI firewx.Opt[float64]
	// GSI is the Growing Season Index, 0 to 1.
	GSI firewx.Opt[float64]

	IgnitionComponent      firewx.Opt[float64]
	EnergyReleaseComponent firewx.Opt[float64]
	SpreadComponent        firewx.Opt[float64]
	BurningIndex           firewx.Opt[float64]
}

// NFDR reads the computed NFDRS output for the request and returns one
// NFDROutput per station per hour. This is the FEMS ground truth for a check of
// the nfdrs package.
func (c *Client) NFDR(ctx context.Context, req Request) ([]NFDROutput, error) {
	extra := url.Values{}
	extra.Set("dataset", "all")
	extra.Set("fuelModels", "Y")

	table, err := c.fetchCSV(ctx, "download-nfdr", extra, req)
	if err != nil {
		return nil, err
	}

	out := make([]NFDROutput, 0, len(table.rows))
	for _, row := range table.rows {
		if table.cell(row, "stationId") == "" {
			continue
		}
		o, err := nfdrOutput(table, row)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func nfdrOutput(table *csvTable, row []string) (NFDROutput, error) {
	t, err := time.Parse(time.RFC3339, table.cell(row, "observationTime"))
	if err != nil {
		return NFDROutput{}, fmt.Errorf("fems: time %q: %w", table.cell(row, "observationTime"), err)
	}

	o := NFDROutput{
		StationID:   table.cell(row, "stationId"),
		StationName: table.cell(row, "stationName"),
		Time:        t.UTC(),
		FuelModel:   table.cell(row, "fuelModelType"),
	}

	o.OneHourFuelMoisture = optPercent(table.cell(row, "oneHR_TL_FuelMoisture"))
	o.TenHourFuelMoisture = optPercent(table.cell(row, "tenHR_TL_FuelMoisture"))
	o.HundredHourFuelMoisture = optPercent(table.cell(row, "hundredHR_TL_FuelMoisture"))
	o.ThousandHourFuelMoisture = optPercent(table.cell(row, "thousandHR_TL_FuelMoisture"))
	o.WoodyFuelMoisture = optPercent(table.cell(row, "woodyLFI_fuelMoisture"))
	o.HerbaceousFuelMoisture = optPercent(table.cell(row, "herbaceousLFI_fuelMoisture"))

	o.KBDI = optNumber(table.cell(row, "kbdi"))
	o.GSI = optNumber(table.cell(row, "gsi"))
	o.IgnitionComponent = optNumber(table.cell(row, "ignitionComponent"))
	o.EnergyReleaseComponent = optNumber(table.cell(row, "energyReleaseComponent"))
	o.SpreadComponent = optNumber(table.cell(row, "spreadComponent"))
	o.BurningIndex = optNumber(table.cell(row, "burningIndex"))

	return o, nil
}

func optPercent(s string) firewx.Opt[firewx.Percent] {
	if v, ok := optFloat(s); ok {
		return firewx.Some(firewx.Percent(v))
	}
	return firewx.None[firewx.Percent]()
}

func optNumber(s string) firewx.Opt[float64] {
	if v, ok := optFloat(s); ok {
		return firewx.Some(v)
	}
	return firewx.None[float64]()
}
