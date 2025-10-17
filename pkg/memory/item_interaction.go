package memory

import (
	"fmt"
	"time"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
	"github.com/micmonay/keybd_event"
	"golang.org/x/sys/windows"
)

type ItemInteraction struct {
	gr *GameReader
}

func NewItemInteraction(gr *GameReader) *ItemInteraction {
	return &ItemInteraction{gr: gr}
}

// GridToScreenCoordinates konwertuje pozycję w gridzie inventory na współrzędne ekranu
func (ii *ItemInteraction) GridToScreenCoordinates(pos data.Position) (int, int) {
	// Współrzędne początku inventory (należy dostosować do rozdzielczości)
	// Te wartości są dla rozdzielczości 1920x1080
	const inventoryStartX = 1687
	const inventoryStartY = 773
	const gridCellWidth = 39
	const gridCellHeight = 39

	screenX := inventoryStartX + (pos.X * gridCellWidth) + (gridCellWidth / 2)
	screenY := inventoryStartY + (pos.Y * gridCellHeight) + (gridCellHeight / 2)

	return screenX, screenY
}

// OpenInventory otwiera inventory używając klawisza 'i'
func (ii *ItemInteraction) OpenInventory() error {
	d, err := ii.gr.GetData()
	if err != nil {
		return err
	}

	if !d.OpenMenus.Inventory {
		kb, err := keybd_event.NewKeyBonding()
		if err != nil {
			return err
		}

		kb.SetKeys(keybd_event.VK_I)
		if err := kb.Launching(); err != nil {
			return err
		}

		// Czekaj na otwarcie inventory
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

// CloseInventory zamyka inventory używając klawisza ESC
func (ii *ItemInteraction) CloseInventory() error {
	d, err := ii.gr.GetData()
	if err != nil {
		return err
	}

	if d.OpenMenus.Inventory {
		kb, err := keybd_event.NewKeyBonding()
		if err != nil {
			return err
		}

		kb.SetKeys(keybd_event.VK_ESC)
		if err := kb.Launching(); err != nil {
			return err
		}

		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

// SimulateShiftLeftClick symuluje SHIFT+LEFT CLICK na określonych współrzędnych
func (ii *ItemInteraction) SimulateShiftLeftClick(x, y int) error {
	user32 := windows.NewLazySystemDLL("user32.dll")
	setCursorPos := user32.NewProc("SetCursorPos")
	mouseEvent := user32.NewProc("mouse_event")

	const (
		MOUSEEVENTF_LEFTDOWN = 0x0002
		MOUSEEVENTF_LEFTUP   = 0x0004
	)

	// Przesuń kursor
	setCursorPos.Call(uintptr(x), uintptr(y))
	time.Sleep(50 * time.Millisecond)

	// Naciśnij SHIFT
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return err
	}
	kb.HasSHIFT(true)
	kb.SetKeys() // Tylko SHIFT, bez innych klawiszy

	// Kliknij lewym przyciskiem myszy
	mouseEvent.Call(uintptr(MOUSEEVENTF_LEFTDOWN), 0, 0, 0, 0)
	time.Sleep(50 * time.Millisecond)
	mouseEvent.Call(uintptr(MOUSEEVENTF_LEFTUP), 0, 0, 0, 0)

	// Puść SHIFT
	kb.HasSHIFT(false)

	time.Sleep(100 * time.Millisecond)
	return nil
}

// ShiftClickItemInInventory wykonuje SHIFT+LEFT CLICK na przedmiocie w inventory
func (ii *ItemInteraction) ShiftClickItemInInventory(itm data.Item) error {
	if itm.Location != item.LocationInventory {
		return fmt.Errorf("item is not in inventory: %s", itm.Location)
	}

	screenX, screenY := ii.GridToScreenCoordinates(itm.Position)
	return ii.SimulateShiftLeftClick(screenX, screenY)
}

// MoveMouse przesuwa kursor bez klikania (do testowania)
func (ii *ItemInteraction) MoveMouse(x, y int) error {
	user32 := windows.NewLazySystemDLL("user32.dll")
	setCursorPos := user32.NewProc("SetCursorPos")
	
	setCursorPos.Call(uintptr(x), uintptr(y))
	return nil
}