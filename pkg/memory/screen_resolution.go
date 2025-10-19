package memory

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32DLL            = windows.NewLazyDLL("user32.dll")
	procFindWindowW      = user32DLL.NewProc("FindWindowW")
	procGetWindowRect    = user32DLL.NewProc("GetWindowRect")
	procGetClientRect    = user32DLL.NewProc("GetClientRect")
	procClientToScreen   = user32DLL.NewProc("ClientToScreen")
)

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type POINT struct {
	X int32
	Y int32
}

type ScreenInfo struct {
	WindowWidth    int
	WindowHeight   int
	ClientWidth    int
	ClientHeight   int
	ClientOffsetX  int
	ClientOffsetY  int
	WindowHandle   windows.Handle
}

// GetD2RScreenInfo pobiera informacje o oknie D2R i jego rozdzielczości
func GetD2RScreenInfo(pid uint) (*ScreenInfo, error) {
	// Znajdź okno D2R po tytule
	className, _ := syscall.UTF16PtrFromString("Diablo II: Resurrected")
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(className)))
	
	if hwnd == 0 {
		// Spróbuj alternatywnych nazw
		className, _ = syscall.UTF16PtrFromString("D2R")
		hwnd, _, _ = procFindWindowW.Call(0, uintptr(unsafe.Pointer(className)))
	}
	
	if hwnd == 0 {
		return nil, fmt.Errorf("could not find D2R window")
	}
	
	// Pobierz rozmiar całego okna (z ramką)
	var windowRect RECT
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&windowRect)))
	if ret == 0 {
		return nil, fmt.Errorf("failed to get window rect")
	}
	
	// Pobierz rozmiar obszaru klienta (bez ramki okna)
	var clientRect RECT
	ret, _, _ = procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&clientRect)))
	if ret == 0 {
		return nil, fmt.Errorf("failed to get client rect")
	}
	
	// Oblicz offset między oknem a obszarem klienta
	var clientPoint POINT
	clientPoint.X = 0
	clientPoint.Y = 0
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&clientPoint)))
	
	info := &ScreenInfo{
		WindowWidth:   int(windowRect.Right - windowRect.Left),
		WindowHeight:  int(windowRect.Bottom - windowRect.Top),
		ClientWidth:   int(clientRect.Right - clientRect.Left),
		ClientHeight:  int(clientRect.Bottom - clientRect.Top),
		ClientOffsetX: int(clientPoint.X - windowRect.Left),
		ClientOffsetY: int(clientPoint.Y - windowRect.Top),
		WindowHandle:  windows.Handle(hwnd),
	}
	
	return info, nil
}

// UICoordinates przechowuje proporcje UI względem rozdzielczości
type UICoordinates struct {
	screenInfo *ScreenInfo
}

func NewUICoordinates(screenInfo *ScreenInfo) *UICoordinates {
	return &UICoordinates{
		screenInfo: screenInfo,
	}
}

// GetInventoryCoordinates zwraca współrzędne inventory z automatycznym skalowaniem
func (ui *UICoordinates) GetInventoryCoordinates(gridX, gridY int) (int, int) {
	// Proporcje dla inventory (bazowane na 1920x1080)
	// Inventory zaczyna się w prawym dolnym rogu ekranu
	
	const (
		// Proporcje względem szerokości/wysokości ekranu
		inventoryRightMargin  = 0.025  // 2.5% od prawej krawędzi
		inventoryBottomMargin = 0.065  // 6.5% od dolnej krawędzi
		inventoryWidth        = 0.185  // 18.5% szerokości ekranu
		inventoryHeight       = 0.370  // 37% wysokości ekranu
		inventoryColumns      = 10
		inventoryRows         = 4
	)
	
	// Oblicz pozycję inventory
	invRight := ui.screenInfo.ClientWidth - int(float64(ui.screenInfo.ClientWidth)*inventoryRightMargin)
	invBottom := ui.screenInfo.ClientHeight - int(float64(ui.screenInfo.ClientHeight)*inventoryBottomMargin)
	invWidth := int(float64(ui.screenInfo.ClientWidth) * inventoryWidth)
	invHeight := int(float64(ui.screenInfo.ClientHeight) * inventoryHeight)
	
	invLeft := invRight - invWidth
	invTop := invBottom - invHeight
	
	// Oblicz rozmiar komórki
	cellWidth := float64(invWidth) / inventoryColumns
	cellHeight := float64(invHeight) / inventoryRows
	
	// Oblicz współrzędne konkretnej komórki (środek)
	screenX := invLeft + int(float64(gridX)*cellWidth+cellWidth/2)
	screenY := invTop + int(float64(gridY)*cellHeight+cellHeight/2)
	
	// Dodaj offset okna
	screenX += ui.screenInfo.ClientOffsetX
	screenY += ui.screenInfo.ClientOffsetY
	
	return screenX, screenY
}

// GetBeltCoordinates zwraca współrzędne paska z automatycznym skalowaniem
func (ui *UICoordinates) GetBeltCoordinates(slotX, rowY int, beltRows int) (int, int) {
	// Proporcje dla paska (bazowane na 1920x1080)
	// Pasek znajduje się na środku dolnej krawędzi ekranu
	
	const (
		beltCenterX        = 0.50   // Środek ekranu w poziomie (50%)
		beltBottomMargin   = 0.008  // 0.8% od dolnej krawędzi
		beltWidth          = 0.180  // 18% szerokości ekranu (4 sloty)
		beltRowHeight      = 0.027  // 2.7% wysokości ekranu (wysokość rzędu)
		beltColumns        = 4
	)
	
	// Oblicz pozycję środka paska
	centerX := int(float64(ui.screenInfo.ClientWidth) * beltCenterX)
	bottomY := ui.screenInfo.ClientHeight - int(float64(ui.screenInfo.ClientHeight)*beltBottomMargin)
	
	// Oblicz szerokość paska i rozmiar slotu
	totalBeltWidth := int(float64(ui.screenInfo.ClientWidth) * beltWidth)
	slotWidth := totalBeltWidth / beltColumns
	rowHeight := int(float64(ui.screenInfo.ClientHeight) * beltRowHeight)
	
	// Oblicz pozycję lewej krawędzi paska
	beltLeft := centerX - totalBeltWidth/2
	
	// Oblicz współrzędne konkretnego slotu (środek)
	screenX := beltLeft + slotX*slotWidth + slotWidth/2
	screenY := bottomY - rowY*rowHeight - rowHeight/2
	
	// Dodaj offset okna
	screenX += ui.screenInfo.ClientOffsetX
	screenY += ui.screenInfo.ClientOffsetY
	
	return screenX, screenY
}

// DebugDrawCoordinates - pomocnicza funkcja do debugowania współrzędnych
func (ui *UICoordinates) DebugDrawCoordinates() {
	fmt.Printf("\n=== Screen Info ===\n")
	fmt.Printf("Window Size: %dx%d\n", ui.screenInfo.WindowWidth, ui.screenInfo.WindowHeight)
	fmt.Printf("Client Size: %dx%d\n", ui.screenInfo.ClientWidth, ui.screenInfo.ClientHeight)
	fmt.Printf("Client Offset: (%d, %d)\n", ui.screenInfo.ClientOffsetX, ui.screenInfo.ClientOffsetY)
	
	fmt.Printf("\n=== Inventory Coordinates (Grid -> Screen) ===\n")
	for y := 0; y < 4; y++ {
		for x := 0; x < 10; x++ {
			screenX, screenY := ui.GetInventoryCoordinates(x, y)
			fmt.Printf("Grid(%d,%d) -> Screen(%d,%d)  ", x, y, screenX, screenY)
		}
		fmt.Println()
	}
	
	fmt.Printf("\n=== Belt Coordinates (Slot -> Screen) ===\n")
	for row := 0; row < 4; row++ {
		for slot := 0; slot < 4; slot++ {
			screenX, screenY := ui.GetBeltCoordinates(slot, row, 4)
			fmt.Printf("Slot(%d,%d) -> Screen(%d,%d)  ", slot, row, screenX, screenY)
		}
		fmt.Println()
	}
}