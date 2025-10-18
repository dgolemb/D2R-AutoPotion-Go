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
// Parametry:
//   itm - przedmiot do przeniesienia
//   beltSlot - slot paska (0-3), odpowiada klawiszy 1-4
//   targetRow - rząd w slocie (0-3), 0 to dół paska
func (ii *ItemInteraction) MovePotionToBelt(itm data.Item, beltSlot int, targetRow int) error {
	if itm.Location != item.LocationInventory {
		return fmt.Errorf("item is not in inventory, current location: %d", itm.Location)
	}

	if beltSlot < 0 || beltSlot > 3 {
		return fmt.Errorf("invalid belt slot: %d, must be 0-3", beltSlot)
	}

	if targetRow < 0 || targetRow > 3 {
		return fmt.Errorf("invalid target row: %d, must be 0-3", targetRow)
	}

	// Znajdź adres przedmiotu w pamięci
	itemAddress := ii.GetItemAddress(itm.UnitID)
	if itemAddress == 0 {
		return fmt.Errorf("could not find item address for UnitID %d", itm.UnitID)
	}

	// Odczytaj wskaźniki struktury przedmiotu
	itemDataBuffer := ii.gr.Process.ReadBytesFromMemory(itemAddress, 144)
	unitDataPtr := uintptr(ReadUIntFromBuffer(itemDataBuffer, 0x10, Uint64))
	pathPtr := uintptr(ReadUIntFromBuffer(itemDataBuffer, 0x38, Uint64))

	if unitDataPtr == 0 || pathPtr == 0 {
		return fmt.Errorf("invalid pointers for item (unitData: %x, path: %x)", unitDataPtr, pathPtr)
	}

	// Krok 1: Zmień lokalizację przedmiotu z inventory (0) na belt (2)
	if err := ii.gr.Process.WriteUInt(itemAddress+0x0C, uint64(2), Uint32); err != nil {
		return fmt.Errorf("failed to write item location: %v", err)
	}

	// Krok 2: Ustaw invPage na 0 (dla paska)
	if err := ii.gr.Process.WriteUInt(unitDataPtr+0x55, uint64(0), Uint8); err != nil {
		return fmt.Errorf("failed to write invPage: %v", err)
	}

	// Krok 3: Ustaw pozycję na pasku (X = slot, Y = rząd)
	// X określa kolumnę paska (0-3 = sloty odpowiadające klawiszom 1-4)
	// Y określa rząd w kolumnie (0 = dół, 3 = góra)
	if err := ii.gr.Process.WriteUInt(pathPtr+0x10, uint64(beltSlot), Uint16); err != nil {
		return fmt.Errorf("failed to write X position: %v", err)
	}

	if err := ii.gr.Process.WriteUInt(pathPtr+0x14, uint64(targetRow), Uint16); err != nil {
		return fmt.Errorf("failed to write Y position: %v", err)
	}

	// Krok 4: Zaktualizuj dodatkowe pola pozycji (dla pewności)
	if err := ii.gr.Process.WriteUInt(pathPtr+0x02, uint64(beltSlot), Uint16); err != nil {
		return fmt.Errorf("failed to write room X: %v", err)
	}

	if err := ii.gr.Process.WriteUInt(pathPtr+0x06, uint64(targetRow), Uint16); err != nil {
		return fmt.Errorf("failed to write room Y: %v", err)
	}

	// Mała pauza dla synchronizacji pamięci
	time.Sleep(10 * time.Millisecond)

	return nil
}

// GetItemsInInventory zwraca wszystkie przedmioty w ekwipunku
func (ii *ItemInteraction) GetItemsInInventory(items data.Items) []data.Item {
	var inventoryItems []data.Item
	for _, itm := range items.AllItems {
		if itm.Location == item.LocationInventory {
			inventoryItems = append(inventoryItems, itm)
		}
	}
	return inventoryItems
}

// FindPotionsInInventory znajduje mikstury określonego typu w ekwipunku
func (ii *ItemInteraction) FindPotionsInInventory(items data.Items, potionType string) []data.Item {
	var potions []data.Item
	inventoryItems := ii.GetItemsInInventory(items)

	for _, itm := range inventoryItems {
		switch potionType {
		case "hp":
			if itm.IsHealingPotion() {
				potions = append(potions, itm)
			}
		case "mana":
			if itm.IsManaPotion() {
				potions = append(potions, itm)
			}
		case "rejuv":
			if itm.IsRejuvPotion() {
				potions = append(potions, itm)
			}
		}
	}
	return potions
}

// GetNextAvailableRowInBeltSlot znajduje pierwszy wolny rząd w danym slocie paska
// Zwraca numer rzędu (0-3) lub -1 jeśli slot jest pełny
func (ii *ItemInteraction) GetNextAvailableRowInBeltSlot(belt data.Belt, beltSlot int) int {
	// Sprawdź każdy rząd od dołu (0) do góry (3)
	for row := 0; row <= 3; row++ {
		occupied := false
		for _, itm := range belt.Items {
			if itm.Position.X == beltSlot && itm.Position.Y == row {
				occupied = true
				break
			}
		}
		if !occupied {
			return row
		}
	}
	return -1 // Slot pełny
}