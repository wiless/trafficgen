package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
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
	SectorID int
}

type IEvent struct {
	Frame    int64
	SectorID int
}

type LinkProfile struct {
	RxNodeID                                                                                                              int
	TxID                                                                                                                  int
	Distance                                                                                                              float64
	IndoorDistance, UEHeight                                                                                              float64
	IsLOS                                                                                                                 bool
	CouplingLoss, Pathloss, O2I, InCar, ShadowLoss, TxPower, BSAasgainDB, UEAasgainDB, TxGCSaz, TxGCSel, RxGCSaz, RxGCSel float64
}

type RelayNode struct {
	UElocation
	FrequencyGHz float64
}

var NObservationSecs float64 = 2 * 3600 // seconds
var basedir string

var GENERATE bool

type EventList []Event
type IEventList []IEvent

var Nsamples = 15
var indir string // Reference for Cell0 results 72k device
var ilinks map[int]CellMap
var ilinksCell0 vLinkFiltered
var LIVE bool
var verbose bool

func bye() {
	fmt.Println()
}

var MaxWindowHr float64
var hrs float64

func init() {
	basedir = "./data/"
	indir = "./n70k/"
	/// EVENT RELATED
	flag.BoolVar(&GENERATE, "generate", false, "Generate Events files -generate=true (default false)")
	flag.StringVar(&basedir, "basedir", basedir, "Base Dir for events and interference stats")
	flag.StringVar(&indir, "indir", indir, "In DIR for active cell to process")
	flag.BoolVar(&LIVE, "live", true, "Runtime Packet Generation or read from files events-xx/cell-0.csv")
	flag.BoolVar(&verbose, "v", false, "Verbose -v=false for silent")

	flag.Float64Var(&hrs, "hr", 1.0, "Event Observation in Hrs -hr=0.5 for 30mins")
	flag.Parse()

	indir += "/"
	basedir += "/"
	rand.Seed(time.Now().Unix())
	defer bye()
	fmt.Println("Base DIR ", basedir)
	fmt.Println("In DIR ", indir)
	fmt.Println("GENERATE EVENT ? ", GENERATE)
	fmt.Println("Live Generate  ? ", LIVE)
	fmt.Println("Observation Hour ? ", hrs)
	MaxWindowHr = hrs * 3600

}

var associationMap map[int]vlib.VectorI
var associationSLSmap map[int][]SLSprofile
var groupedEvents map[int]EventList
var frameIndex vlib.VectorI
var groupedIEvents map[int]vlib.VectorI

func main() {
	loadSysParams()

	if GENERATE {
		GenerateTrafficEvents(72500, NBsectors, MaxWindowHr) // sufficient for center cell
	}

	/// LOAD ACTIVE DEVICES in Cell 0
	// Loaded for ACTIVE DEVICE Information  72k devices
	associationMap = make(map[int]vlib.VectorI)
	associationSLSmap := make(map[int][]SLSprofile)
	cell0links := make(vLinkFiltered, 0) // Ideally will have 0,61,122 devices
	ilinksCell0 = make(vLinkFiltered, 0) // Links of Cell 0 devices interfering to adjacent sectors

	pbar := progressbar.Default(int64(itucfg.NumUEperCell*3), "Center Cell UEs")
	counter := 0
	d3.ForEachParse(indir+"hybridnewsls-mini.csv", func(s SLSprofile) {
		/// backward compatibility
		l := LinkFiltered{RxNodeID: s.RxNodeID, TxID: s.BestRSRPNode, CouplingLoss: s.BestCouplingLoss, BestRSRPNode: s.BestRSRPNode}
		var isConnectedToRelay bool = false

		if s.FreqInGHz > 0 {
			isConnectedToRelay = true
		}
		// fmt.Printf("\n Device %d | Connected to %v on %v | Is RelayLink ? %v", s.RxNodeID, s.BestRSRPNode, s.FreqInGHz, isConnectedToRelay)
		if math.Mod(float64(l.BestRSRPNode), float64(ActiveBSCells)) == 0 || isConnectedToRelay {
			cell0links = append(cell0links, l)
			index := associationMap[l.BestRSRPNode]
			index = append(index, counter)
			associationMap[l.BestRSRPNode] = index
			// fmt.Printf("\t Added  %d devices to %d", len(index), l.BestRSRPNode)
			/// Append SLS for the each TxNodes, Relays
			nodes := associationSLSmap[l.BestRSRPNode]
			associationSLSmap[l.BestRSRPNode] = append(nodes, s)
			counter++
		}
		pbar.Add(1)
	})

	fmt.Printf("\n Total Associations %d %d \n ", len(associationMap), counter)
	fmt.Printf("\n Total Nodes in 0 %d  \n ", len(associationMap[0]))
	fmt.Printf("\n Total Nodes in 61 %d  \n ", len(associationMap[61]))
	fmt.Printf("\n Total Nodes in 122 %d \n ", len(associationMap[122]))

	pbar = progressbar.Default(int64(counter*3), "Scan Links")
	// Inteference links to adjacent sectors , remove devices that are already connected to RELAY
	node0 := associationMap[0]
	node1 := associationMap[61]
	node2 := associationMap[122]

	d3.ForEachParse(indir+"linkproperties-mini-filtered.csv", func(l LinkFiltered) {

		if node0.Contains(l.RxNodeID) || node1.Contains(l.RxNodeID) || node2.Contains(l.RxNodeID) {

			if l.TxID != l.BestRSRPNode { // Not found in 122
				// Link property of device connected to SECTOR 0, 61 and 122 and interfering each other
				ilinksCell0 = append(ilinksCell0, l)
			}
		}

		pbar.Add(1)
	})

	frameIndex = LoadAndFilterEvents(true)
	fmt.Printf("\n FrameEvents : %d ", len(frameIndex))

	////  INTERFERENCE RELATED
	/// LOAD INTERFERENCE related paramters
	ilinks = LoadULInterferenceLinks(basedir + "isectorproperties.csv")
	MeanIPerSectordBm = GetMeanInterference(ilinks)

	/// LOAD EVENTS

	/// EVALUTE SINR per Frame
	// frame := frameIndex[0]
	Nframes := len(frameIndex)
	type FrameLog struct {
		Frame        int64
		RxNodeID     int
		FrequencyGHz float64
		SectorID     int
		SINR         float64
		NIevents     int
		Nevents      int
		NInterferers int
	}
	rfd, _ := os.Create(indir + "eventsinr.csv")
	header, _ := vlib.Struct2HeaderLine(FrameLog{})
	rfd.WriteString(header)
	defer rfd.Close()
	pbar = progressbar.Default(int64(Nframes), "Event Frame : "+"eventsinr.csv")
	for k := 0; k < Nframes; k++ {
		frame := frameIndex[k]
		events := groupedEvents[frame]
		isectors, _ := groupedIEvents[frame]

		fmt.Printf("\nFrame %d | NEvents %d : %v", frame, len(events), events)
		for _, e := range events {
			// {34992 61874 0}
			indx := associationMap[e.SectorID][e.DeviceID]
			selUElink := cell0links[indx]

			tmpsls := associationSLSmap[e.SectorID][0] // Pick first entry to check the frequency
			currentfreqGHz := tmpsls.FreqInGHz
			if currentfreqGHz == 0 {
				fmt.Printf("\n\t Device#%d=>RxNodeID %d, Sector %d Channel[%v]", e.DeviceID, selUElink.RxNodeID, selUElink.BestRSRPNode, currentfreqGHz)
				fmt.Printf("\n\t Interfering sectors %v ", isectors)
			}

			// rxnodeid := Loopkup(e.DeviceID)

			// ISectors  [8 12 24 35 38 52 53 55 74 98 110 113 135 143 144 152 155 173]
			ievents := d3.Filter(events, func(d Event) bool {
				if (d.SectorID == e.SectorID) && (d.DeviceID == e.DeviceID) {
					// skip the same device
					return false
				}
				iFrequency := associationSLSmap[d.SectorID][0].FreqInGHz
				// fmt.Printf("\n\tI.Device#%d=>RxNodeID ??, Sector %d Channel[%v]", d.DeviceID, d.SectorID, iFrequency)
				if currentfreqGHz != iFrequency {
					fmt.Printf("**Different Channel***")
					// e.g. Sector 0 and any Relay Device will be on separate frequency , so interference
					return false
				}
				if d.SectorID == e.SectorID {
					// Multiple devices of same sector (Sector 0 or Same Relay Node )
					return DoesCollides(e.DeviceID, d.DeviceID)
				}
				// if Same Frequency but different Sector ID .. they will still co-incide like two different RELAY simulataneous transmission
				return true

			}).(EventList)
			if len(events) > 1 {
				fmt.Printf("\n\t Incell Sectors %v ", ievents)
			}

			iRxnodeIDs := vlib.NewVectorI(len(ievents))
			d3.ForEach(ievents, func(i int, ie Event) {
				//
				indx := associationMap[ie.SectorID][ie.DeviceID]
				iRxnodeIDs[i] = cell0links[indx].RxNodeID
				if len(events) > 1 {
					fmt.Printf("\n\t\t Other I.DeviceID#%d=> rxids %v@Sector %d", ie.DeviceID, cell0links[indx].RxNodeID, ie.SectorID)
				}
			})

			// devIDs := d3.FlatMap(ievents, "DeviceID").([]int)
			/// map devIDs to rxNodeIDs

			result1 := EvaluateTotalI(selUElink, isectors...) // [8 12 24 35 38 52 53 55 74 98 110 113 135 143 144 152 155 173]
			result2 := EvaluateSINR(selUElink, iRxnodeIDs...) // 18823, 33748 // {34992 18823 61} {34992 33748 122}
			result2r := EvaluateRelaySINR(selUElink, iRxnodeIDs...)
			// All SINR
			// fmt.Printf("\n SINR Adjacent Cells   %#v", result1)
			// fmt.Printf("\n SINR Adjacent Sectors [%d Devices] %#v", len(iRxnodeIDs), result2)
			// fmt.Printf("\n SINR Relay[%d] @ %v : %#v", selUElink.BestRSRPNode, currentfreqGHz, result2r)
			noiseLinear := bs_N0
			if currentfreqGHz > 0 {
				// relay is the receiver
				noiseLinear = ue_N0
			}
			totalI := vlib.Db(vlib.InvDb(result1.I) + vlib.InvDb(result2.I) + vlib.InvDb(result2r.I) + noiseLinear)
			result3 := SINR{S: result1.S, I: totalI, SINRdB: result1.S - totalI}

			fl := FrameLog{Frame: e.Frame,
				RxNodeID:     selUElink.RxNodeID,
				FrequencyGHz: currentfreqGHz,
				SectorID:     e.SectorID,
				SINR:         result3.SINRdB,
				NIevents:     len(iRxnodeIDs),
				Nevents:      len(events),
				NInterferers: len(isectors),
			}

			// fmt.Printf("\nEffective  = %#v dBm", result3)
			if verbose {
				fmt.Printf("\n\t Effecting SINR %#v ", fl)
			}

			str, _ := vlib.Struct2String(fl)
			rfd.WriteString("\n" + str)

		}
		// pbar.Add(1)
	}

}

func LoadULInterferenceLinks(fname string) map[int]CellMap {
	result := make(map[int]CellMap)
	var Cell0Sec0, Cell0Sec1, Cell0Sec2 vLinkFiltered
	Cell0Sec0 = make(vLinkFiltered, 0)
	Cell0Sec1 = make(vLinkFiltered, 0)
	Cell0Sec2 = make(vLinkFiltered, 0)

	// fmt.Printf("\n Total Inteference Samples : %v ", 3*int64(simcfg.ActiveUECells)*int64(itucfg.NumUEperCell))
	pbar := progressbar.Default(3*int64(simcfg.ActiveUECells)*int64(itucfg.NumUEperCell), "Load ILinks properties")

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

func LoadAndFilterEvents(live bool) vlib.VectorI {

	var result vlib.VectorI

	// Grouping of EVENTS based on FRAME
	groupedEvents = make(map[int]EventList)
	groupedIEvents = make(map[int]vlib.VectorI)
	if !live {
		allev := make([]Event, 0)
		d3.ForEachParse(basedir+"events-cell0.csv", func(ev Event) {
			if ev.DeviceID < associationMap[ev.SectorID].Len() {
				allev = append(allev, ev)
			}
		})

		d3.ForEach(allev, func(ev Event) {
			// ev.DeviceID>  remove events of devices > actula device in cell
			evlist, ok := groupedEvents[int(ev.Frame)]
			evlist = append(evlist, ev)
			groupedEvents[int(ev.Frame)] = evlist
			if !ok {
				result = append(result, int(ev.Frame))
			}

		})

		pbar1 := progressbar.Default(int64(MaxWindowHr/0.01), "Loading events-xx.csv")
		count := 0
		d3.ForEachParse(basedir+"events-xx.csv", func(ev IEvent) {
			pbar1.Add(int(float64(ev.Frame)))
			count++

			evlist, _ := groupedIEvents[int(ev.Frame)]

			if !evlist.Contains(ev.SectorID) {
				evlist = append(evlist, ev.SectorID)
			}

			groupedIEvents[int(ev.Frame)] = evlist

		})
	} else {

		var sectorIDs vlib.VectorI
		totalUE := 0
		pbar1 := progressbar.Default(int64(len(associationMap)), "Live Events Cell 0")
		for sector, v := range associationMap {
			sectorIDs = append(sectorIDs, sector)
			// NdevicesPerSector := math.Ceil(1.5 * float64(Ndevices) / 3)
			NdevicesPerSector := v.Len()
			totalUE += NdevicesPerSector
			// fmt.Printf("\n Sector %d - Generating Live Traffics %v ", sector, NdevicesPerSector)
			allev := GenerateLiveTrafficEvents(int64(NdevicesPerSector), sector, MaxWindowHr)

			d3.ForEach(allev, func(ev Event) {
				// ev.DeviceID>  remove events of devices > actula device in cell
				if ev.DeviceID < associationMap[ev.SectorID].Len() {
					evlist, ok := groupedEvents[int(ev.Frame)]
					evlist = append(evlist, ev)
					groupedEvents[int(ev.Frame)] = evlist
					if !ok {
						result = append(result, int(ev.Frame))
					}
				}

			})
			pbar1.Add(1)

		}

		var NdevicesPerSector int = int(float64(totalUE) / 3)
		pbar1 = progressbar.Default(int64(NBsectors), "Adjacent Cell Events")
		for sector := 0; sector < NBsectors; sector++ {
			pbar1.Describe(fmt.Sprintf("Adjacent Sector %d", sector))
			if !sectorIDs.Contains(sector) {
				// fmt.Printf("\nIevents for Adjacent Sector %d", sector)
				iallev := GenerateLiveTrafficEvents(int64(NdevicesPerSector), sector, MaxWindowHr)
				// fmt.Printf("Generated I Events : %d for %d", len(iallev), sector)
				d3.ForEach(iallev, func(ev Event) {

					evlist, _ := groupedIEvents[int(ev.Frame)]

					if !evlist.Contains(ev.SectorID) {
						// Avoid duplicate events in Interference cell
						evlist = append(evlist, ev.SectorID)
					}

					groupedIEvents[int(ev.Frame)] = evlist

				})
			}
			pbar1.Add(1)

		}

	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result

}

// DoesCollides returns if the two devices collide in the same sector
func DoesCollides(device1, device2 int) bool {
	return false
}
