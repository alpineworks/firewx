// Package rothermel implements the Rothermel surface fire spread model.
//
// The model computes the rate of spread of the head of a surface fire from a
// fuel model, the fuel moisture, the midflame wind speed, and the slope. It is
// the spread engine that the US National Fire Danger Rating System uses for the
// spread component and the burning index.
//
// The model groups the fuel particles into a dead category and a live category.
// Each category has one or more size classes. The model weights the size classes
// by surface area to find the characteristic properties of the fuel bed, then it
// solves the fire spread equation.
//
// The equations follow Andrews (2018), which gives the complete model with the
// later modifications. The equation numbers in the code refer to that report.
// The code was cross-checked against the firelab/NFDRS4 C++ program, which uses
// the same equations. The golden test uses the worked example in Andrews (2018),
// Table 17.
//
// The model weights each fuel size class by its own surface-area fraction. It
// does not group the size classes into the surface-area-to-volume bands that
// Albini (1976) uses for the net fuel load (Andrews 2018, equation 59). This
// matches firelab/NFDRS4, which weights the fixed size classes directly. The
// two methods give the same result unless two size classes in one category fall
// in the same band.
//
// References:
//   - Rothermel, R.C. 1972. A mathematical model for predicting fire spread in
//     wildland fuels. Research Paper INT-115. Ogden, UT: USDA Forest Service,
//     Intermountain Forest and Range Experiment Station.
//   - Andrews, P.L. 2018. The Rothermel surface fire spread model and associated
//     developments: a comprehensive explanation. General Technical Report
//     RMRS-GTR-371. Fort Collins, CO: USDA Forest Service, Rocky Mountain
//     Research Station.
//   - Albini, F.A. 1976. Estimating wildfire behavior and effects. General
//     Technical Report INT-30. Ogden, UT: USDA Forest Service.
//   - Anderson, H.E. 1969. Heat transfer and fire spread. Research Paper INT-69.
//     Ogden, UT: USDA Forest Service.
package rothermel
