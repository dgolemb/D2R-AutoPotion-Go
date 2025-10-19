package memory

import (
	"fmt"
	"time"
	"syscall"
	"unsafe"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
	"golang.org/x/sys/windows"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procSendInput        = user32.NewProc("SendInput")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
)

// Stałe dla SendInput
const (
	INPUT_MOUSE    = 0
	INPUT_KEYBOARD = 1

	MOUSEEVENTF_LEFTDOWN  = 0x0002
	MOUSEEVENTF_LEFTUP    = 0x0004
	MOUSEEVENTF_MOVE      = 0x0001
	MOUSEEVENTF_ABSOLUTE  = 0x8000

	KEYEVENTF_KEYDOWN = 0x0000
	KEYEVENTF_KEYUP   = 0x0002

	VK_SHIFT = 0x10
	VK_I     = 0x49
	VK_ESCAPE = 0x1B
)

// Struktury dla SendInput
type POINT struct {
	X, Y int32
}

type MOUSEINPUT struct {
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type KEYBDINPUT struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type INPUT struct {
	dwType uint32
	union  [24]byte
}

type ItemInteraction struct {
	gr            *GameReader
	uiCoordinates *UICoordinates
	screenInfo    *ScreenInfo
}

func NewItemInteraction(gr *GameReader) *ItemInteraction {
	// Wykryj rozdzielczość ekranu przy inicjalizacji
	screenInfo, err := GetD2RScreenInfo(gr.Process.pid)
	if err != nil {
		fmt.Printf("Warning: Could not detect screen resolution: %v\n", err)
		fmt.Println("Using fallback 1920x1080 coordinates")
		// Użyj domyślnych wartości dla 1920x1080
		screenInfo = &ScreenInfo{
			ClientWidth:   1920,
			ClientHeight:  1080,
			ClientOffsetX: 0,
			ClientOffsetY: 0,
		}
	} else {
		fmt.Printf("✓ Detected screen resolution: %dx%d\n", screenInfo.ClientWidth, screenInfo.ClientHeight)
	}
	
	return &ItemInteraction{
		gr:            gr,
		screenInfo:    screenInfo,
		uiCoordinates: NewUICoordinates(screenInfo),
	}
}

// GetItemAddress znajduje adres przedmiotu w pamięci po UnitID
func (ii *ItemInteraction) GetItemAddress(unitID data.UnitID) uintptr {
	baseAddr := ii.gr.Process.moduleBaseAddressPtr + ii.gr.offset.UnitTable + (4 * 1024)
	unitTableBuffer := ii.gr.Process.ReadBytesFromMemory(baseAddr, 128*8)

	for i := 0; i < 128; i++ {
		itemOffset := 8 * i
		itemUnitPtr := uintptr(ReadUIntFromBuffer(unitTableBuffer, uint(itemOffset), Uint64))
		for itemUnitPtr > 0 {
			currentUnitID := ii.gr.Process.ReadUInt(itemUnitPtr+0x08, Uint32)
			if data.UnitID(currentUnitID) == unitID {
				return itemUnitPtr
			}
			itemUnitPtr = uintptr(ii.gr.Process.ReadUInt(itemUnitPtr+0x150, Uint64))
		}
	}
	return 0
}

// MovePotionToBelt - używa automatycznego skalowania współrzędnych
func (ii *ItemInteraction) MovePotionToBelt(itm data.Item, beltSlot int, targetRow int) error {
	if itm.Location != item.LocationInventory {
		return fmt.Errorf("item is not in inventory: %s", itm.Location)
	}

	// KROK 1: Otwórz inventory (jeśli zamknięte)
	d, _ := ii.gr.GetData()
	if !d.OpenMenus.Inventory {
		if err := ii.OpenInventory(); err != nil {
			return fmt.Errorf("failed to open inventory: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// KROK 2: Oblicz pozycję mikstury (używając auto-skalowania!)
	potionScreenX, potionScreenY := ii.uiCoordinates.GetInventoryCoordinates(itm.Position.X, itm.Position.Y)

	// KROK 3: Oblicz pozycję na pasku (używając auto-skalowania!)
	beltScreenX, beltScreenY := ii.uiCoordinates.GetBeltCoordinates(beltSlot, targetRow, d.Items.Belt.Rows())

	fmt.Printf("Moving potion from Inv(%d,%d)->Screen(%d,%d) to Belt(%d,%d)->Screen(%d,%d)\n", 
		itm.Position.X, itm.Position.Y, potionScreenX, potionScreenY,
		beltSlot, targetRow, beltScreenX, beltScreenY)

	// KROK 4: Kliknij na miksturę w inventory
	if err := ii.ClickAt(potionScreenX, potionScreenY); err != nil {
		return fmt.Errorf("failed to click potion: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// KROK 5: Kliknij na pozycję na pasku
	if err := ii.ClickAt(beltScreenX, beltScreenY); err != nil {
		return fmt.Errorf("failed to click belt slot: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// KROK 6: Zamknij inventory
	if err := ii.CloseInventory(); err != nil {
		return fmt.Errorf("failed to close inventory: %v", err)
	}

	fmt.Printf("✓ Moved %s to belt slot %d via UI simulation\n", itm.Name, beltSlot)
	return nil
}

// GridToScreenCoordinates - DEPRECATED, użyj uiCoordinates.GetInventoryCoordinates()
func (ii *ItemInteraction) GridToScreenCoordinates(gridPos data.Position) (int, int) {
	return ii.uiCoordinates.GetInventoryCoordinates(gridPos.X, gridPos.Y)
}

// FindFirstEmptyRowInBeltSlot znajduje pierwszy wolny rząd w danym slocie paska
func (ii *ItemInteraction) FindFirstEmptyRowInBeltSlot(beltSlot int, maxRows int) (int, error) {
	d, err := ii.gr.GetData()
	if err != nil {
		return -1, err
	}

	// Sprawdź każdy rząd od dołu (0) do góry
	for row := 0; row < maxRows; row++ {
		occupied := false
		for _, beltItem := range d.Items.Belt.Items {
			if beltItem.Position.X == beltSlot && beltItem.Position.Y == row {
				occupied = true
				break
			}
		}
		if !occupied {
			return row, nil
		}
	}

	return -1, fmt.Errorf("no empty row found in belt slot %d", beltSlot)
}

// DebugItemMemory wyświetla szczegóły pamięci przedmiotu
func (ii *ItemInteraction) DebugItemMemory(itm data.Item) {
	itemAddress := ii.GetItemAddress(itm.UnitID)
	if itemAddress == 0 {
		fmt.Printf("Could not find address for item %s (ID: %d)\n", itm.Name, itm.UnitID)
		return
	}

	fmt.Printf("\n=== ITEM MEMORY DEBUG: %s ===\n", itm.Name)
	fmt.Printf("Item Address: 0x%X\n", itemAddress)
	fmt.Printf("Unit ID: %d\n", itm.UnitID)
	
	itemDataBuffer := ii.gr.Process.ReadBytesFromMemory(itemAddress, 144)
	
	itemType := ReadUIntFromBuffer(itemDataBuffer, 0x00, Uint32)
	txtFileNo := ReadUIntFromBuffer(itemDataBuffer, 0x04, Uint32)
	unitID := ReadUIntFromBuffer(itemDataBuffer, 0x08, Uint32)
	itemLoc := ReadUIntFromBuffer(itemDataBuffer, 0x0C, Uint32)
	unitDataPtr := uintptr(ReadUIntFromBuffer(itemDataBuffer, 0x10, Uint64))
	pathPtr := uintptr(ReadUIntFromBuffer(itemDataBuffer, 0x38, Uint64))
	
	fmt.Printf("Item Type: %d\n", itemType)
	fmt.Printf("TXT File No: %d\n", txtFileNo)
	fmt.Printf("Unit ID: %d\n", unitID)
	fmt.Printf("Item Location: %d (0=inventory, 1=equipped, 2=belt)\n", itemLoc)
	fmt.Printf("Unit Data Ptr: 0x%X\n", unitDataPtr)
	fmt.Printf("Path Ptr: 0x%X\n", pathPtr)
	
	if unitDataPtr > 0 {
		unitDataBuffer := ii.gr.Process.ReadBytesFromMemory(unitDataPtr, 144)
		invPage := ReadUIntFromBuffer(unitDataBuffer, 0x55, Uint8)
		fmt.Printf("Inv Page: %d\n", invPage)
	}
	
	if pathPtr > 0 {
		pathBuffer := ii.gr.Process.ReadBytesFromMemory(pathPtr, 144)
		itemX := ReadUIntFromBuffer(pathBuffer, 0x10, Uint16)
		itemY := ReadUIntFromBuffer(pathBuffer, 0x14, Uint16)
		fmt.Printf("Position: X=%d, Y=%d\n", itemX, itemY)
	}
	
	fmt.Println("========================")
}

// DebugUICoordinates wyświetla wszystkie obliczone współrzędne UI
func (ii *ItemInteraction) DebugUICoordinates() {
	ii.uiCoordinates.DebugDrawCoordinates()
}

// ========== METODY DO SYMULACJI KLIKNIĘĆ ==========

// GetCursorPosition zwraca aktualną pozycję kursora
func (ii *ItemInteraction) GetCursorPosition() (int, int) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return int(pt.X), int(pt.Y)
}

// ClickAt wykonuje kliknięcie na podanych współrzędnych
func (ii *ItemInteraction) ClickAt(x, y int) error {
	// Zapisz aktualną pozycję kursora
	originalX, originalY := ii.GetCursorPosition()

	// Przesuń kursor
	procSetCursorPos.Call(uintptr(x), uintptr(y))
	time.Sleep(50 * time.Millisecond)

	// Kliknij
	if err := ii.simulateMouseClick(); err != nil {
		procSetCursorPos.Call(uintptr(originalX), uintptr(originalY))
		return err
	}

	time.Sleep(50 * time.Millisecond)

	// Przywróć pozycję kursora
	procSetCursorPos.Call(uintptr(originalX), uintptr(originalY))

	return nil
}

// simulateMouseClick symuluje kliknięcie lewym przyciskiem
func (ii *ItemInteraction) simulateMouseClick() error {
	// Naciśnij
	down := INPUT{dwType: INPUT_MOUSE}
	mouseDown := (*MOUSEINPUT)(unsafe.Pointer(&down.union[0]))
	mouseDown.dwFlags = MOUSEEVENTF_LEFTDOWN

	ret, _, _ := procSendInput.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&down)),
		uintptr(unsafe.Sizeof(down)),
	)
	if ret == 0 {
		return fmt.Errorf("failed to send mouse down")
	}

	time.Sleep(30 * time.Millisecond)

	// Puść
	up := INPUT{dwType: INPUT_MOUSE}
	mouseUp := (*MOUSEINPUT)(unsafe.Pointer(&up.union[0]))
	mouseUp.dwFlags = MOUSEEVENTF_LEFTUP

	ret, _, _ = procSendInput.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&up)),
		uintptr(unsafe.Sizeof(up)),
	)
	if ret == 0 {
		return fmt.Errorf("failed to send mouse up")
	}

	return nil
}

// OpenInventory otwiera inventory (klawisz I)
func (ii *ItemInteraction) OpenInventory() error {
	return ii.PressKey(VK_I)
}

// CloseInventory zamyka inventory (ESC)
func (ii *ItemInteraction) CloseInventory() error {
	return ii.PressKey(VK_ESCAPE)
}

// PressKey symuluje naciśnięcie klawisza
func (ii *ItemInteraction) PressKey(vkCode uint16) error {
	// Naciśnij
	down := INPUT{dwType: INPUT_KEYBOARD}
	kbDown := (*KEYBDINPUT)(unsafe.Pointer(&down.union[0]))
	kbDown.wVk = vkCode

	ret, _, _ := procSendInput.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&down)),
		uintptr(unsafe.Sizeof(down)),
	)
	if ret == 0 {
		return fmt.Errorf("failed to send key down")
	}

	time.Sleep(30 * time.Millisecond)

	// Puść
	up := INPUT{dwType: INPUT_KEYBOARD}
	kbUp := (*KEYBDINPUT)(unsafe.Pointer(&up.union[0]))
	kbUp.wVk = vkCode
	kbUp.dwFlags = KEYEVENTF_KEYUP

	ret, _, _ = procSendInput.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&up)),
		uintptr(unsafe.Sizeof(up)),
	)
	if ret == 0 {
		return fmt.Errorf("failed to send key up")
	}

	return nil
}