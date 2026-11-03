package spec

import (
	"testing"

	"github.com/burgrp/bleriot/lib/shared/firmware"
	"github.com/burgrp/bleriot/lib/shared/inventory"
)

func TestTypeValidates(t *testing.T) {
	if err := Type().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFirmwareProfile(t *testing.T) {
	profile := Type().Firmware
	if profile.Package != "github.com/burgrp/bleriot-CPG/fw" {
		t.Fatalf("firmware package = %q", profile.Package)
	}
	if profile.TinyGo.Scheduler != firmware.SchedulerNone || profile.TinyGo.StackSizeBytes != 0 {
		t.Fatalf("TinyGo profile = %+v", profile.TinyGo)
	}
	if profile.PyOCD.FrequencyHz != 100_000 || profile.PyOCD.LoadMode != firmware.ConnectUnderReset || profile.PyOCD.RTTMode != firmware.ConnectAttach || profile.PyOCD.GDBMode != firmware.ConnectAttach {
		t.Fatalf("pyOCD profile = %+v", profile.PyOCD)
	}
}

func TestRegisterContract(t *testing.T) {
	want := []struct {
		tag  uint16
		name string
	}{
		{RegValveCW, "vcw"},
		{RegValveCCW, "vccw"},
		{RegPump, "pump"},
		{RegWatchdog, "watchdog"},
	}

	registers := Type().Registers
	if len(registers) != len(want) {
		t.Fatalf("register count = %d, want %d", len(registers), len(want))
	}
	for index, expected := range want {
		if registers[index].Tag != expected.tag || registers[index].Name != expected.name {
			t.Errorf("register %d = (%d, %q), want (%d, %q)", index, registers[index].Tag, registers[index].Name, expected.tag, expected.name)
		}
	}

	watchdog := registers[len(registers)-1]
	instance := inventory.Instance{Name: "cpg", Type: Type()}
	if got := instance.RegistryName(watchdog); got != "cpg.watchdog" {
		t.Errorf("watchdog Registry name = %q, want %q", got, "cpg.watchdog")
	}
	if watchdog.Tag != 5 {
		t.Errorf("watchdog tag = %d, want 5 with retired tag 4 reserved", watchdog.Tag)
	}
	if watchdog.Type != inventory.TypeInt || watchdog.ReadOnly {
		t.Errorf("watchdog contract = type %q, read-only %v; want writable int", watchdog.Type, watchdog.ReadOnly)
	}
	if watchdog.Metadata["min"] != "0" || watchdog.Metadata["max"] != "60" || watchdog.Metadata["unit"] != "seconds" {
		t.Errorf("watchdog metadata = %v, want range 0..60 seconds", watchdog.Metadata)
	}
}
