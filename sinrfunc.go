package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/wiless/d3"
	"github.com/wiless/vlib"
)

type vLinkFiltered []LinkFiltered
type CellMap map[int]vLinkFiltered

type LinkFiltered struct {
	RxNodeID, TxID int
	CouplingLoss   float64
	BestRSRPNode   int
}

type SINRInfo struct {
	RxNodeID     int
	BestRSRPNode int
	SINRmean     float64
	SINRsnap     float64
	SINRideal    float64
}

// SaveSINRProfiles saves SINR of user links in the userlinks array, Interference samples
// are taken from the Ilinks Cell-Map
func SaveSINRProfiles(fname string, userlinks vLinkFiltered, Ilinks map[int]CellMap) {
	// Find Interfering Cells

	// MeanIPerSectordBm := GetMeanInterference(Ilinks)

	NActive := rand.Intn(ActiveBSCells*3 - 1)
	// NActive = ActiveBSCells*3 - 1 // ???
	seq := vlib.NewSegmentI(0, ActiveBSCells*3)
	rand.Shuffle(seq.Len(), func(i, j int) {
		seq[i], seq[j] = seq[j], seq[i]
	})

	SnapShotIPerSectordBm := GetSnapShotInterference(Ilinks, seq[0:NActive]...)
	fmt.Printf("\n MeanIPerSectordBm  - Sector 0   %#v", MeanInDb(MeanIPerSectordBm[0]))
	fmt.Printf("\n MeanIPerSectordBm  - Sector 61   %#v", MeanInDb(MeanIPerSectordBm[61]))
	fmt.Printf("\n MeanIPerSectordBm  - Sector 122   %#v", MeanInDb(MeanIPerSectordBm[122]))
	fmt.Printf("\n SnapShotIPerSectordBm %#v", SnapShotIPerSectordBm)
	fmt.Printf("\n Active I Sectors %d :  %#v\n", NActive, seq[0:NActive])

	fd, er := os.Create(fname)
	defer fd.Close()
	fmt.Print(er)
	header, _ := vlib.Struct2HeaderLine(SINRInfo{})
	fd.WriteString(header)
	pbar := progressbar.Default(int64(len(userlinks)))
	d3.ForEach(userlinks, func(lp LinkFiltered) {

		totalIdBm := vlib.Db(vlib.Sum(vlib.InvDbF(MeanIPerSectordBm[lp.BestRSRPNode])))
		signal := lp.CouplingLoss + ueTxPowerdBm
		SINRmean := signal - totalIdBm                                  // UL_N0dB need to be added
		SINRsnapshot := signal - SnapShotIPerSectordBm[lp.BestRSRPNode] // UL_N0dB need to be added
		SINRideal := signal - UL_N0dB
		info := SINRInfo{RxNodeID: lp.RxNodeID, BestRSRPNode: lp.BestRSRPNode, SINRmean: SINRmean, SINRsnap: SINRsnapshot, SINRideal: SINRideal}
		infostr, _ := vlib.Struct2String(info)
		fd.WriteString("\n" + infostr)
		pbar.Add(1)
	})
}

func GetMeanInterference(linkinfo map[int]CellMap) map[int]vlib.VectorF {
	MeanIPerSectordBm := make(map[int]vlib.VectorF)
	for sector := range linkinfo {
		// cm := linkinfo[sector]
		fmt.Printf("\n Current Sector %d of %d \n CellMap information  ", sector, NBsectors)
		// for i := 0; i < NBs; i++ {
		// 	v := cm[i]
		// 	fmt.Printf("\n Key %v | Value %v ", i, len(v))
		// }

		meanI := vlib.NewVectorF(NBsectors)
		for i := 0; i < NBsectors; i++ {
			adjSector := i
			allueslinks, ok := linkinfo[sector][adjSector]
			if ok && i%61 != 0 {
				// fmt.Printf("\nISector ID %d | NUEs = %v", k, len(v))
				closs := d3.Map(allueslinks, func(lf LinkFiltered) float64 {
					return vlib.InvDb(lf.CouplingLoss + itucfg.UETxDbm)

				}).([]float64)
				meanI[i] = vlib.Db(vlib.Mean(closs))
			} else {
				// fmt.Printf("\n Unknown Interfereing Sector %d \n %v", adjSector, allueslinks)
				meanI[i] = -9999.0
			}

		}
		MeanIPerSectordBm[sector] = meanI
		// fmt.Printf("\n Sector %d : Inteference dBm from adj Sectors = %v  ", sector, MeanIPerSectordBm[sector])
	}
	return MeanIPerSectordBm

}

func GetSnapShotInterference(linkinfo map[int]CellMap, activeSectors ...int) map[int]float64 {
	SnapShotIPerSectordBm := make(map[int]float64)
	for sector := range linkinfo {
		// fmt.Printf("\n Current Sector %d", sector)
		var snapShotI float64
		SnapShotIPerSectordBm[sector] = -1000
		for _, k := range activeSectors {
			v, ok := linkinfo[sector][k]
			if ok && k != sector {

				/// Pick a Random User from the "Active Adjacent Sector"
				picked := v[rand.Intn(len(v))]

				snapShotI += vlib.InvDb(picked.CouplingLoss + itucfg.UETxDbm)
			}

		}
		if snapShotI != 0 {
			SnapShotIPerSectordBm[sector] = vlib.Db(snapShotI)
		}

		fmt.Printf("\n Sector %d :  SnapShotInterference   dBm = %v @ %v UETxpower  ", sector, SnapShotIPerSectordBm[sector], itucfg.UETxDbm)
	}
	return SnapShotIPerSectordBm

}

var MeanIPerSectordBm map[int]vlib.VectorF

type SINR struct {
	S      float64
	I      float64
	SINRdB float64
}

// EvaluateSINR return Ideal SINR with Interference from adjacent sectors of Cell0
func EvaluateSINR(ulp LinkFiltered, otherdevicesID ...int) SINR {
	// Inteference from adjacent sectors ?
	totalI := 0.0
	for _, device := range otherdevicesID {
		ilp := d3.FindFirst(ilinksCell0, func(lp LinkFiltered) bool {

			found := lp.RxNodeID == device && lp.TxID == ulp.BestRSRPNode
			// if found {
			// 	fmt.Println("Found .. ", lp)
			// }
			return found
		}).(LinkFiltered)
		if ilp.RxNodeID != 0 {
			totalI += vlib.InvDb(ilp.CouplingLoss + ueTxPowerdBm)
		}

	}

	// if totalI != 0 {
	sinr := ulp.CouplingLoss + ueTxPowerdBm - vlib.Db(totalI+UL_N0)

	return SINR{S: ulp.CouplingLoss + ueTxPowerdBm, I: vlib.Db(totalI), SINRdB: sinr}
	// }
	// return ulp.CouplingLoss + ueTxPowerdBm - UL_N0dB

}

// EvaluateTotalI returns SINR based on adjacent active sectors
func EvaluateTotalI(ulp LinkFiltered, activeSectors ...int) SINR {
	var totalI float64 = 0

	sector := ulp.BestRSRPNode
	for _, k := range activeSectors {

		allueslinks, ok := ilinks[sector][k]
		if ok && k != sector {
			picked := allueslinks[rand.Intn(len(allueslinks))]
			totalI += vlib.InvDb(picked.CouplingLoss + itucfg.UETxDbm)
		}

	}
	// return ulp.CouplingLoss + ueTxPowerdBm - UL_N0dB

	sinr := ulp.CouplingLoss + ueTxPowerdBm - vlib.Db(totalI+UL_N0)
	IdBm := -1000.0
	if totalI != 0 {
		IdBm = vlib.Db(totalI)
	}
	result := SINR{S: ulp.CouplingLoss + ueTxPowerdBm, I: IdBm, SINRdB: sinr}
	return result
}

func EvaluateSINRMean(ulp LinkFiltered, activeSectors ...int) SINR {
	var totalI float64 = 0

	sector := ulp.BestRSRPNode
	for _, k := range activeSectors {

		if k != sector {
			IdBm := MeanIPerSectordBm[sector][k]
			totalI += vlib.InvDb(IdBm)
		}

	}

	sinr := ulp.CouplingLoss + ueTxPowerdBm - vlib.Db(totalI+UL_N0)
	IdBm := -1000.0
	if totalI != 0 {
		IdBm = vlib.Db(totalI)
	}
	result := SINR{S: ulp.CouplingLoss + ueTxPowerdBm, I: IdBm, SINRdB: sinr}
	return result
}

// returns the mean of values which are in dB, in dB
func MeanInDb(v vlib.VectorF) float64 {
	return vlib.Db(vlib.Mean(vlib.InvDbF(v)))

}
