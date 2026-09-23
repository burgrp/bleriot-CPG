package main

import "github.com/burgrp/bleriot-CPG/fw/spec"

const (
	defaultValveDeadTimeMilliseconds = 100
	statusLEDOnlineRGB               = 0x101010
	statusLEDPumpRGB                 = 0x104010
	statusLEDValveCWRGB              = 0x401010
	statusLEDValveCCWRGB             = 0x101040
	statusLEDOfflineRGB              = 0xff0000
)

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

func linkStatusLEDColor(online, pulse bool, operatingRGB int32) int32 {
	if online {
		return operatingRGB
	}
	if pulse {
		return statusLEDOfflineRGB
	}
	return 0
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
