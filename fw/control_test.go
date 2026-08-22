package main

import (
	"testing"

	"github.com/burgrp/bleriot-CPG/fw/spec"
)

const testDeadTime = int64(100)

func TestStatusLEDPolicy(t *testing.T) {
	var state controlState
	if got := state.outputs().ledRGB; got != statusLEDDefaultRGB {
		t.Fatalf("default LED = %#06x, want %#06x", got, statusLEDDefaultRGB)
	}

	state.command(spec.RegPump, 1, 0, testDeadTime)
	if got := state.outputs().ledRGB; got != statusLEDPumpRGB {
		t.Fatalf("pump LED = %#06x, want %#06x", got, statusLEDPumpRGB)
	}

	state.command(spec.RegValveCW, 1, 0, testDeadTime)
	if got := state.outputs().ledRGB; got != statusLEDValveCWRGB {
		t.Fatalf("CW LED = %#06x, want %#06x", got, statusLEDValveCWRGB)
	}

	state.command(spec.RegValveCCW, 1, 10, testDeadTime)
	if got := state.outputs().ledRGB; got != statusLEDPumpRGB {
		t.Fatalf("dead-time LED = %#06x, want pump color %#06x", got, statusLEDPumpRGB)
	}
	state.service(110)
	if got := state.outputs().ledRGB; got != statusLEDValveCCWRGB {
		t.Fatalf("CCW LED = %#06x, want %#06x", got, statusLEDValveCCWRGB)
	}
}

func TestValveReversalUsesDeadTime(t *testing.T) {
	var state controlState
	state.command(spec.RegValveCW, 1, 0, testDeadTime)
	state.command(spec.RegValveCCW, 1, 10, testDeadTime)

	if outputs := state.outputs(); outputs.vcw != 0 || outputs.vccw != 0 {
		t.Fatalf("during dead time outputs = CW %d, CCW %d; want both off", outputs.vcw, outputs.vccw)
	}
	state.service(109)
	if outputs := state.outputs(); outputs.vcw != 0 || outputs.vccw != 0 {
		t.Fatalf("before deadline outputs = CW %d, CCW %d; want both off", outputs.vcw, outputs.vccw)
	}
	state.service(110)
	if outputs := state.outputs(); outputs.vcw != 0 || outputs.vccw != 1 {
		t.Fatalf("after deadline outputs = CW %d, CCW %d; want CCW on", outputs.vcw, outputs.vccw)
	}
}

func TestLatestValveCommandWinsDuringDeadTime(t *testing.T) {
	var state controlState
	state.command(spec.RegValveCW, 1, 0, testDeadTime)
	state.command(spec.RegValveCCW, 1, 10, testDeadTime)
	state.command(spec.RegValveCW, 1, 50, testDeadTime)
	state.service(110)

	if outputs := state.outputs(); outputs.vcw != 1 || outputs.vccw != 0 {
		t.Fatalf("outputs = CW %d, CCW %d; want latest CW command", outputs.vcw, outputs.vccw)
	}
}

func TestPendingValveCanBeCancelled(t *testing.T) {
	var state controlState
	state.command(spec.RegValveCW, 1, 0, testDeadTime)
	state.command(spec.RegValveCCW, 1, 10, testDeadTime)
	state.command(spec.RegValveCCW, 0, 50, testDeadTime)
	state.service(110)

	if outputs := state.outputs(); outputs.vcw != 0 || outputs.vccw != 0 {
		t.Fatalf("outputs = CW %d, CCW %d; want both off", outputs.vcw, outputs.vccw)
	}
}

func TestExplicitOffPreservesDeadTimeBeforeReverse(t *testing.T) {
	var state controlState
	state.command(spec.RegValveCW, 1, 0, testDeadTime)
	state.command(spec.RegValveCW, 0, 10, testDeadTime)
	state.command(spec.RegValveCCW, 1, 20, testDeadTime)

	state.service(109)
	if outputs := state.outputs(); outputs.vcw != 0 || outputs.vccw != 0 {
		t.Fatalf("before deadline outputs = CW %d, CCW %d; want both off", outputs.vcw, outputs.vccw)
	}
	state.service(110)
	if outputs := state.outputs(); outputs.vcw != 0 || outputs.vccw != 1 {
		t.Fatalf("after deadline outputs = CW %d, CCW %d; want CCW on", outputs.vcw, outputs.vccw)
	}
}

func TestValveOutputsAreMutuallyExclusive(t *testing.T) {
	var state controlState
	commands := []struct {
		tag   uint16
		value int32
		now   int64
	}{
		{spec.RegValveCW, 1, 0},
		{spec.RegValveCCW, 1, 10},
		{spec.RegValveCW, 1, 20},
		{spec.RegValveCCW, 0, 30},
		{spec.RegValveCCW, 1, 40},
		{spec.RegValveCW, 0, 50},
	}

	for _, command := range commands {
		state.command(command.tag, command.value, command.now, testDeadTime)
		outputs := state.outputs()
		if outputs.vcw != 0 && outputs.vccw != 0 {
			t.Fatalf("after tag %d=%d, both valve outputs are active", command.tag, command.value)
		}
	}
	state.service(110)
	outputs := state.outputs()
	if outputs.vcw != 0 && outputs.vccw != 0 {
		t.Fatal("both valve outputs are active after servicing deadline")
	}
}

func TestControlReadsOnlyActualRegisters(t *testing.T) {
	var state controlState
	state.command(spec.RegValveCW, 1, 0, testDeadTime)
	state.command(spec.RegValveCCW, 1, 10, testDeadTime)

	for _, tag := range []uint16{spec.RegValveCW, spec.RegValveCCW} {
		if value, null := state.read(tag); null || value != 0 {
			t.Errorf("read tag %d during dead time = (%d, %v), want (0, false)", tag, value, null)
		}
	}
	if value, null := state.read(4); !null || value != 0 {
		t.Errorf("read retired LED tag = (%d, %v), want (0, true)", value, null)
	}
}

func TestNormalizeConfigDefaultsDeadTime(t *testing.T) {
	config := normalizeConfig(spec.Config{})
	if config.ValveDeadTimeMilliseconds != defaultValveDeadTimeMilliseconds {
		t.Fatalf("dead time = %d ms, want %d ms", config.ValveDeadTimeMilliseconds, defaultValveDeadTimeMilliseconds)
	}
}
