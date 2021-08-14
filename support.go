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

func (ue UElocation) Location3D() vlib.Location3D {
	return vlib.Location3D{ue.X, ue.Y, ue.Z}
}

var ueNoiseFdB, bsNoiseFdB float64
var itucfg config.ITUconfig
var simcfg config.SIMconfig

// var bslocs []BSlocation
var bsTxPowerdBm, ueTxPowerdBm float64
var NBsectors int
var ActiveBSCells int
var ue_N0, bs_N0 float64     // Total N0 in linear scale
var UE_N0dB, BS_N0dB float64 // Total Noise in dB

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
	ueNoiseFdB = itucfg.UENoiseFigureDb // For Downlink
	bsNoiseFdB = itucfg.BSNoiseFigureDb // For Uplink

	UE_N0dB = -174 + vlib.Db(BW*1e6) + ueNoiseFdB // Noise at the UE device
	ue_N0 = vlib.InvDb(UE_N0dB)

	BS_N0dB = -174 + vlib.Db(BW*1e6) + bsNoiseFdB // Noise at the BS
	bs_N0 = vlib.InvDb(BS_N0dB)

	bsTxPowerdBm = itucfg.TxPowerDbm
	ueTxPowerdBm = itucfg.UETxDbm

	fmt.Println("Total Active Sectors ", NBsectors)

	fmt.Println("DL: N0 @ UE (dBm)", UE_N0dB)
	fmt.Println("UL: N0 @ BS (dBm)", BS_N0dB)
}
