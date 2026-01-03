package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// WindowConfig configures a desktop window.
type WindowConfig struct {
	Title  string
	Width  float32
	Height float32
	Body   fyne.CanvasObject
}

// Run starts a basic Fyne window.
func Run(cfg WindowConfig) {
	a := app.New()
	w := a.NewWindow(cfg.Title)
	if cfg.Body == nil {
		cfg.Body = container.NewCenter(widget.NewLabel("bebo desktop"))
	}
	w.SetContent(cfg.Body)
	if cfg.Width > 0 && cfg.Height > 0 {
		w.Resize(fyne.NewSize(cfg.Width, cfg.Height))
	}
	w.ShowAndRun()
}
