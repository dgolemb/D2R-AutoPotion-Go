package lifewatcher

import (
	"fmt"
	"time"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/memory"
)

type BeltRefiller struct {
	gr          *GameReader
	interaction *memory.ItemInteraction
	lastRefill  time.Time
}

func NewBeltRefiller(gr *memory.GameReader) *BeltRefiller {
	return &BeltRefiller{
		gr:          gr,
		interaction: memory.NewItemInteraction(gr),
		lastRefill:  time.Time{},
	}
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

	// Nie uzupełniaj w mieście lub podczas walki
	if d.PlayerUnit.Area.IsTown() {
		return nil
	}

	belt := d.Items.Belt
	needsRefill := false

	// Sprawdź sloty z miksturami zdrowia
	hpSlots := belt.GetSlotsByPotionType(data.HealingPotion)
	for _, slot := range hpSlots {
		if belt.NeedsRefill(slot) {
			needsRefill = true
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

// refillSlot uzupełnia konkretny slot paska
func (br *BeltRefiller) refillSlot(slotX int, potionType data.PotionType, d *data.Data) error {
	// Znajdź mikstury odpowiedniego typu w inventory
	potionsInInventory := br.findPotionsInInventory(potionType, d)
	
	if len(potionsInInventory) == 0 {
		return fmt.Errorf("no %s potions found in inventory", potionType)
	}

	// Otwórz inventory
	if err := br.interaction.OpenInventory(); err != nil {
		return fmt.Errorf("failed to open inventory: %v", err)
	}
	defer br.interaction.CloseInventory()

	// Oblicz ile mikstur potrzebujemy
	beltRows := d.Items.Belt.Rows()
	currentCount := d.Items.Belt.GetPotionCountInSlot(slotX)
	neededCount := beltRows - currentCount

	if neededCount <= 0 {
		return nil
	}

	// Uzupełnij slot
	refilled := 0
	for _, potion := range potionsInInventory {
		if refilled >= neededCount {
			break
		}

		// SHIFT+LEFT CLICK na miksturze w inventory
		if err := br.interaction.ShiftClickItemInInventory(potion); err != nil {
			fmt.Printf("Failed to shift-click potion: %v\n", err)
			continue
		}

		refilled++
		time.Sleep(150 * time.Millisecond) // Opóźnienie między kliknięciami
	}

	fmt.Printf("Refilled slot %d with %d %s potions\n", slotX, refilled, potionType)
	return nil
}

// findPotionsInInventory znajduje wszystkie mikstury danego typu w inventory
func (br *BeltRefiller) findPotionsInInventory(potionType data.PotionType, d *data.Data) []data.Item {
	var potions []data.Item

	for _, item := range d.Items.AllItems {
		if item.Location != data.item.LocationInventory {
			continue
		}

		switch potionType {
		case data.HealingPotion:
			if item.IsHealingPotion() {
				potions = append(potions, item)
			}
		case data.ManaPotion:
			if item.IsManaPotion() {
				potions = append(potions, item)
			}
		case data.RejuvenationPotion:
			if item.IsRejuvPotion() {
				potions = append(potions, item)
			}
		}
	}

	return potions
}