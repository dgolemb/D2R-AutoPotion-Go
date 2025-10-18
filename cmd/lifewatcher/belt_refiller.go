package lifewatcher

import (
	"fmt"
	"time"

	"github.com/Hefero/D2R-AutoPotion-Go/cmd/config"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/memory"
)

type BeltRefiller struct {
	gr          *memory.GameReader
	interaction *memory.ItemInteraction
	lastRefill  time.Time
	debugMode   bool
}

func NewBeltRefiller(gr *memory.GameReader) *BeltRefiller {
	return &BeltRefiller{
		gr:          gr,
		interaction: memory.NewItemInteraction(gr),
		lastRefill:  time.Time{},
		debugMode:   false,
	}
}

func (br *BeltRefiller) SetDebugMode(enabled bool) {
	br.debugMode = enabled
}

func (br *BeltRefiller) CheckAndRefillBelt() error {
	if !config.Config.BeltRefill.Enabled {
		return nil
	}

	if time.Since(br.lastRefill) < time.Duration(config.Config.BeltRefill.CheckInterval)*time.Second {
		return nil
	}

	br.lastRefill = time.Now()

	d, err := br.gr.GetData()
	if err != nil {
		return err
	}

	if d.PlayerUnit.Area.IsTown() {
		return nil
	}

	belt := d.Items.Belt
	needsRefill := false

	if br.debugMode {
		fmt.Printf("\n=== Belt Refill Check ===\n")
		fmt.Printf("Belt type: %s (Max rows: %d)\n", belt.Name, belt.Rows())
		fmt.Printf("Total belt items: %d\n", len(belt.Items))
		fmt.Printf("Config: HP slots=%v, Mana slots=%v\n", 
			config.Config.Bindings.PotionHP, config.Config.Bindings.PotionMANA)
	}

	// Sprawdź sloty HP z konfiguracji
	for _, slotNum := range config.Config.Bindings.PotionHP {
		slot := slotNum - 1 // Konwersja z 1-4 na 0-3
		currentCount := belt.GetPotionCountInSlot(slot)
		
		if br.debugMode {
			fmt.Printf("HP Slot %d (key %d): %d potions\n", slot, slotNum, currentCount)
		}
		
		if currentCount < config.Config.BeltRefill.MinPotionsInSlot {
			needsRefill = true
			if br.debugMode {
				fmt.Printf("  -> Needs refill! (has %d, needs %d)\n", 
					currentCount, config.Config.BeltRefill.MinPotionsInSlot)
			}
			if err := br.refillSlot(slot, data.HealingPotion, &d); err != nil {
				fmt.Printf("Error refilling HP slot %d: %v\n", slot, err)
			}
		}
	}

	// Sprawdź sloty Mana z konfiguracji
	for _, slotNum := range config.Config.Bindings.PotionMANA {
		slot := slotNum - 1 // Konwersja z 1-4 na 0-3
		currentCount := belt.GetPotionCountInSlot(slot)
		
		if br.debugMode {
			fmt.Printf("Mana Slot %d (key %d): %d potions\n", slot, slotNum, currentCount)
		}
		
		if currentCount < config.Config.BeltRefill.MinPotionsInSlot {
			needsRefill = true
			if br.debugMode {
				fmt.Printf("  -> Needs refill! (has %d, needs %d)\n", 
					currentCount, config.Config.BeltRefill.MinPotionsInSlot)
			}
			if err := br.refillSlot(slot, data.ManaPotion, &d); err != nil {
				fmt.Printf("Error refilling Mana slot %d: %v\n", slot, err)
			}
		}
	}

	if needsRefill && br.debugMode {
		fmt.Println("=== Belt Refill Complete ===\n")
	}

	return nil
}

func (br *BeltRefiller) refillSlot(slotX int, potionType data.PotionType, d *data.Data) error {
	potionsInInventory := br.findPotionsInInventory(potionType, d)
	
	if br.debugMode {
		fmt.Printf("\n--- Refilling slot %d with %s ---\n", slotX, potionType)
		fmt.Printf("Found %d potions in inventory\n", len(potionsInInventory))
	}
	
	if len(potionsInInventory) == 0 {
		if br.debugMode {
			fmt.Printf("No %s potions in inventory\n", potionType)
		}
		return nil
	}

	beltRows := d.Items.Belt.Rows()
	currentCount := d.Items.Belt.GetPotionCountInSlot(slotX)
	neededCount := beltRows - currentCount

	if neededCount <= 0 {
		if br.debugMode {
			fmt.Printf("Slot %d already full (%d/%d)\n", slotX, currentCount, beltRows)
		}
		return nil
	}

	if br.debugMode {
		fmt.Printf("Current: %d, Max: %d, Need: %d\n", currentCount, beltRows, neededCount)
	}

	refilled := 0
	for i, potion := range potionsInInventory {
		if refilled >= neededCount {
			break
		}

		targetRow, err := br.interaction.FindFirstEmptyRowInBeltSlot(slotX, beltRows)
		if err != nil {
			if br.debugMode {
				fmt.Printf("No empty row in slot %d: %v\n", slotX, err)
			}
			break
		}

		if br.debugMode {
			fmt.Printf("[%d/%d] Moving %s (ID:%d, Loc:%s) to slot %d, row %d\n", 
				i+1, len(potionsInInventory), potion.Name, potion.UnitID, 
				potion.Location, slotX, targetRow)
			
			// Pokaż szczegóły pamięci PRZED przeniesieniem
			br.interaction.DebugItemMemory(potion)
		}

		// Próba przeniesienia
		err = br.interaction.MovePotionToBelt(potion, slotX, targetRow)
		
		if err != nil {
			fmt.Printf("❌ FAILED to move potion: %v\n", err)
			continue
		}
		
		if br.debugMode {
			fmt.Printf("✓ Write operations completed\n")
		}

		refilled++
		time.Sleep(100 * time.Millisecond)
		
		// Odśwież dane i sprawdź czy się przeniosło
		newData, err := br.gr.GetData()
		if err != nil {
			return fmt.Errorf("failed to refresh game data: %v", err)
		}
		
		// Sprawdź czy mikstura jest teraz na pasku
		found := false
		for _, beltItem := range newData.Items.Belt.Items {
			if beltItem.UnitID == potion.UnitID {
				found = true
				if br.debugMode {
					fmt.Printf("✓ VERIFIED: Potion now in belt at slot %d, row %d\n", 
						beltItem.Position.X, beltItem.Position.Y)
				}
				break
			}
		}
		
		if !found {
			fmt.Printf("⚠ WARNING: Potion not found in belt after move (writes succeeded but game didn't register change)\n")
			fmt.Printf("   This suggests the memory structure or timing needs adjustment\n")
		}
		
		*d = newData
	}

	if refilled > 0 {
		fmt.Printf("✓ Refilled slot %d with %d potions\n", slotX, refilled)
	} else {
		fmt.Printf("⚠ No potions were successfully moved to slot %d\n", slotX)
	}
	
	return nil
}

func (br *BeltRefiller) findPotionsInInventory(potionType data.PotionType, d *data.Data) []data.Item {
	var potions []data.Item

	for _, itm := range d.Items.AllItems {
		if itm.Location != item.LocationInventory {
			continue
		}

		switch potionType {
		case data.HealingPotion:
			if itm.IsHealingPotion() {
				potions = append(potions, itm)
			}
		case data.ManaPotion:
			if itm.IsManaPotion() {
				potions = append(potions, itm)
			}
		case data.RejuvenationPotion:
			if itm.IsRejuvPotion() {
				potions = append(potions, itm)
			}
		}
	}

	return potions
}