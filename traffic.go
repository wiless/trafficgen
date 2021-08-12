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

// var NSectors = 61 * 3

func GenerateTrafficEvents(Ndevices int64, NSectors int, MaxWindowHr float64) {
	// Ndevices = 72000

	var Lamda float64 = 1.0 / (2 * 3600)
	fmt.Printf("\n Mean Exp Distribution is %v", 1.0/Lamda)
	fmt.Printf("\n Ndevices  %v", Ndevices)
	fmt.Printf("\n MaxWindow Hr  %v", MaxWindowHr)
	var frameInterval = 0.01 // 10ms

	fname := fmt.Sprintf(basedir + "events-cell0.csv")
	fd, _ := os.Create(fname)
	defer fd.Close()
	header, _ := vlib.Struct2HeaderLine(Event{})
	fd.WriteString(header)

	cfname := fmt.Sprintf(basedir + "events-xx.csv")
	cfd, _ := os.Create(cfname)
	defer cfd.Close()
	header, _ = vlib.Struct2HeaderLine(IEvent{})
	cfd.WriteString(header)

	for cell := 0; cell < NSectors; cell++ {

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
					events = append(events, Event{Frame: v, DeviceID: devid, SectorID: cell})
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

		if math.Mod(float64(cell), float64(ActiveBSCells)) == 0 {

			device = progressbar.Default(int64(NEvents), "Saving to "+fname)
			d3.ForEach(events, func(indx int, v Event) {
				device.Add(1)
				str, _ := vlib.Struct2String(v)
				fd.WriteString("\n" + str)
			})
		} else {
			device = progressbar.Default(int64(NEvents), fmt.Sprintf("Saving to %s : cell %d", cfname, cell))
			d3.ForEach(events, func(indx int, v Event) {
				iv := IEvent{Frame: v.Frame, SectorID: cell}
				device.Add(1)
				str, _ := vlib.Struct2String(iv)
				cfd.WriteString("\n" + str)
			})
		}

	}
}
