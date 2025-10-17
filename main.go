package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/Hefero/D2R-AutoPotion-Go/cmd/config"
	"github.com/Hefero/D2R-AutoPotion-Go/cmd/lifewatcher"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/memory"
	"github.com/faiface/beep"
)

func main() {

	a := app.New()
	w := a.NewWindow("Hefero Diablo 2 Ressurrected AutoPotion")

	var manager = lifewatcher.Manager{}
	manager.Timer = time.Now()

	var XP = lifewatcher.ExperienceCalc{}

	errL := config.Load()
	if errL != nil {
		log.Fatalf("Error loading configuration file config.yaml: %s", errL.Error())
	}

	audioBufferL, err := lifewatcher.InitAudio("cmd/lifewatcher/assets/life.wav")
	audioBufferM, err := lifewatcher.InitAudio("cmd/lifewatcher/assets/mana.wav")
	audioBufferR, err := lifewatcher.InitAudio("cmd/lifewatcher/assets/rejuv.wav")

	ctx := contextWithSigterm(context.Background())

	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	var cmd *exec.Cmd
	//cmd = exec.Command(path + "\\gui.exe") // zakomentowane - bez GUI

	hello := widget.NewLabel("Diablo 2 Ressurrected AutoPotion")
	
	// ➕ Dodaj nowe przyciski testowe
	w.SetContent(container.NewVBox(
		hello,
		widget.NewButton("Start", func() {
			process, err := memory.NewProcess()

			if err != nil {
				fmt.Printf("error starting process: player needs to be inside a running game\n")
				fmt.Print("\033[A")
			}
			if err == nil {

				gr := memory.NewGameReader(process)

				watcher := lifewatcher.NewWatcher(gr)

				// GUI zakomentowane
				// if cmd.Process == nil {
				// 	cmd = exec.Command(path + "\\gui.exe")
				// 	cmd.Start()
				// }
				// if cmd.Process != nil {
				// 	cmd.Process.Kill()
				// 	cmd = exec.Command(path + "\\gui.exe")
				// 	cmd.Start()
				// }
				
				go StartWatcher(*watcher, ctx, &manager, &XP, audioBufferL, audioBufferM, audioBufferR, path)
			}
		}),
		widget.NewButton("Reset", func() {
			lifewatcher.ResetXPCalc(&XP)
		}),
		// ➕ Nowe przyciski debugowania
		widget.NewSeparator(),
		widget.NewLabel("--- Debug Tools ---"),
		widget.NewButton("Debug Coordinates", func() {
			process, err := memory.NewProcess()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			gr := memory.NewGameReader(process)
			debugger := lifewatcher.NewCoordinateDebugger(gr)
			debugger.DebugInventoryCoordinates()
		}),
		widget.NewButton("Test Mouse Movement", func() {
			process, err := memory.NewProcess()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			gr := memory.NewGameReader(process)
			debugger := lifewatcher.NewCoordinateDebugger(gr)
			go debugger.TestMouseMovement()
		}),
		widget.NewButton("Test Click on Potion", func() {
			process, err := memory.NewProcess()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			gr := memory.NewGameReader(process)
			debugger := lifewatcher.NewCoordinateDebugger(gr)
			go debugger.TestClickOnPotion()
		}),
	))

	w.ShowAndRun()

	defer func() {
		fmt.Println("\ncleanup")
		// cmd.Process.Kill() // zakomentowane
	}()

}

func StartWatcher(watcher lifewatcher.Watcher, ctx context.Context, manager *lifewatcher.Manager, XP *lifewatcher.ExperienceCalc, audioBufferL *beep.Buffer, audioBufferM *beep.Buffer, audioBufferR *beep.Buffer, path string) {
	ticker := time.NewTicker(time.Nanosecond * 1)
	for range ticker.C {
		watcher.Start(ctx, manager, XP, audioBufferL, audioBufferM, audioBufferR)
	}
}

func updateLabel(label *widget.Label, text string) {
	label.SetText(text)
}

func contextWithSigterm(ctx context.Context) context.Context {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)

		select {
		case <-signalCh:
		case <-ctx.Done():
		}
	}()

	return ctxWithCancel
}
