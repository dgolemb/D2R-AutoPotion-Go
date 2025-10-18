package lifewatcher

import (
	"fmt"
	"time"

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
		debugMode:   false, // Ustaw na true dla szczegółowych logów
	}
}

// SetDebugMode włącza/wyłącza szczegółowe logi
func (br *BeltRefiller) SetDebugMode(enabled bool) {
	br.debugMode = enabled
}

// CheckAndRefillBelt sprawdza i uzupełnia pasek miksturami z inventory
func (br *BeltRefiller) CheckAndRefillBelt() error {
	// Nie częściej niż co 5 sekund
	if time.Since(br.lastRefill) < 5*time.Second {
		return nil
	}

	d, err := br.gr.GetData()
	if err != nil {
		return err
	}

	// Nie uzupełniaj w mieście
	if d.PlayerUnit.Area.IsTown() {
		return nil
	}

	belt := d.Items.Belt
	needsRefill := false

	if br.debugMode {
		fmt.Printf("\n=== Belt Refill Check ===\n")
		fmt.Printf("Belt type: %s (Max rows: %d)\n", belt.Name, belt.Rows())
	}

	// Sprawdź sloty z miksturami zdrowia
	hpSlots := belt.GetSlotsByPotionType(data.HealingPotion)
	for _, slot := range hpSlots {
		if belt.NeedsRefill(slot) {
			needsRefill = true
			if br.debugMode {
				fmt.Printf("HP Slot %d needs refill\n", slot)
			}
			if err := br.refillSlot(slot, data.HealingPotion, &d); err != nil {
				fmt.Printf("Error refilling HP slot %d: %v\n", slot, err)
			}
		}
	}

	// Sprawdź sloty z miksturami many
	manaSlots := belt.GetSlotsByPotionType(data.ManaPotion)
	for _, slot := range manaSlots {
		if belt.NeedsRefill(slot) {
			needsRefill = true
			if br.debugMode {
				fmt.Printf("Mana Slot %d needs refill\n", slot)
			}
			if err := br.refillSlot(slot, data.ManaPotion, &d); err != nil {
				fmt.Printf("Error refilling Mana slot %d: %v\n", slot, err)
			}
		}
	}

	if needsRefill {
		br.lastRefill = time.Now()
	}

	return nil
}

// refillSlot uzupełnia slot paska miksturami z inventory (przez modyfikację pamięci)
func (br *BeltRefiller) refillSlot(slotX int, potionType data.PotionType, d *data.Data) error {
	// Znajdź mikstury odpowiedniego typu w inventory
	potionsInInventory := br.findPotionsInInventory(potionType, d)
	
	if len(potionsInInventory) == 0 {
		if br.debugMode {
			fmt.Printf("No %s potions found in inventory\n", potionType)
		}
		return nil // Nie zwracaj błędu, po prostu brak mikstur
	}

	// Oblicz ile mikstur potrzebujemy
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
		fmt.Printf("Slot %d: current=%d, max=%d, need=%d potions\n", 
			slotX, currentCount, beltRows, neededCount)
		fmt.Printf("Found %d %s potions in inventory\n", len(potionsInInventory), potionType)
	}

	// Przenieś mikstury bezpośrednio w pamięci
	refilled := 0
	for _, potion := range potionsInInventory {
		if refilled >= neededCount {
			break
		}

		// Znajdź pierwszy wolny rząd w tym slocie
		targetRow, err := br.interaction.FindFirstEmptyRowInBeltSlot(slotX, beltRows)
		if err != nil {
			if br.debugMode {
				fmt.Printf("No empty row in slot %d: %v\n", slotX, err)
			}
			break
		}

		if br.debugMode {
			fmt.Printf("Moving %s (ID:%d) to slot %d, row %d\n", 
				potion.Name, potion.UnitID, slotX, targetRow)
			br.interaction.DebugItemMemory(potion)
		}

		// Przenieś miksturę przez modyfikację pamięci
		if err := br.interaction.MovePotionToBelt(potion, slotX, targetRow); err != nil {
			fmt.Printf("Failed to move potion: %v\n", err)
			continue
		}

		refilled++
		
		// Małe opóźnienie dla stabilności
		time.Sleep(100 * time.Millisecond)
		
		// Odśwież dane po każdej zmianie
		*d, _ = br.gr.GetData()
	}

	if refilled > 0 {
		fmt.Printf("✓ Refilled slot %d with %d %s potions\n", slotX, refilled, potionType)
	}
	
	return nil
}

// findPotionsInInventory znajduje wszystkie mikstury danego typu w inventory
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