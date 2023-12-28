package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/subutux/hass_companion/hass/mobile_app/sensors"
)

type SensorView struct {
	Title  string
	sensor sensors.SensorInterface
}

func NewSensorView(sensor sensors.SensorInterface) SensorView {
	return SensorView{
		Title:  "Sensor",
		sensor: sensor,
	}
}

func (s SensorView) Container() fyne.CanvasObject {
	mainTitle := widget.NewRichTextFromMarkdown("# Sensor")
	sensor := s.sensor.GetSensors()[0]
	title := widget.NewLabel(sensor.Name)
	title.TextStyle.Bold = true

	tTitle := widget.NewLabel("Type")
	tTitle.TextStyle.Bold = true
	tValue := widget.NewLabel(sensor.Type)
	Type := container.NewHBox(tTitle, tValue)

	dcTitle := widget.NewLabel("Device class")
	dcTitle.TextStyle.Bold = true
	dcValue := widget.NewLabel(sensor.DeviceClass)
	DeviceClass := container.NewHBox(dcTitle, dcValue)

	sTitle := widget.NewLabel("State")
	sTitle.TextStyle.Bold = true
	sValue := widget.NewLabel(fmt.Sprintf("%v%s", sensor.State, sensor.UnitOfMeasurement))
	state := container.NewHBox(sTitle, sValue)

	dTitle := widget.NewLabel("Disabled")
	dTitle.TextStyle.Bold = true
	dValue := widget.NewLabel(fmt.Sprintf("%v", sensor.Disabled))
	disabled := container.NewHBox(dTitle, dValue)

	return container.NewVBox(mainTitle, title, Type, DeviceClass, state, disabled)
}
