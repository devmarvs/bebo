package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/devmarvs/bebo/desktop"
)

func main() {
	content := container.NewCenter(widget.NewLabel("Hello from bebo desktop"))

	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Quit", func() {
			fyne.CurrentApp().Quit()
		}),
	)
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {}),
	)

	desktop.Run(desktop.WindowConfig{
		Title:    "bebo",
		Width:    520,
		Height:   320,
		Body:     content,
		Menu:     fyne.NewMainMenu(fileMenu, helpMenu),
		TrayMenu: fyne.NewMenu("bebo", fyne.NewMenuItem("Open", func() {})),
	})
}
