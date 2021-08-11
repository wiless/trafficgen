package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/schollz/progressbar"
	"github.com/wiless/d3"
	"github.com/wiless/vlib"
)

var Iprofilefname string

type Event struct {
	Frame    int64
	DeviceID int
}

type IEvent struct {
	Frame    int64
	SectorID int
}

var NObservationSecs float64 = 2 * 3600 // seconds
var Ndevices int64
var basedir string

func bye() {
	fmt.Println()
}
func init() {
	basedir = "./data/"
	indir = "./n70k/"
	rand.Seed(time.Now().Unix())
	defer bye()
}

type EventList []Event

var Nsamples = 10
var indir string // Reference for Cell0 results 72k device
var ilinks map[int]CellMap
var ilinksCell0 vLinkFiltered

func main() {
	loadSysParams()

	/// EVENT RELATED
	MaxWindowHr := 0.25 * 3600 // in Hr
	//   GenerateTrafficEvents(72000, NBsectors, MaxWindowHr) // sufficient for center cell

	// Load Sector 0 events
	ev0 := make([]Event, 0)
	ev61 := make([]Event, 0)
	ev122 := make([]Event, 0)
	allev := make([]Event, 0)
	d3.ForEachParse(basedir+"event-cell00.csv", func(ev Event) {
		ev0 = append(ev0, ev)
		allev = append(allev, ev)
	})
	d3.ForEachParse(basedir+"event-cell61.csv", func(ev Event) {
		ev61 = append(ev61, ev)
		allev = append(allev, ev)
	})
	d3.ForEachParse(basedir+"event-cell122.csv", func(ev Event) {
		ev122 = append(ev122, ev)
		allev = append(allev, ev)
	})
	interferencesectors := make([]IEvent, 0)
	pbar1 := progressbar.Default(int64(MaxWindowHr/0.01), "Loading Events")
	count := 0
	d3.ForEachParse(basedir+"event-xx.csv", func(ev IEvent) {
		pbar1.Add(int(float64(ev.Frame)))
		count++
		interferencesectors = append(interferencesectors, ev)

	})
	fmt.Printf("\nI Event %d %#v ", count, len(interferencesectors))
	fmt.Printf("\nEvent %d %#v ", len(ev0))
	fmt.Printf("\nEvent %d %#v ", len(ev61))
	fmt.Printf("\nEvent %d %#v ", len(ev122))

	fmt.Printf("\nEvent  %#v ", allev[0:10])
	sort.Slice(allev, func(i, j int) bool {
		return allev[i].Frame < allev[j].Frame
		// ev0[j], ev0[i] = ev0[i], ev0[j]
	})
	fmt.Printf("\nEvent  %#v ", allev[0:10])
	/// iframes = d3.group(
	//   d3.sort(interferencesectors, (d) => d.Frame),
	//   (d) => d.Frame
	// )

	return
	////  INTERFERENCE RELATED

	/// LOAD INTERFERENCE related paramters
	ilinks = LoadULInterferenceLinks(basedir + "isectorproperties.csv")
	MeanIPerSectordBm = GetMeanInterference(ilinks)

	// Loaded for ACTIVE DEVICE Information  72k devices
	NUEs := 600
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
	// SaveSINRProfiles("ulsinr.csv", cell0links, ilinks)

	/// LOAD EVENTS

	/// EVALUTE SINR per Frame
	fmt.Println()
	result1 := EvaluateTotalI(cell0links[0], 10)
	result2 := EvaluateSINR(cell0links[0], 184, 185)
	fmt.Printf("\nTotalI = %#v dBm", result1)
	fmt.Printf("\nSINR  = %#v dBm", result2)
	totalI := vlib.Db(vlib.InvDb(result1.I) + vlib.InvDb(result1.I) + UL_N0)
	result3 := SINR{S: result1.S, I: totalI, SINRdB: result1.S - totalI}

	fmt.Printf("\nEffective  = %#v dBm", result3)

	// fmt.Printf("%#v", ilinks)

}

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
