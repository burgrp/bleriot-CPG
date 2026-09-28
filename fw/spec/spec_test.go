package spec

import (
	"testing"

	"github.com/burgrp/bleriot/lib/shared/firmware"
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
}
