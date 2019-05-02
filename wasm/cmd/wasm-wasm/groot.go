package main

import (
	"github.com/maxence-charriere/app"
	"github.com/spf13/cobra"
	"github.com/will-rowe/baby-groot/cmd"
)

// Groot is a component that describes what parameters to submit for a GROOT run -  it implements the app.Compo interface
type Groot struct {
	Name       string
	SketchSize string
	FlagB      string
	FlagC      string
	Filename   string
}

// Render returns what to display
func (Groot *Groot) Render() string {
	return `
<div class="Groot">
	<button class="Menu" onclick="OnMenuClick" oncontextmenu="OnMenuClick">☰</button>
	<app.contextmenu>

	<h1>
		I am, 
		{{if .Name}}
			{{.Name}}
		{{else}}
			groot
		{{end}}!
	</h1>
	<input value="{{.Name}}" placeholder="What is your name?" onchange="{{bind "Name"}}" autofocus>


	<input value="{{.SketchSize}}" placeholder="sketch size" onchange="{{bind "SketchSize"}}" autofocus>
	<input type="radio"  value="true"  onchange="{{bind "FlagB"}}"> FlagB
	<input type="radio"  value="true"  onchange="{{bind "FlagC"}}"> FlagC
	<input type="file" value="{{.Filename}}" onchange="{{bind "Filename"}}">

	

	
	<button class="Button" onclick="ShowSettings">show settings</button>
	<button class="Button" onclick="Run">run</button>


</div>

`
}

// OnMenuClick creates a context menu when the menu button is clicked.
func (Groot *Groot) OnMenuClick() {
	app.NewContextMenu(
		app.MenuItem{
			Label:   "Reload",
			Keys:    "cmdorctrl+r",
			OnClick: app.Reload},
		app.MenuItem{Separator: true},
		app.MenuItem{
			Label: "Go to repository",
			OnClick: func() {
				app.Navigate("https://github.com/will-rowe/baby-groot")
			}},
		app.MenuItem{
			Label: "Source code",
			OnClick: func() {
				app.Navigate("https://github.com/will-rowe/baby-groot/blob/master/wasm/cmd/wasm-wasm/groot.go")
			}},
	)
}

// ShowSettings is for debugging the form
func (Groot *Groot) ShowSettings() {
	app.Log("here are the settings:")
	app.Log(Groot)
}

// Run will launch GROOT
func (Groot *Groot) Run() {

	var RootCmd = &cobra.Command{}
	cmd.RootCmd.AddCommand(cmd.IamgrootCmd)

	app.Log(RootCmd.Execute())

}
