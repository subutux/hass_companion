package ui

import "fyne.io/fyne/v2"

type View interface {
	Container() fyne.CanvasObject
}
