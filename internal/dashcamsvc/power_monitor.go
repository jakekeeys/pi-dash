package dashcamsvc

import (
	"github.com/stianeikeland/go-rpio"
	"time"
	"github.com/sirupsen/logrus"
	"os/exec"
)

type state string
const (
	usb state = "USB"
	lipo state = "LIPO"
	unknown state = "UNKNOWN"
)

type PowerMonitor struct {
	stop chan struct{}
	pin rpio.Pin
	state state
	dcsvc *DashCamService
}

func NewPowerMonitor(pinNo uint8, dcsvc *DashCamService) *PowerMonitor {
	pin := rpio.Pin(pinNo)
	pin.Input()

	return &PowerMonitor{
		stop: make(chan struct{}),
		pin: pin,
		state: unknown,
		dcsvc: dcsvc,
	}
}

func (m *PowerMonitor) Quit() {
	logrus.Info("PowerMonitor stopping")
	close(m.stop)
}

func (m *PowerMonitor) Run() {
	for {
		select {
		case <- m.stop:
				break
		default:
			newState := m.getState()
			if newState != m.state {
				switch newState {
				case usb:
					logrus.Info("PowerMonitor detected usb power")
					m.dcsvc.StartRecording()
					m.cancelShutdown()
				case lipo:
					logrus.Info("PowerMonitor detected battery power")
					m.dcsvc.StopRecording()
					m.scheduleShutdown()
				}
				m.state = newState
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (m *PowerMonitor) getState() state {
	res := m.pin.Read()
	switch res {
	case 0:
		return lipo
	case 1:
		return usb
	default:
		return unknown
	}
}

func (m *PowerMonitor) scheduleShutdown() {
	logrus.Debugf("Scheduling a shutdown in 15 minutes")
	err := exec.Command("sudo", "shutdown", "-P", "+15").Run()
	if err != nil {
		logrus.WithError(err).Error("error scheduling shutdown")
	}
}

func (m *PowerMonitor) cancelShutdown() {
	logrus.Debugf("Cancelling any scheduled shutdowns")
	err := exec.Command("sudo", "shutdown", "-c",).Run()
	if err != nil {
		logrus.WithError(err).Error("Error scheduling shutdown")
	}
}