package main

import (
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

type Event struct {
	Frame    int64
	DeviceID int
}

var NObservationSecs float64 = 2 * 3600 // seconds
var Ndevices int64

func bye() {
	fmt.Printf("\n")
}
func init() {
	defer bye()
}

type EventList []Event

// func (a EventList) Len() int           { return len(a) }
// func (a EventList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
// func (a EventList) Less(i, j int) bool { return a[i].Frame < a[j].Frame }
func main() {
	Ndevices = 100000
	Nsamples := 10
	var MaxWindowHr float64 = 2.0 * 3600 // in Hr
	var Lamda float64 = 1.0 / (2 * 3600)
	fmt.Printf("\n Mean Exp Distribution is %v", 1.0/Lamda)
	fmt.Printf("\n Ndevices  %v", Ndevices)
	fmt.Printf("\n MaxWindow Hr  %v", MaxWindowHr)
	var frameInterval = 0.01 // 10ms

	for cell := 0; cell < 19; cell++ {

		var events EventList

		device := progressbar.Default(Ndevices, "Devices")
		var NEvents = 0
		for devid := 0; devid < int(Ndevices); devid++ {
			rndgen := rand.New(rand.NewSource(time.Now().Unix() + rand.Int63()))
			nsamples := make([]float64, 0, Nsamples)
			tEventFrameIndex := make([]int64, 0, Nsamples)
			var pvs = 0.0
			var maxcount = -1
			for j := 0; j < Nsamples; j++ {
				// nsamples[j] = rand.ExpFloat64() / Lamda
				diff := rndgen.ExpFloat64() / Lamda
				abstime := pvs + diff
				if abstime > MaxWindowHr {
					maxcount = j - 1
					break
				}
				nsamples = append(nsamples, abstime) // rndgen.ExpFloat64()/Lamda
				tEventFrameIndex = append(tEventFrameIndex, int64(math.Ceil(nsamples[j]/frameInterval)))
				pvs = abstime
			}
			if maxcount > -1 {
				for _, v := range tEventFrameIndex {
					events = append(events, Event{Frame: v, DeviceID: devid})
				}
				NEvents += len(tEventFrameIndex)

			}
			device.Add(1)

		}

		device = progressbar.Default(int64(NEvents), "Sorting")
		sort.Slice(events, func(i, j int) bool {
			device.Add(1)
			return events[i].Frame < events[j].Frame
		})
		fname := fmt.Sprintf("event-cell%02d.csv", cell)
		fd, _ := os.Create(fname)
		defer fd.Close()
		header, _ := vlib.Struct2HeaderLine(Event{})
		fmt.Fprint(fd, header)

		device = progressbar.Default(int64(NEvents), "Saving to "+fname)
		d3.ForEach(events, func(indx int, v Event) {
			// fmt.Printf("\n %d : %#v ", indx, v)
			device.Add(1)
			str, _ := vlib.Struct2String(v)
			fd.WriteString("\n" + str)
		})
	}
}
