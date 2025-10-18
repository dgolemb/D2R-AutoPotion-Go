package memory

import (
	"fmt"
	"time"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
)

type ItemInteraction struct {
	gr *GameReader
}

func NewItemInteraction(gr *GameReader) *ItemInteraction {
	return &ItemInteraction{gr: gr}
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

// MovePotionToBelt przenosi miksturę z inventory na pasek przez modyfikację pamięci
func (ii *ItemInteraction) MovePotionToBelt(itm data.Item, beltSlot int, targetRow int) error {
	if itm.Location != item.LocationInventory {
		return fmt.Errorf("item is not in inventory: %s", itm.Location)
	}

	// Znajdź adres przedmiotu
	itemAddress := ii.GetItemAddress(itm.UnitID)
	if itemAddress == 0 {
		return fmt.Errorf("could not find item address for UnitID %d", itm.UnitID)
	}

	// Odczytaj obecne dane
	itemDataBuffer := ii.gr.Process.ReadBytesFromMemory(itemAddress, 144)
	unitDataPtr := uintptr(ReadUIntFromBuffer(itemDataBuffer, 0x10, Uint64))
	pathPtr := uintptr(ReadUIntFromBuffer(itemDataBuffer, 0x38, Uint64))

	if unitDataPtr == 0 || pathPtr == 0 {
		return fmt.Errorf("invalid pointers for item")
	}

	// 1. Zmień lokalizację z inventory (0) na belt (2)
	if err := ii.gr.Process.WriteUInt(itemAddress+0x0C, 2, Uint32); err != nil {
		return fmt.Errorf("failed to write item location: %v", err)
	}

	// 2. Ustaw invPage na 0 (główne inventory/belt)
	if err := ii.gr.Process.WriteUInt(unitDataPtr+0x55, 0, Uint8); err != nil {
		return fmt.Errorf("failed to write invPage: %v", err)
	}

	// 3. Ustaw pozycję na pasku (X = slot kolumny, Y = rząd)
	if err := ii.gr.Process.WriteUInt(pathPtr+0x10, uint(beltSlot), Uint16); err != nil {
		return fmt.Errorf("failed to write belt X position: %v", err)
	}

	if err := ii.gr.Process.WriteUInt(pathPtr+0x14, uint(targetRow), Uint16); err != nil {
		return fmt.Errorf("failed to write belt Y position: %v", err)
	}

	// 4. Zaktualizuj dodatkowe pola pozycji
	if err := ii.gr.Process.WriteUInt(pathPtr+0x02, uint(beltSlot), Uint16); err != nil {
		return fmt.Errorf("failed to write room X: %v", err)
	}

	if err := ii.gr.Process.WriteUInt(pathPtr+0x06, uint(targetRow), Uint16); err != nil {
		return fmt.Errorf("failed to write room Y: %v", err)
	}

	// Małe opóźnienie dla stabilności
	time.Sleep(50 * time.Millisecond)
	
	return nil
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

// GetBeltSlotContents zwraca listę przedmiotów w danym slocie paska
func (ii *ItemInteraction) GetBeltSlotContents(beltSlot int) []data.Item {
	d, _ := ii.gr.GetData()
	var items []data.Item
	
	for _, beltItem := range d.Items.Belt.Items {
		if beltItem.Position.X == beltSlot {
			items = append(items, beltItem)
		}
	}
	
	return items
}

// DebugItemMemory wyświetla szczegóły pamięci przedmiotu (do debugowania)
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