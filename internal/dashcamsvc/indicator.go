package dashcamsvc

import (
	"github.com/stianeikeland/go-rpio"
	"github.com/sirupsen/logrus"
)

type Indicator struct {
	pin rpio.Pin
}

func NewIndicator(pinNo uint8) *Indicator {
	pin := rpio.Pin(pinNo)
	pin.Output()

	return &Indicator{
		pin: pin,
	}
}

func (i *Indicator) Illuminate() {
	logrus.Info("Indicator illuminating")
	i.pin.High()
}

func (i *Indicator) Extinguish() {
	logrus.Info("Indicator extinguishing")
	i.pin.Low()
}