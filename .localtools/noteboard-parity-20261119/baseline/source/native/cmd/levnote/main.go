package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/app"
)

func main() {
	config, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	s := app.New(filepath.Join(config, "LevNoteNative"))
	mygo.App.SetName("轻笺")
	mygo.App.WhenReady(func() {
		s.Window = mygo.NewWindow(mygo.WindowOptions{Title: "轻笺 · 原生版", Width: 1180, Height: 780, MinWidth: 760, MinHeight: 480, Content: ui.View(s.View)})
		s.Restore()
		s.Window.OnClose(func(e *mygo.CloseEvent) {
			if !s.CanClose() {
				e.PreventDefault()
			}
		})
		s.Window.OnClosed(s.Shutdown)
		s.Window.OnFocus(s.CheckExternal)
		s.Window.OnFileDrop(func(e *mygo.FileDropEvent) {
			s.OpenMany(e.Paths)
		})
		if len(os.Args) > 1 {
			s.OpenMany(os.Args[1:])
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
