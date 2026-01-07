package main

import "log"

func main() {
	app := NewApp()
	if err := app.RunWithSignals(); err != nil {
		log.Fatal(err)
	}
}
