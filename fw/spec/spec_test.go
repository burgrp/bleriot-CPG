package spec

import "testing"

func TestTypeValidates(t *testing.T) {
	if err := Type().Validate(); err != nil {
		t.Fatal(err)
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
