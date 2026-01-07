package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/config"
	"github.com/devmarvs/bebo/middleware"
)

type homeData struct {
	Title string
	Items []string
}

func NewApp() *bebo.App {
	cfg := config.Default()
	cfg.TemplatesDir = templatesDir()
	cfg.LayoutTemplate = "layout.html"

	app := bebo.New(bebo.WithConfig(cfg))
	app.Use(middleware.RequestID(), middleware.Recover(), middleware.Logger())

	app.GET("/", func(ctx *bebo.Context) error {
		data := homeData{
			Title: "bebo web",
			Items: []string{"fast", "stdlib", "templates"},
		}
		return ctx.HTML(http.StatusOK, "home.html", data)
	})

	return app
}

func templatesDir() string {
	if _, err := os.Stat("templates"); err == nil {
		return "templates"
	}
	return filepath.Join("examples", "web", "templates")
}
