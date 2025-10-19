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
		fmt.Printf("Belt: %s (Rows: %d) | Items: %d\n", 
			belt.Name, belt.Rows(), len(belt.Items))
	}

	// Sprawdź HP slots z config
	for _, slotNum := range config.Config.Bindings.PotionHP {
		slot := slotNum - 1
		count := belt.GetPotionCountInSlot(slot)
		
		if count < config.Config.BeltRefill.MinPotionsInSlot {
			needsRefill = true
			if br.debugMode {
				fmt.Printf("HP Slot %d: %d/%d potions → refilling\n", 
					slot, count, config.Config.BeltRefill.MinPotionsInSlot)
			}
			if err := br.refillSlot(slot, data.HealingPotion, &d); err != nil && br.debugMode {
				fmt.Printf("  ⚠ Error: %v\n", err)
			}
		}
	}

	// Sprawdź Mana slots z config
	for _, slotNum := range config.Config.Bindings.PotionMANA {
		slot := slotNum - 1
		count := belt.GetPotionCountInSlot(slot)
		
		if count < config.Config.BeltRefill.MinPotionsInSlot {
			needsRefill = true
			if br.debugMode {
				fmt.Printf("Mana Slot %d: %d/%d potions → refilling\n", 
					slot, count, config.Config.BeltRefill.MinPotionsInSlot)
			}
			if err := br.refillSlot(slot, data.ManaPotion, &d); err != nil && br.debugMode {
				fmt.Printf("  ⚠ Error: %v\n", err)
			}
		}
	}

	if needsRefill {
		br.lastRefill = time.Now()
		if br.debugMode {
			fmt.Println("=== Refill Complete ===")
		}
	}

	return nil
}

func (br *BeltRefiller) refillSlot(slotX int, potionType data.PotionType, d *data.Data) error {
	potionsInInventory := br.findPotionsInInventory(potionType, d)
	
	if len(potionsInInventory) == 0 {
		if br.debugMode {
			fmt.Printf("  ⚠ No %s in inventory\n", potionType)
		}
		return nil
	}

	beltRows := d.Items.Belt.Rows()
	currentCount := d.Items.Belt.GetPotionCountInSlot(slotX)
	neededCount := beltRows - currentCount

	if neededCount <= 0 {
		return nil
	}

	if br.debugMode {
		fmt.Printf("  Found %d %s in inventory\n", len(potionsInInventory), potionType)
	}

	refilled := 0
	for _, potion := range potionsInInventory {
		if refilled >= neededCount {
			break
		}

		targetRow, err := br.interaction.FindFirstEmptyRowInBeltSlot(slotX, beltRows)
		if err != nil {
			break
		}

		// ✅ TYLKO W DEBUG MODE pokaż szczegóły
		if br.debugMode && config.Config.Debug.ShowMemoryDetails {
			fmt.Printf("  [%d/%d] %s (ID:%d, TxtNo:%d) → slot %d, row %d\n", 
				refilled+1, neededCount, potion.Name, potion.UnitID, potion.TxtFileNo, slotX, targetRow)
		}

		if err := br.interaction.MovePotionToBelt(potion, slotX, targetRow); err != nil {
			if br.debugMode {
				fmt.Printf("  ❌ Move failed: %v\n", err)
			}
			continue
		}

		refilled++
		time.Sleep(100 * time.Millisecond)
		*d, _ = br.gr.GetData()
	}

	if refilled > 0 && br.debugMode {
		fmt.Printf("  ✓ Moved %d potions to slot %d\n", refilled, slotX)
	} else if refilled == 0 && br.debugMode {
		fmt.Printf("  ⚠ Failed to move any potions\n")
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