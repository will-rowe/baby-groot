package main

import (
	"log"

	"github.com/maxence-charriere/app"
)

// The app entry point.
func main() {
	// Imports the Groot component declared in groot.go in order to make it loadable in a page or usable in other components
	app.Import(&Groot{})

	// Defines the component to load when an URL without path is loaded
	app.DefaultPath = "/groot"

	// Runs the app in the browser
	if err := app.Run(); err != nil {
		log.Print(err)
	}
}
