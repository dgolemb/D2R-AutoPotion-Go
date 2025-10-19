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
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
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
	cmd = exec.Command(path + "\\gui.exe")

	hello := widget.NewLabel("Diablo 2 Ressurrected AutoPotion")
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

				if cmd.Process == nil {
					cmd = exec.Command(path + "\\gui.exe")
					cmd.Start()
				}
				if cmd.Process != nil {
					cmd.Process.Kill()
					cmd = exec.Command(path + "\\gui.exe")
					cmd.Start()
				}
				go StartWatcher(*watcher, ctx, &manager, &XP, audioBufferL, audioBufferM, audioBufferR, path)
			}
		}),
		widget.NewButton("Reset", func() {
			lifewatcher.ResetXPCalc(&XP)
		}),
		widget.NewButton("Debug Screen Resolution", func() {
			process, err := memory.NewProcess()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			gr := memory.NewGameReader(process)
			interaction := memory.NewItemInteraction(gr)
			
			// Wyświetl informacje o rozdzielczości i obliczone współrzędne
			interaction.DebugUICoordinates()
		}),
		widget.NewButton("Debug Potion IDs", func() {
			process, err := memory.NewProcess()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			gr := memory.NewGameReader(process)
			
			d, err := gr.GetData()
			if err != nil {
				fmt.Printf("Error getting data: %v\n", err)
				return
			}
			
			fmt.Println("\n╔══════════════════════════════════════╗")
			fmt.Println("║    POTION TXT FILE NO DEBUGGER      ║")
			fmt.Println("╚══════════════════════════════════════╝")
			
			hpCount, manaCount, rejuvCount := 0, 0, 0
			
			fmt.Println("\n📦 INVENTORY POTIONS:")
			for _, itm := range d.Items.AllItems {
				if itm.Location == item.LocationInventory && itm.IsPotion() {
					potionType := "UNKNOWN"
					if itm.IsHealingPotion() {
						potionType = "HP"
						hpCount++
					} else if itm.IsManaPotion() {
						potionType = "MANA"
						manaCount++
					} else if itm.IsRejuvPotion() {
						potionType = "REJUV"
						rejuvCount++
					}
					
					fmt.Printf("  [%s] %s | TxtFileNo: %d | Grid(%d,%d)\n",
						potionType, itm.Name, itm.TxtFileNo, itm.Position.X, itm.Position.Y)
				}
			}
			
			fmt.Println("\n🎒 BELT POTIONS:")
			for _, itm := range d.Items.Belt.Items {
				potionType := "OTHER"
				if itm.IsHealingPotion() {
					potionType = "HP"
				} else if itm.IsManaPotion() {
					potionType = "MANA"
				} else if itm.IsRejuvPotion() {
					potionType = "REJUV"
				}
				
				fmt.Printf("  [%s] %s | TxtFileNo: %d | Slot(%d,%d)\n",
					potionType, itm.Name, itm.TxtFileNo, itm.Position.X, itm.Position.Y)
			}
			
			fmt.Printf("\n📊 SUMMARY:\n")
			fmt.Printf("  HP Potions in inventory: %d\n", hpCount)
			fmt.Printf("  Mana Potions in inventory: %d\n", manaCount)
			fmt.Printf("  Rejuv Potions in inventory: %d\n", rejuvCount)
			fmt.Println("\n✅ Expected TxtFileNo ranges:")
			fmt.Println("  HP:    586-590 (Minor→Super)")
			fmt.Println("  Mana:  591-595 (Minor→Super)")
			fmt.Println("  Rejuv: 515-516 (Normal→Full)")
			fmt.Println("═══════════════════════════════════════\n")
		}),
	))

	w.ShowAndRun()

	defer func() {
		fmt.Println("\ncleanup")
		cmd.Process.Kill()
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
