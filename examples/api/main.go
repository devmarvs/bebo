package main

import (
	"log"
	"net/http"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/middleware"
)

type userRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	app := bebo.New()
	app.Use(middleware.RequestID(), middleware.Recover(), middleware.Logger())

	app.GET("/health", func(ctx *bebo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	app.GET("/users/:id", func(ctx *bebo.Context) error {
		id := ctx.Param("id")
		return ctx.JSON(http.StatusOK, map[string]string{"id": id})
	})

	app.POST("/users", func(ctx *bebo.Context) error {
		var req userRequest
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		if req.Name == "" {
			return apperr.New(apperr.CodeValidation, http.StatusBadRequest, "name is required", nil)
		}
		return ctx.JSON(http.StatusCreated, map[string]string{"status": "created"})
	})

	if err := app.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
