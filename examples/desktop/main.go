package main

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/devmarvs/bebo/desktop"
)

func main() {
	content := container.NewCenter(widget.NewLabel("Hello from bebo desktop"))
	desktop.Run(desktop.WindowConfig{
		Title:  "bebo",
		Width:  520,
		Height: 320,
		Body:   content,
	})
}
