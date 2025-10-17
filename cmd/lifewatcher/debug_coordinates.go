package lifewatcher

import (
	"fmt"
	"time"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"  // ✅ Dodaj import
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/memory"
)

type CoordinateDebugger struct {
	gr          *memory.GameReader
	interaction *memory.ItemInteraction
}

func NewCoordinateDebugger(gr *memory.GameReader) *CoordinateDebugger {
	return &CoordinateDebugger{
		gr:          gr,
		interaction: memory.NewItemInteraction(gr),
	}
}

// DebugInventoryCoordinates wyświetla wszystkie mikstury w inventory i ich współrzędne
func (cd *CoordinateDebugger) DebugInventoryCoordinates() {
	d, err := cd.gr.GetData()
	if err != nil {
		fmt.Printf("Error getting data: %v\n", err)
		return
	}

	fmt.Println("\n=== INVENTORY ITEMS DEBUG ===")
	fmt.Printf("Inventory open: %v\n", d.OpenMenus.Inventory)
	fmt.Println("Items in inventory:")

	for i, itm := range d.Items.AllItems {
		if itm.Location == item.LocationInventory {  // ✅ Zmieniono z data.item na item
			screenX, screenY := cd.interaction.GridToScreenCoordinates(itm.Position)
			
			potionType := "Other"
			if itm.IsHealingPotion() {
				potionType = "HP Potion"
			} else if itm.IsManaPotion() {
				potionType = "Mana Potion"
			} else if itm.IsRejuvPotion() {
				potionType = "Rejuv Potion"
			}

			fmt.Printf("[%d] %s - Grid(%d,%d) -> Screen(%d,%d) - Type: %s\n",
				i, itm.Name, itm.Position.X, itm.Position.Y, screenX, screenY, potionType)
		}
	}

	fmt.Println("\n=== BELT ITEMS DEBUG ===")
	for i, itm := range d.Items.Belt.Items {
		fmt.Printf("[%d] %s - Slot(%d,%d)\n", i, itm.Name, itm.Position.X, itm.Position.Y)
	}

	for slot := 0; slot <= 3; slot++ {
		count := d.Items.Belt.GetPotionCountInSlot(slot)
		fmt.Printf("Slot %d: %d potions\n", slot, count)
	}
}

// TestClickOnPotion testuje kliknięcie na pierwszej miksturze w inventory
func (cd *CoordinateDebugger) TestClickOnPotion() error {
	d, err := cd.gr.GetData()
	if err != nil {
		return err
	}

	// Otwórz inventory
	if err := cd.interaction.OpenInventory(); err != nil {
		return fmt.Errorf("failed to open inventory: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Znajdź pierwszą miksturę HP w inventory
	var testPotion *data.Item
	for _, itm := range d.Items.AllItems {
		if itm.Location == item.LocationInventory && itm.IsHealingPotion() {  // ✅ Zmieniono z data.item na item
			testPotion = &itm
			break
		}
	}

	if testPotion == nil {
		cd.interaction.CloseInventory()
		return fmt.Errorf("no HP potion found in inventory for testing")
	}

	screenX, screenY := cd.interaction.GridToScreenCoordinates(testPotion.Position)
	
	fmt.Printf("\n=== TEST CLICK ===\n")
	fmt.Printf("Clicking on: %s\n", testPotion.Name)
	fmt.Printf("Grid position: (%d, %d)\n", testPotion.Position.X, testPotion.Position.Y)
	fmt.Printf("Screen position: (%d, %d)\n", screenX, screenY)
	fmt.Println("Clicking in 2 seconds...")
	
	time.Sleep(2 * time.Second)

	// Wykonaj SHIFT+CLICK
	if err := cd.interaction.SimulateShiftLeftClick(screenX, screenY); err != nil {
		cd.interaction.CloseInventory()
		return fmt.Errorf("click failed: %v", err)
	}

	fmt.Println("Click executed!")
	
	time.Sleep(1 * time.Second)
	cd.interaction.CloseInventory()

	return nil
}

// TestMouseMovement tylko przesuwa mysz bez klikania (do sprawdzenia współrzędnych)
func (cd *CoordinateDebugger) TestMouseMovement() error {
	d, err := cd.gr.GetData()
	if err != nil {
		return err
	}

	if err := cd.interaction.OpenInventory(); err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n=== MOUSE MOVEMENT TEST ===")
	fmt.Println("Moving mouse to each potion position (without clicking)")
	fmt.Println("Watch where the cursor goes to verify coordinates...")

	count := 0
	for _, itm := range d.Items.AllItems {
		if itm.Location == item.LocationInventory &&  // ✅ Zmieniono z data.item na item
		   (itm.IsHealingPotion() || itm.IsManaPotion()) {
			
			screenX, screenY := cd.interaction.GridToScreenCoordinates(itm.Position)
			
			fmt.Printf("\nMoving to: %s at Grid(%d,%d) -> Screen(%d,%d)\n",
				itm.Name, itm.Position.X, itm.Position.Y, screenX, screenY)
			
			cd.interaction.MoveMouse(screenX, screenY)
			time.Sleep(1500 * time.Millisecond)
			
			count++
			if count >= 5 { // Test tylko 5 pierwszych
				break
			}
		}
	}

	cd.interaction.CloseInventory()
	return nil
}