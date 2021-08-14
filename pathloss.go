package main

import (
	"math"
	"math/rand"
)

// Implemenation based on 3GPP TS. 38.901 and M.2412

// For UMa-mMTC
// Percentage of high loss and  low loss building type (for both Configs TABLE 5, Sec 8.4 )
//  20% high loss, 80% low loss

var hUT = 1.5

func IsLOS(distance2d float64) bool {
	if distance2d < 18 {
		return true
	}

	prlos := 18/distance2d + math.Exp(-distance2d/36)*(1-18/distance2d)
	if rand.Float64() <= prlos {
		return true
	}
	return false

}

func PLNLOS(d2D, fcghz, hBS float64) float64 {
	var d3Distance = math.Sqrt(math.Pow((hBS-hUT), 2) + math.Pow(d2D, 2))
	// var ddBP = BPDist(fcghz, hBS)
	d3Distance = math.Sqrt(math.Pow((hBS-hUT), 2) + math.Pow(d2D, 2))

	var LOS = PL(d2D, fcghz, hBS)

	var PLN = 35.4*math.Log10(d3Distance) +
		22.4 +
		21.3*math.Log10(fcghz) -
		0.3*(hUT-1.5)

	return math.Max(LOS, PLN)
}
func PL(d2D, fcghz, hBS float64) float64 {
	var d3Distance = math.Sqrt(math.Pow((hBS-hUT), 2) + math.Pow(d2D, 2))
	var ddBP = BPDist(fcghz, hBS)
	d3Distance = math.Sqrt(math.Pow((hBS-hUT), 2) + math.Pow(d2D, 2))
	var PL1 = 32.4 + 21*math.Log10(d3Distance) + 20*math.Log10(fcghz)
	var PL2 = 32.4 +
		40*math.Log10(d3Distance) +
		20*math.Log10(fcghz) -
		9.5*math.Log10(math.Pow(ddBP, 2)+math.Pow((hBS-hUT), 2))

	if ddBP <= d2D {
		return PL2
	} else {
		return PL1
	}
}

var mlog = math.Log10

// For UMa-mMTC
// Percentage of high loss and  low loss building type (for both Configs TABLE 5, Sec 8.4 )
//  20% high loss, 80% low loss

// O2ICarLossDb returns the Car penetration loss in dB
// Ref M.2412 Section 3.3 μ = 9, and σP = 5
func O2ICarLossDb() float64 {
	// μ = 9, and σP = 5
	var mean float64
	mean = 9.0
	sigmaP := 5.0
	return rand.NormFloat64()*sigmaP + mean
}

func O2ILossDb(fGHz float64, d2Din float64) float64 {
	if d2Din == 0 {
		return 0
	}
	PLin := 0.5 * d2Din
	var HIGHLOSS bool = false

	if rand.Float64() > 0.2 {
		// 80% OF THE time..
		HIGHLOSS = true
	}

	if HIGHLOSS {
		Lg := 2 + 0.2*fGHz
		Lc := 5 + 4*fGHz
		sigmaP := 4.4
		mean := 0.0
		// Equation for Low Loss
		Ptw := 5 - 10*mlog(0.3*math.Pow(10, (-Lg/10.0))+0.7*math.Pow(10, (-Lc/10.0))) // for 3GPP low-loss model equation to be used..
		PLin := 0.5 * d2Din
		return Ptw + PLin + rand.NormFloat64()*sigmaP + mean
	} else {
		// 20% OF THE time..
		Lirrg := 23 + 0.3*fGHz
		Lc := 5 + 4*fGHz
		sigmaP := 6.5
		mean := 0.0
		// Equation for High Loss
		Ptw := 5 - 10*mlog(0.7*math.Pow(10, (-Lirrg/10.0))+0.3*math.Pow(10, (-Lc/10.0))) // for 3GPP high-loss model equation to be used..

		return Ptw + PLin + rand.NormFloat64()*sigmaP + mean
	}

}

func BPDist(fcghz, hBS float64) float64 {
	// Always use hBS=2m  ??
	var fcHz = fcghz * 1e9
	var hE = 1.0
	var hdBS = hBS - hE
	var hdUT = hUT - hE
	var C = 3.0 * 1e8
	return (4 * hdBS * hdUT * fcHz) / C
}
