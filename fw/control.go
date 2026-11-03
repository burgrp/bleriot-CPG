package cpg

import "github.com/burgrp/bleriot-CPG/fw/spec"

const (
	defaultValveDeadTimeMilliseconds = 100
	nanosecondsPerSecond             = int64(1_000_000_000)
	statusLEDOnlineRGB               = 0xA05000
	statusLEDOfflineRGB              = 0x0020FF
	statusLEDPumpRGB                 = 0x00FF10
	statusLEDValveCWRGB              = 0xFF000A
	statusLEDValveCCWRGB             = 0x0020FF
)

type linkState struct {
	online bool
}

func (state *linkState) update(online bool) (wentOffline bool) {
	wentOffline = state.online && !online
	state.online = online
	return wentOffline
}

type valveDirection uint8

const (
	valveOff valveDirection = iota
	valveCW
	valveCCW
)

type controlState struct {
	activeValve  valveDirection
	pendingValve valveDirection
	pump         int32
	deadUntil    int64
	watchdog     watchdogLease
	safeCCW      bool
	deadTime     int64
}

type controlOutputs struct {
	vcw    int32
	vccw   int32
	pump   int32
	ledRGB int32
}

func normalizeConfig(config spec.Config) spec.Config {
	if config.ValveDeadTimeMilliseconds == 0 {
		config.ValveDeadTimeMilliseconds = defaultValveDeadTimeMilliseconds
	}
	return config
}

func newControl(safeCCW bool, deadTime int64) controlState {
	return controlState{safeCCW: safeCCW, deadTime: deadTime}
}

func statusLEDColor(online, pulse, watchdogExpired bool, operatingRGB int32) int32 {
	if !online || watchdogExpired {
		if pulse {
			return statusLEDOfflineRGB
		}
		return 0
	}
	return operatingRGB
}

func (state *controlState) readAt(tag uint16, now int64) (value int32, null bool) {
	state.expireWatchdog(now)
	if tag == spec.RegWatchdog {
		return state.watchdog.remaining(now), false
	}
	return state.read(tag)
}

func (state *controlState) writeAt(tag uint16, value int32, null bool, now int64) {
	state.expireWatchdog(now)
	if tag == spec.RegWatchdog {
		if !state.watchdog.set(value, null, now) {
			state.forceSafe(now)
		}
		return
	}
	if null || !state.watchdog.active(now) {
		return
	}
	state.command(tag, value, now, state.deadTime)
}

func (state *controlState) serviceAt(now int64) {
	state.expireWatchdog(now)
	state.service(now)
}

func (state *controlState) forceSafe(now int64) {
	state.applyWatchdog(state.safeCCW, now, state.deadTime)
}

func (state *controlState) expireWatchdog(now int64) bool {
	if !state.watchdog.expire(now) {
		return false
	}
	state.forceSafe(now)
	return true
}

func (state *controlState) watchdogExpired(now int64) bool {
	return !state.watchdog.active(now)
}

type watchdogLease struct {
	deadline int64
}

func (lease *watchdogLease) set(value int32, null bool, now int64) bool {
	if null || value <= 0 {
		lease.deadline = 0
		return false
	}
	if value > spec.WatchdogMaxSeconds {
		value = spec.WatchdogMaxSeconds
	}
	lease.deadline = now + int64(value)*nanosecondsPerSecond
	return true
}

func (lease *watchdogLease) active(now int64) bool {
	return lease.deadline != 0 && lease.deadline > now
}

func (lease *watchdogLease) remaining(now int64) int32 {
	if !lease.active(now) {
		return 0
	}
	remaining := lease.deadline - now
	return int32((remaining + nanosecondsPerSecond - 1) / nanosecondsPerSecond)
}

func (lease *watchdogLease) expire(now int64) bool {
	if lease.deadline == 0 || lease.deadline > now {
		return false
	}
	lease.deadline = 0
	return true
}

func (state *controlState) read(tag uint16) (value int32, null bool) {
	outputs := state.outputs()
	switch tag {
	case spec.RegValveCW:
		return outputs.vcw, false
	case spec.RegValveCCW:
		return outputs.vccw, false
	case spec.RegPump:
		return outputs.pump, false
	default:
		return 0, true
	}
}

func (state *controlState) command(tag uint16, value int32, now, deadTime int64) {
	value = normalizeBool(value)
	switch tag {
	case spec.RegValveCW:
		state.commandValve(valveCW, value, now, deadTime)
	case spec.RegValveCCW:
		state.commandValve(valveCCW, value, now, deadTime)
	case spec.RegPump:
		state.pump = value
	}
}

func (state *controlState) applyWatchdog(safePositionCCW bool, now, deadTime int64) {
	state.command(spec.RegPump, 0, now, deadTime)
	if safePositionCCW {
		state.command(spec.RegValveCCW, 1, now, deadTime)
		return
	}
	state.command(spec.RegValveCW, 1, now, deadTime)
}

func (state *controlState) commandValve(direction valveDirection, value int32, now, deadTime int64) {
	if value == 0 {
		if state.pendingValve == direction {
			state.pendingValve = valveOff
		}
		if state.activeValve == direction {
			state.activeValve = valveOff
			state.deadUntil = now + deadTime
		}
		return
	}

	if state.activeValve == direction {
		return
	}
	if state.activeValve != valveOff {
		state.activeValve = valveOff
		state.deadUntil = now + deadTime
	}
	if now < state.deadUntil {
		state.pendingValve = direction
		return
	}

	state.activeValve = direction
	state.pendingValve = valveOff
}

func (state *controlState) service(now int64) {
	if state.activeValve != valveOff || state.pendingValve == valveOff || now < state.deadUntil {
		return
	}
	state.activeValve = state.pendingValve
	state.pendingValve = valveOff
}

func (state *controlState) outputs() controlOutputs {
	outputs := controlOutputs{pump: state.pump}
	switch state.activeValve {
	case valveCW:
		outputs.vcw = 1
		outputs.ledRGB = statusLEDValveCWRGB
	case valveCCW:
		outputs.vccw = 1
		outputs.ledRGB = statusLEDValveCCWRGB
	case valveOff:
		if state.pump != 0 {
			outputs.ledRGB = statusLEDPumpRGB
		} else {
			outputs.ledRGB = statusLEDOnlineRGB
		}
	}
	return outputs
}
