package spec

import (
	"github.com/burgrp/bleriot/lib/shared/firmware"
	"github.com/burgrp/bleriot/lib/shared/inventory"
	"github.com/burgrp/bleriot/lib/shared/puya"
)

type Config struct {
	ValveDeadTimeMilliseconds uint32
	WatchdogSafePositionCCW   bool
}

const (
	RegValveCW  = 1
	RegValveCCW = 2
	RegPump     = 3
	RegWatchdog = 5

	WatchdogMaxSeconds int32 = 60
)

var Chip = puya.PY32F030x8

func Type() inventory.DeviceType {
	return inventory.DeviceType{
		Name: "cpg",
		Chip: Chip,
		Firmware: firmware.Manifest{
			Package: "github.com/burgrp/bleriot-CPG/fw",
			TinyGo: firmware.TinyGoProfile{
				Scheduler:        firmware.SchedulerNone,
				GarbageCollector: firmware.GCLeaking,
				Serial:           firmware.SerialRTT,
				SizeReport:       firmware.SizeReportHTML,
				PrintAllocs:      true,
			},
			PyOCD: firmware.PyOCDProfile{
				FrequencyHz: 100_000,
				LoadMode:    firmware.ConnectUnderReset,
				RTTMode:     firmware.ConnectAttach,
				GDBMode:     firmware.ConnectAttach,
			},
		},
		Registers: []inventory.Register{
			{Tag: RegValveCW, Name: "vcw", Type: inventory.TypeBool},
			{Tag: RegValveCCW, Name: "vccw", Type: inventory.TypeBool},
			{Tag: RegPump, Name: "pump", Type: inventory.TypeBool},
			{
				Tag:  RegWatchdog,
				Name: "watchdog",
				Type: inventory.TypeInt,
				Metadata: map[string]string{
					"min":  "0",
					"max":  "60",
					"unit": "seconds",
				},
			},
		},
	}
}
