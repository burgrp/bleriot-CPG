package cpg

import (
	"testing"

	"github.com/burgrp/bleriot-CPG/fw/spec"
)

const testDeadTime = int64(100)

func TestStatusLEDPalette(t *testing.T) {
	tests := []struct {
		name  string
		value int32
		want  int32
	}{
		{name: "online idle", value: statusLEDOnlineRGB, want: 0xA05000},
		{name: "offline", value: statusLEDOfflineRGB, want: 0xFF3000},
		{name: "watchdog", value: statusLEDWatchdogRGB, want: 0x0000FF},
		{name: "pump", value: statusLEDPumpRGB, want: 0x00FF10},
		{name: "valve CW", value: statusLEDValveCWRGB, want: 0xFF000A},
		{name: "valve CCW", value: statusLEDValveCCWRGB, want: 0x0020FF},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.value != test.want {
				t.Fatalf("LED color = %#06x, want %#06x", test.value, test.want)
			}
		})
	}
}

func TestLinkStateReportsOnlyOnlineToOfflineTransition(t *testing.T) {
	var state linkState
	for index, test := range []struct {
		online bool
		want   bool
	}{
		{online: false, want: false},
		{online: true, want: false},
		{online: true, want: false},
		{online: false, want: true},
		{online: false, want: false},
		{online: true, want: false},
		{online: false, want: true},
	} {
		if got := state.update(test.online); got != test.want {
			t.Errorf("update %d expiry = %v, want %v", index, got, test.want)
		}
	}
}

func TestWatchdogLeaseClampsAndRoundsRemainingUp(t *testing.T) {
	const now = int64(12_345)
	var lease watchdogLease
	if armed := lease.set(spec.WatchdogMaxSeconds+1, false, now); !armed {
		t.Fatal("positive watchdog write did not arm lease")
	}

	for _, test := range []struct {
		name string
		now  int64
		want int32
	}{
		{name: "at set", now: now, want: 60},
		{name: "fractional first second", now: now + 1, want: 60},
		{name: "one second elapsed", now: now + nanosecondsPerSecond, want: 59},
		{name: "fractional final second", now: now + 59*nanosecondsPerSecond + 1, want: 1},
		{name: "at deadline", now: now + 60*nanosecondsPerSecond, want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := lease.remaining(test.now); got != test.want {
				t.Errorf("remaining(%d) = %d, want %d", test.now, got, test.want)
			}
		})
	}
}

func TestWatchdogRegisterSafetyLifecycle(t *testing.T) {
	const now = int64(1_000)
	state := newControl(false, testDeadTime)

	state.writeAt(spec.RegPump, 1, false, now)
	if outputs := state.outputs(); outputs.pump != 0 {
		t.Fatalf("pump write while expired = %d, want 0", outputs.pump)
	}
	if got, null := state.readAt(spec.RegWatchdog, now); got != 0 || null {
		t.Fatalf("initial watchdog read = (%d, %v), want (0, false)", got, null)
	}

	state.writeAt(spec.RegWatchdog, 2, false, now)
	state.writeAt(spec.RegPump, 1, false, now)
	state.writeAt(spec.RegValveCCW, 1, false, now)
	if got, null := state.readAt(spec.RegWatchdog, now+nanosecondsPerSecond+1); got != 1 || null {
		t.Fatalf("counted-down watchdog read = (%d, %v), want (1, false)", got, null)
	}

	deadline := now + 2*nanosecondsPerSecond
	if got, null := state.readAt(spec.RegWatchdog, deadline); got != 0 || null {
		t.Fatalf("expired watchdog read = (%d, %v), want (0, false)", got, null)
	}
	if outputs := state.outputs(); outputs.pump != 0 || outputs.vcw != 0 || outputs.vccw != 0 {
		t.Fatalf("expiry reversal outputs = pump %d, CW %d, CCW %d; want all off", outputs.pump, outputs.vcw, outputs.vccw)
	}
	state.serviceAt(deadline + testDeadTime)
	if outputs := state.outputs(); outputs.vcw != 1 || outputs.vccw != 0 {
		t.Fatalf("safe outputs = CW %d, CCW %d; want CW safe end", outputs.vcw, outputs.vccw)
	}

	state.writeAt(spec.RegValveCCW, 1, false, deadline+testDeadTime+1)
	if outputs := state.outputs(); outputs.vcw != 1 || outputs.vccw != 0 {
		t.Fatalf("write while expired changed safe outputs to CW %d, CCW %d", outputs.vcw, outputs.vccw)
	}
	state.writeAt(spec.RegWatchdog, 5, false, deadline+testDeadTime+1)
	if outputs := state.outputs(); outputs.vcw != 1 || outputs.vccw != 0 {
		t.Fatalf("re-arm changed safe outputs to CW %d, CCW %d", outputs.vcw, outputs.vccw)
	}
	state.writeAt(spec.RegPump, 1, false, deadline+testDeadTime+1)
	if outputs := state.outputs(); outputs.pump != 1 {
		t.Fatalf("fresh pump write after re-arm = %d, want 1", outputs.pump)
	}
	state.writeAt(spec.RegWatchdog, 0, false, deadline+testDeadTime+2)
	if outputs := state.outputs(); outputs.pump != 0 || outputs.vcw != 1 || outputs.vccw != 0 {
		t.Fatalf("zero watchdog outputs = pump %d, CW %d, CCW %d; want 0/1/0", outputs.pump, outputs.vcw, outputs.vccw)
	}
}

func TestWatchdogNegativeAndNullWritesExpire(t *testing.T) {
	for _, test := range []struct {
		name  string
		value int32
		null  bool
	}{
		{name: "negative", value: -1},
		{name: "null", value: 30, null: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := newControl(true, testDeadTime)
			state.writeAt(spec.RegWatchdog, 30, false, 0)
			state.writeAt(spec.RegPump, 1, false, 0)
			state.writeAt(spec.RegWatchdog, test.value, test.null, 1)
			if outputs := state.outputs(); outputs.pump != 0 || outputs.vccw != 1 {
				t.Fatalf("expiry outputs = pump %d, CCW %d; want 0/1", outputs.pump, outputs.vccw)
			}
		})
	}
}

func TestWatchdogControlDoesNotAllocate(t *testing.T) {
	state := newControl(false, testDeadTime)
	state.writeAt(spec.RegWatchdog, spec.WatchdogMaxSeconds, false, 0)

	if allocs := testing.AllocsPerRun(1000, func() {
		state.writeAt(spec.RegPump, 1, false, 1)
		state.readAt(spec.RegWatchdog, 1)
	}); allocs != 0 {
		t.Fatalf("steady-state watchdog operations allocated %v times, want 0", allocs)
	}
}

func TestWatchdogStopsPumpAndMovesValveToConfiguredSafePosition(t *testing.T) {
	for _, test := range []struct {
		name            string
		safePositionCCW bool
		activeTag       uint16
		wantCW          int32
		wantCCW         int32
	}{
		{name: "CW", activeTag: spec.RegValveCCW, wantCW: 1},
		{name: "CCW", safePositionCCW: true, activeTag: spec.RegValveCW, wantCCW: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var state controlState
			state.command(spec.RegPump, 1, 0, testDeadTime)
			state.command(test.activeTag, 1, 0, testDeadTime)

			state.applyWatchdog(test.safePositionCCW, 10, testDeadTime)
			if outputs := state.outputs(); outputs.pump != 0 || outputs.vcw != 0 || outputs.vccw != 0 {
				t.Fatalf("during watchdog reversal outputs = pump %d, CW %d, CCW %d; want all off", outputs.pump, outputs.vcw, outputs.vccw)
			}
			state.service(109)
			if outputs := state.outputs(); outputs.vcw != 0 || outputs.vccw != 0 {
				t.Fatalf("before deadline outputs = CW %d, CCW %d; want both off", outputs.vcw, outputs.vccw)
			}
			state.service(110)
			if outputs := state.outputs(); outputs.pump != 0 || outputs.vcw != test.wantCW || outputs.vccw != test.wantCCW {
				t.Fatalf("safe outputs = pump %d, CW %d, CCW %d; want 0/%d/%d", outputs.pump, outputs.vcw, outputs.vccw, test.wantCW, test.wantCCW)
			}
		})
	}
}

func TestWatchdogSafePositionReplacesPendingValveCommand(t *testing.T) {
	var state controlState
	state.command(spec.RegValveCW, 1, 0, testDeadTime)
	state.command(spec.RegValveCCW, 1, 10, testDeadTime)

	state.applyWatchdog(false, 50, testDeadTime)
	state.service(110)
	if outputs := state.outputs(); outputs.vcw != 1 || outputs.vccw != 0 {
		t.Fatalf("safe outputs = CW %d, CCW %d; want watchdog CW command", outputs.vcw, outputs.vccw)
	}
}

func TestStatusLEDPolicy(t *testing.T) {
	var state controlState
	if got := state.outputs().ledRGB; got != statusLEDOnlineRGB {
		t.Fatalf("default LED = %#06x, want %#06x", got, statusLEDOnlineRGB)
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

func TestLinkStatusLEDPolicy(t *testing.T) {
	tests := []struct {
		name            string
		online          bool
		pulse           bool
		watchdogExpired bool
		operating       int32
		want            int32
	}{
		{name: "online", online: true, operating: statusLEDValveCCWRGB, want: statusLEDValveCCWRGB},
		{name: "offline pulse", pulse: true, operating: statusLEDPumpRGB, want: statusLEDOfflineRGB},
		{name: "offline dark", operating: statusLEDPumpRGB, want: 0},
		{name: "offline overrides watchdog", pulse: true, watchdogExpired: true, operating: statusLEDPumpRGB, want: statusLEDOfflineRGB},
		{name: "watchdog pulse", online: true, pulse: true, watchdogExpired: true, operating: statusLEDPumpRGB, want: statusLEDWatchdogRGB},
		{name: "watchdog dark", online: true, watchdogExpired: true, operating: statusLEDPumpRGB, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := statusLEDColor(test.online, test.pulse, test.watchdogExpired, test.operating); got != test.want {
				t.Fatalf("LED = %#06x, want %#06x", got, test.want)
			}
		})
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
