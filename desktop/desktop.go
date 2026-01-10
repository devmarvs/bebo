package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// WindowConfig configures a desktop window.
type WindowConfig struct {
	Title    string
	Width    float32
	Height   float32
	Body     fyne.CanvasObject
	Icon     fyne.Resource
	Menu     *fyne.MainMenu
	TrayMenu *fyne.Menu
	OnClose  func()
}

// Run starts a basic Fyne window.
func Run(cfg WindowConfig) {
	a := app.New()
	w := a.NewWindow(cfg.Title)
	if cfg.Icon != nil {
		a.SetIcon(cfg.Icon)
		w.SetIcon(cfg.Icon)
	}
	if cfg.Body == nil {
		cfg.Body = container.NewCenter(widget.NewLabel("bebo desktop"))
	}
	w.SetContent(cfg.Body)
	if cfg.Menu != nil {
		w.SetMainMenu(cfg.Menu)
	}
	if cfg.TrayMenu != nil {
		if desktopApp, ok := a.(desktop.App); ok {
			desktopApp.SetSystemTrayMenu(cfg.TrayMenu)
		}
	}
	if cfg.OnClose != nil {
		w.SetCloseIntercept(func() {
			cfg.OnClose()
			w.Close()
		})
	}
	if cfg.Width > 0 && cfg.Height > 0 {
		w.Resize(fyne.NewSize(cfg.Width, cfg.Height))
	}
	w.ShowAndRun()
}

// LoadIcon loads an app icon from disk.
func LoadIcon(path string) (fyne.Resource, error) {
	return fyne.LoadResourceFromPath(path)
}
