//go:build tinygo

package main

import (
	"machine"
	"time"
	_ "unsafe"

	"github.com/burgrp/bleriot-CPG/fw/spec"
	"github.com/burgrp/bleriot/lib/node"
	"github.com/burgrp/bleriot/lib/node/pan211x"
	"github.com/burgrp/tinygo-drivers/ws2812"
)

const (
	pinPump     = machine.PB4
	pinValveCCW = machine.PB5
	pinValveCW  = machine.PB6
	pinSmartLED = machine.PB7

	pinSpiCsn  = machine.PF3
	pinSpiSck  = machine.PA2
	pinSpiData = machine.PA3
)

type Device struct {
	bleNode            *node.Node
	led                ws2812.Device
	control            controlState
	ledRGB             int32
	ledBytes           [3]byte
	valveDeadTimeNanos int64
}

func bleriotMain(provisioning node.Provisioning, config spec.Config) {
	config = normalizeConfig(config)
	device := newDevice(int64(config.ValveDeadTimeMilliseconds) * int64(time.Millisecond))

	bleNode, err := pan211x.StartNode(provisioning, pinSpiSck, pinSpiData, pinSpiCsn, device)
	if err != nil {
		halt("failed to start BleRiot node: " + err.Error())
	}
	device.bleNode = bleNode

	for {
		bleNode.Poll()
		device.service(monotonicNanoseconds())
	}
}

//go:linkname monotonicNanoseconds runtime.nanotime
func monotonicNanoseconds() int64

func newDevice(valveDeadTimeNanos int64) *Device {
	configureActiveLowOutput(pinPump)
	configureActiveLowOutput(pinValveCCW)
	configureActiveLowOutput(pinValveCW)

	pinSmartLED.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	pinSmartLED.Low()
	pinSmartLED.Configure(machine.PinConfig{Mode: machine.PinOutput})

	device := &Device{
		led:                ws2812.NewWS2812(pinSmartLED),
		valveDeadTimeNanos: valveDeadTimeNanos,
	}
	if err := device.applyLED(device.control.outputs().ledRGB); err != nil {
		halt("failed to start smart LED: " + err.Error())
	}
	return device
}

func configureActiveLowOutput(pin machine.Pin) {
	pin.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	pin.High()
	pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
}

func (device *Device) Read(tag uint16) (value int32, null bool) {
	return device.control.read(tag)
}

func (device *Device) Write(tag uint16, value int32, null bool) {
	if null {
		return
	}

	previous := device.control.outputs()
	device.control.command(tag, value, monotonicNanoseconds(), device.valveDeadTimeNanos)
	device.applyControl(previous)
}

func (device *Device) service(now int64) {
	previous := device.control.outputs()
	device.control.service(now)
	device.applyControl(previous)
}

func (device *Device) applyControl(previous controlOutputs) {
	current := device.control.outputs()
	if current == previous {
		return
	}

	if previous.vcw != 0 && current.vcw == 0 {
		pinValveCW.High()
	}
	if previous.vccw != 0 && current.vccw == 0 {
		pinValveCCW.High()
	}
	if previous.vcw == 0 && current.vcw != 0 {
		pinValveCCW.High()
		pinValveCW.Low()
	}
	if previous.vccw == 0 && current.vccw != 0 {
		pinValveCW.High()
		pinValveCCW.Low()
	}
	if previous.pump != current.pump {
		pinPump.Set(current.pump == 0)
	}
	if previous.ledRGB != current.ledRGB {
		if err := device.applyLED(current.ledRGB); err != nil {
			halt("failed to write smart LED: " + err.Error())
		}
	}

	if previous.vcw != current.vcw {
		device.bleNode.Notify(spec.RegValveCW, current.vcw, false)
	}
	if previous.vccw != current.vccw {
		device.bleNode.Notify(spec.RegValveCCW, current.vccw, false)
	}
	if previous.pump != current.pump {
		device.bleNode.Notify(spec.RegPump, current.pump, false)
	}
}

func (device *Device) applyLED(value int32) error {
	device.ledBytes = rgbBytes(value)
	if _, err := device.led.Write(device.ledBytes[:]); err != nil {
		return err
	}
	device.ledRGB = value
	return nil
}

func halt(message string) {
	pinPump.High()
	pinValveCCW.High()
	pinValveCW.High()
	pinSmartLED.Low()
	println(message)
	for {
		time.Sleep(time.Second)
	}
}
