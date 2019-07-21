package decor

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/VividCortex/ewma"
)

const (
	perSecond = "/s"
)

type speedType struct {
	sizeT     fmt.Formatter
	perSecond string
}

func (self *speedType) Format(st fmt.State, verb rune) {
	self.sizeT.Format(st, verb)
	io.WriteString(st, self.perSecond)
}

func sizePerSecond(sizeT fmt.Formatter) fmt.Formatter {
	return &speedType{
		sizeT:     sizeT,
		perSecond: perSecond,
	}
}

// EwmaSpeed exponential-weighted-moving-average based speed decorator.
// Note that it's necessary to supply bar.Incr* methods with incremental
// work duration as second argument, in order for this decorator to
// work correctly. This decorator is a wrapper of MovingAverageSpeed.
func EwmaSpeed(unit int, fmt string, age float64, wcc ...WC) Decorator {
	return MovingAverageSpeed(unit, fmt, ewma.NewMovingAverage(age), wcc...)
}

// MovingAverageSpeed decorator relies on MovingAverage implementation
// to calculate its average.
//
//	`unit` one of [0|UnitKiB|UnitKB] zero for no unit
//
//	`fmt` printf compatible verb for value, like "%f" or "%d"
//
//	`average` MovingAverage implementation
//
//	`wcc` optional WC config
//
// fmt examples:
//
//	unit=UnitKiB, fmt="%.1f"  output: "1.0MiB/s"
//	unit=UnitKiB, fmt="% .1f" output: "1.0 MiB/s"
//	unit=UnitKB,  fmt="%.1f"  output: "1.0MB/s"
//	unit=UnitKB,  fmt="% .1f" output: "1.0 MB/s"
//
func MovingAverageSpeed(unit int, fmt string, average MovingAverage, wcc ...WC) Decorator {
	var wc WC
	for _, widthConf := range wcc {
		wc = widthConf
	}
	wc.Init()
	if fmt == "" {
		fmt = "%.0f"
	}
	d := &movingAverageSpeed{
		WC:      wc,
		unit:    unit,
		fmt:     fmt,
		average: average,
	}
	return d
}

type movingAverageSpeed struct {
	WC
	unit        int
	fmt         string
	average     ewma.MovingAverage
	msg         string
	completeMsg *string
}

func (d *movingAverageSpeed) Decor(st *Statistics) string {
	if st.Completed {
		if d.completeMsg != nil {
			return d.FormatMsg(*d.completeMsg)
		}
		return d.FormatMsg(d.msg)
	}

	speed := d.average.Value()

	switch d.unit {
	case UnitKiB:
		d.msg = fmt.Sprintf(d.fmt, SizeB1024(math.Round(speed)))
	case UnitKB:
		d.msg = fmt.Sprintf(d.fmt, SizeB1000(math.Round(speed)))
	default:
		d.msg = fmt.Sprintf(d.fmt, speed)
	}

	return d.FormatMsg(d.msg)
}

func (d *movingAverageSpeed) NextAmount(n int64, wdd ...time.Duration) {
	var workDuration time.Duration
	for _, wd := range wdd {
		workDuration = wd
	}
	speed := float64(n) / workDuration.Seconds() / 1000
	if math.IsInf(speed, 0) || math.IsNaN(speed) {
		return
	}
	d.average.Add(speed)
}

func (d *movingAverageSpeed) OnCompleteMessage(msg string) {
	d.completeMsg = &msg
}

// AverageSpeed decorator with dynamic unit measure adjustment. It's
// a wrapper of NewAverageSpeed.
func AverageSpeed(unit int, fmt string, wcc ...WC) Decorator {
	return NewAverageSpeed(unit, fmt, time.Now(), wcc...)
}

// NewAverageSpeed decorator with dynamic unit measure adjustment and
// user provided start time.
//
//	`unit` one of [0|UnitKiB|UnitKB] zero for no unit
//
//	`fmt` printf compatible verb for value, like "%f" or "%d"
//
//	`startTime` start time
//
//	`wcc` optional WC config
//
// fmt examples:
//
//	unit=UnitKiB, fmt="%.1f"  output: "1.0MiB/s"
//	unit=UnitKiB, fmt="% .1f" output: "1.0 MiB/s"
//	unit=UnitKB,  fmt="%.1f"  output: "1.0MB/s"
//	unit=UnitKB,  fmt="% .1f" output: "1.0 MB/s"
//
func NewAverageSpeed(unit int, fmt string, startTime time.Time, wcc ...WC) Decorator {
	var wc WC
	for _, widthConf := range wcc {
		wc = widthConf
	}
	wc.Init()
	if fmt == "" {
		fmt = "%.0f"
	}
	d := &averageSpeed{
		WC:        wc,
		unit:      unit,
		startTime: startTime,
		fmt:       fmt,
	}
	return d
}

type averageSpeed struct {
	WC
	unit        int
	startTime   time.Time
	fmt         string
	msg         string
	completeMsg *string
}

func (d *averageSpeed) Decor(st *Statistics) string {
	if st.Completed {
		if d.completeMsg != nil {
			return d.FormatMsg(*d.completeMsg)
		}
		return d.FormatMsg(d.msg)
	}

	timeElapsed := time.Since(d.startTime)
	speed := float64(st.Current) / timeElapsed.Seconds()

	switch d.unit {
	case UnitKiB:
		d.msg = fmt.Sprintf(d.fmt, sizePerSecond(SizeB1024(math.Round(speed))))
	case UnitKB:
		d.msg = fmt.Sprintf(d.fmt, sizePerSecond(SizeB1000(math.Round(speed))))
	default:
		d.msg = fmt.Sprintf(d.fmt, speed)
	}

	return d.FormatMsg(d.msg)
}

func (d *averageSpeed) OnCompleteMessage(msg string) {
	d.completeMsg = &msg
}

func (d *averageSpeed) AverageAdjust(startTime time.Time) {
	d.startTime = startTime
}
