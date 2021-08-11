package main

import (
	"fmt"
	"log"

	"github.com/5gif/config"
	"github.com/wiless/vlib"
)

var BW float64 // Can be different than itucfg.BandwidthMHz, based on uplink/downlink

type BSlocation struct {
	ID                                     int
	X, Y, Z, TxPowerdBm, Hdirection, VTilt float64
	Active, Alias                          int
}

type UElocation struct {
	ID            int
	X, Y, Z       float64
	Indoor, InCar bool
	BSdist        float64
	GCellID       int

	// Name string `csv:"name"`
	// Address
}

var RxNoisedB, TxNoisedB float64
var itucfg config.ITUconfig
var simcfg config.SIMconfig

// var bslocs []BSlocation
var bsTxPowerdBm, ueTxPowerdBm float64
var NBsectors int
var ActiveBSCells int
var N0, UL_N0 float64 // N0 in linear scale
var UL_N0dB float64

var Er = func(err error) {
	if err != nil {
		log.Println("Error ", err)
	}
}
var err error

func loadSysParams() {
	simcfg, err = config.ReadSIMConfig(basedir + "sim.cfg")
	Er(err)
	ActiveBSCells = simcfg.ActiveBSCells
	fmt.Println("Active BSCells = ", simcfg.ActiveBSCells)
	fmt.Println("Active UECells = ", simcfg.ActiveUECells)
	itucfg, _ = config.ReadITUConfig(basedir + "itu.cfg")
	// ----
	// d3.CSV(basedir+"bslocation.csv", &bslocs) // needed ?
	NBsectors = ActiveBSCells * 3 // len(bslocs)

	BW = itucfg.BandwidthMHz
	RxNoisedB = itucfg.UENoiseFigureDb // For Downlink
	TxNoisedB = itucfg.BSNoiseFigureDb // For Uplink

	N0dB := -174 + vlib.Db(BW*1e6) + RxNoisedB // in linear scale
	N0 = vlib.InvDb(N0dB)

	UL_N0dB = -174 + vlib.Db(BW*1e6) + TxNoisedB // in linear scale
	UL_N0 = vlib.InvDb(UL_N0dB)

	bsTxPowerdBm = itucfg.TxPowerDbm
	ueTxPowerdBm = itucfg.UETxDbm

	fmt.Println("Total Active Sectors ", NBsectors)

	fmt.Println("DL : N0 (dB)", N0dB)
	fmt.Println("UL : N0 (dB)", UL_N0dB)
}
