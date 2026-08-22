//go:build !tinygo

package main

import (
	"github.com/burgrp/bleriot-CPG/fw/spec"
	"github.com/burgrp/bleriot/lib/shared/config"
	"github.com/burgrp/bleriot/lib/shared/inventory"
	"github.com/burgrp/bleriot/lib/site/cli"
)

var far = inventory.Channel{Name: "far", Number: 37, SpreadFactor: config.SpreadFactorS8}

func main() {
	cli.Start(inventory.Inventory{
		{
			Name:    "cpg",
			Address: [4]byte{0x1c, 0x95, 0xdb, 0x60},
			Key:     [16]byte{0xf7, 0xff, 0x37, 0x50, 0x4c, 0x88, 0xb3, 0xee, 0x79, 0x8f, 0x72, 0xfe, 0xa7, 0xb3, 0x5f, 0xc6},
			Channel: far,
			Type:    spec.Type(),
			Config: spec.Config{
				ValveDeadTimeMilliseconds: defaultValveDeadTimeMilliseconds,
			},
		},
	})
}
