package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/subutux/hass_companion/hass/mobile_app/sensors"
)

type MainContent struct {
	sensors   []sensors.SensorInterface
	views     map[string]View
	viewNames []string
	viewsList binding.ExternalStringList
	container *fyne.Container
	list      *widget.List
}

func NewMainContent(status_content View) *MainContent {
	m := MainContent{
		sensors:   []sensors.SensorInterface{},
		views:     make(map[string]View),
		viewNames: []string{},
		container: container.NewStack(),
	}

	m.viewsList = binding.BindStringList(&m.viewNames)
	m.viewNames = append(m.viewNames, "status")
	m.views["status"] = status_content
	m.viewsList.Reload()
	return &m
}

func (m *MainContent) AppendSensor(sensor sensors.SensorInterface) {
	s := sensor.GetSensors()[0]
	m.sensors = append(m.sensors, sensor)
	m.viewNames = append(m.viewNames, s.UniqueID)
	view := NewSensorView(sensor)
	m.views[s.UniqueID] = &view
	m.viewsList.Reload()
}

func (m *MainContent) SetContent(name string) {
	v, ok := m.views[name]

	if ok {
		m.container.Objects = []fyne.CanvasObject{v.Container()}
		m.container.Refresh()
	}

}
func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}
func (m *MainContent) Select(name string) {
	_, ok := m.views[name]
	idx := Index(m.viewNames, name)
	if ok && idx != -1 {
		m.list.Select(idx)
	}
}

func (m *MainContent) Container() fyne.CanvasObject {
	m.list = widget.NewListWithData(m.viewsList, func() fyne.CanvasObject {
		return widget.NewLabel("template")
	}, func(i binding.DataItem, o fyne.CanvasObject) {
		b := i.(binding.String)
		o.(*widget.Label).Bind(b)
		if str, _ := b.Get(); str == "status" {
			o.(*widget.Label).TextStyle.Bold = true
		}
	})

	m.list.OnSelected = func(id widget.ListItemID) {
		m.SetContent(m.viewNames[id])
	}
	m.list.OnUnselected = func(id widget.ListItemID) {
		m.SetContent("status")
	}
	m.list.Select(0)
	split := container.NewHSplit(container.NewScroll(m.list), m.container)
	split.SetOffset(0.30)
	return split
}
