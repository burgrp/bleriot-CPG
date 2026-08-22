package spec

import (
	"github.com/burgrp/bleriot/lib/shared/inventory"
	"github.com/burgrp/bleriot/lib/shared/puya"
)

type Config struct {
	ValveDeadTimeMilliseconds uint32
}

const (
	RegValveCW  = 1
	RegValveCCW = 2
	RegPump     = 3
)

var Chip = puya.PY32F030x8

func Type() inventory.DeviceType {
	return inventory.DeviceType{
		Name: "cpg",
		Chip: Chip,
		Registers: []inventory.Register{
			{Tag: RegValveCW, Name: "vcw", Type: inventory.TypeBool},
			{Tag: RegValveCCW, Name: "vccw", Type: inventory.TypeBool},
			{Tag: RegPump, Name: "pump", Type: inventory.TypeBool},
		},
	}
}
