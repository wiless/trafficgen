package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/schollz/progressbar"
	"github.com/wiless/d3"
)

var Iprofilefname string

type Event struct {
	Frame    int64
	DeviceID int
}

var NObservationSecs float64 = 2 * 3600 // seconds
var Ndevices int64
var basedir string

func bye() {
	fmt.Printf("\n")
}
func init() {
	basedir = "./data/"
	indir = "./n70k/"
	rand.Seed(time.Now().Unix())
	defer bye()
}

type EventList []Event

var Nsamples = 10
var NSectors = 61
var indir string // Reference for Cell0 results 72k device
var ilinks map[int]CellMap
var ilinksCell0 vLinkFiltered

func main() {
	loadSysParams()
	NUEs := 600
	ilinks = LoadULInterferenceLinks(basedir + "linkproperties-mini-filtered.csv")

	// Loaded for 72k devices
	cell0links := make(vLinkFiltered, 0) // Ideally will have 0,61,122 devices
	ilinksCell0 = make(vLinkFiltered, 0) // Links of Cell 0 devices interfering to adjacent sectors
	pbar := progressbar.Default(int64(NUEs*3), "Center Cell UEs")
	d3.ForEachParse(indir+"linkproperties-mini-filtered.csv", func(l LinkFiltered) {
		if math.Mod(float64(l.BestRSRPNode), float64(ActiveBSCells)) == 0 && l.BestRSRPNode == l.TxID {
			// Link property of device connected to SECTOR 0, 61 and 122
			cell0links = append(cell0links, l)
		}
		if math.Mod(float64(l.BestRSRPNode), float64(ActiveBSCells)) == 0 && l.BestRSRPNode != l.TxID {
			// Link property of device connected to SECTOR 0, 61 and 122 and interfering each other
			ilinksCell0 = append(ilinksCell0, l)
		}
		pbar.Add(1)
	})

	SaveSINRProfiles("ulsinr.csv", cell0links, ilinks)

	// d3.ForEach(ilinksCell0, func(lp LinkFiltered) {

	// 	fmt.Println(lp)
	// })

	/// Evaluate mean Interference for all Icells

	MeanInterference = GetMeanInterference(ilinks)
	fmt.Println()
	result1 := EvaluateTotalI(cell0links[0], 10)
	result2 := EvaluateSINR(cell0links[0], 184, 185)
	fmt.Printf("\nTotalI = %#v dBm", result1)
	fmt.Printf("\nSINR  = %#v dBm", result2)

	// fmt.Printf("%#v", ilinks)

}

// func CalculateSINR(lp LinkFiltered, activeBSnodes []int) SINRInfo {
// 	Iprofilefname = basedir + "linkproperties-mini-filtered.csv"
// 	linkprofiles := LoadULInterferenceLinks(Iprofilefname)
// 	SaveSINRProfiles(linkprofiles)

// }

func LoadULInterferenceLinks(fname string) map[int]CellMap {
	result := make(map[int]CellMap)
	var Cell0Sec0, Cell0Sec1, Cell0Sec2 vLinkFiltered
	Cell0Sec0 = make(vLinkFiltered, 0)
	Cell0Sec1 = make(vLinkFiltered, 0)
	Cell0Sec2 = make(vLinkFiltered, 0)

	// fmt.Printf("\n Total Inteference Samples : %v ", 3*int64(simcfg.ActiveUECells)*int64(itucfg.NumUEperCell))
	pbar := progressbar.Default(3*int64(simcfg.ActiveUECells)*int64(itucfg.NumUEperCell), "Interference Links")

	fn := func(l LinkFiltered) {
		if math.Mod(float64(l.BestRSRPNode), float64(ActiveBSCells)) != 0 { // Remove Center Cell USERs

			if l.TxID == 0 && l.BestRSRPNode != 0 {
				Cell0Sec0 = append(Cell0Sec0, l)
			}
			if l.TxID == ActiveBSCells && l.BestRSRPNode != ActiveBSCells {
				Cell0Sec1 = append(Cell0Sec1, l)
			}
			if l.TxID == 2*ActiveBSCells && l.BestRSRPNode != 2*ActiveBSCells {
				Cell0Sec2 = append(Cell0Sec2, l)
			}
		}
		pbar.Add(1)
	}
	// Filter USERs based on sector interference..
	d3.ForEachParse(fname, fn)

	// Create Map for sector interference sector to Map of Each Adjacent cells
	// Equivalent of d3.Group
	result[0] = make(CellMap)
	d3.ForEach(Cell0Sec0, func(l LinkFiltered) {
		tmp := result[0][l.BestRSRPNode]
		tmp = append(tmp, l)
		result[0][l.BestRSRPNode] = tmp
	})

	result[ActiveBSCells] = make(CellMap)
	d3.ForEach(Cell0Sec1, func(l LinkFiltered) {
		tmp := result[ActiveBSCells][l.BestRSRPNode]
		tmp = append(tmp, l)
		result[ActiveBSCells][l.BestRSRPNode] = tmp
	})

	result[2*ActiveBSCells] = make(CellMap)
	d3.ForEach(Cell0Sec2, func(l LinkFiltered) {
		tmp := result[2*ActiveBSCells][l.BestRSRPNode]
		tmp = append(tmp, l)
		result[ActiveBSCells*2][l.BestRSRPNode] = tmp
	})

	fmt.Printf("\n Interferers to Cell 0 | Sector 0 : %d", len(Cell0Sec0))
	fmt.Printf("\n Interferers to Cell 0 | Sector 1 : %d", len(Cell0Sec1))
	fmt.Printf("\n Interferers to Cell 0 | Sector 2 : %d\n", len(Cell0Sec2))
	return result
}
