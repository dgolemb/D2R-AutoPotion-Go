package lifewatcher

import (
	"fmt"
	"time"

	"github.com/Hefero/D2R-AutoPotion-Go/cmd/config"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/memory"
)

type BeltRefiller struct {
	gr              *memory.GameReader
	interaction     *memory.ItemInteraction
	lastRefillCheck time.Time
	debugMode       bool
}

func NewBeltRefiller(gr *memory.GameReader) *BeltRefiller {
	return &BeltRefiller{
		gr:              gr,
		interaction:     memory.NewItemInteraction(gr),
		lastRefillCheck: time.Now(),
		debugMode:       false,
	}
}

func (br *BeltRefiller) SetDebugMode(enabled bool) {
	br.debugMode = enabled
}

func (br *BeltRefiller) log(format string, args ...interface{}) {
	if br.debugMode {
		fmt.Printf("[BeltRefiller] "+format+"\n", args...)
	}
}

// CheckAndRefillBelt sprawdza stan paska i uzupełnia go automatycznie
func (br *BeltRefiller) CheckAndRefillBelt() error {
	if !config.Config.BeltRefill.Enabled {
		return nil
	}

	// Sprawdź czy minął odpowiedni czas od ostatniego sprawdzenia
	if time.Since(br.lastRefillCheck) < time.Duration(config.Config.BeltRefill.CheckInterval)*time.Second {
		return nil
	}

	br.lastRefillCheck = time.Now()

	// Pobierz dane gry
	d, err := br.gr.GetData()
	if err != nil {
		return fmt.Errorf("failed to get game data: %v", err)
	}

	br.log("Checking belt status...")

	// Uzupełnij mikstury HP
	if len(config.Config.Bindings.PotionHP) > 0 {
		for _, slot := range config.Config.Bindings.PotionHP {
			beltSlot := slot - 1 // Konwersja z 1-4 na 0-3
			if err := br.refillBeltSlot(d, beltSlot, "hp"); err != nil {
				br.log("Error refilling HP slot %d: %v", slot, err)
			}
		}
	}

	// Uzupełnij mikstury Many
	if len(config.Config.Bindings.PotionMANA) > 0 {
		for _, slot := range config.Config.Bindings.PotionMANA {
			beltSlot := slot - 1 // Konwersja z 1-4 na 0-3
			if err := br.refillBeltSlot(d, beltSlot, "mana"); err != nil {
				br.log("Error refilling MANA slot %d: %v", slot, err)
			}
		}
	}

	// Uzupełnij mikstury Rejuv (jeśli skonfigurowane)
	if len(config.Config.Bindings.PotionREJUV) > 0 {
		for _, slot := range config.Config.Bindings.PotionREJUV {
			beltSlot := slot - 1 // Konwersja z 1-4 na 0-3
			if err := br.refillBeltSlot(d, beltSlot, "rejuv"); err != nil {
				br.log("Error refilling REJUV slot %d: %v", slot, err)
			}
		}
	}

	return nil
}

// refillBeltSlot uzupełnia konkretny slot paska
func (br *BeltRefiller) refillBeltSlot(d data.Data, beltSlot int, potionType string) error {
	// Sprawdź ile mikstur jest już w slocie
	currentCount := d.Items.Belt.GetPotionCountInSlot(beltSlot)
	
	br.log("Slot %d (%s): %d potions", beltSlot, potionType, currentCount)

	// Jeśli slot ma wystarczająco dużo mikstur, pomiń
	if currentCount >= config.Config.BeltRefill.MinPotionsInSlot {
		br.log("Slot %d has enough potions (%d >= %d)", beltSlot, currentCount, config.Config.BeltRefill.MinPotionsInSlot)
		return nil
	}

	// Oblicz ile mikstur trzeba dodać
	potionsNeeded := config.Config.BeltRefill.MinPotionsInSlot - currentCount
	br.log("Need to add %d %s potions to slot %d", potionsNeeded, potionType, beltSlot)

	// Znajdź mikstury w ekwipunku
	availablePotions := br.interaction.FindPotionsInInventory(d.Items, potionType)
	
	if len(availablePotions) == 0 {
		br.log("No %s potions found in inventory for slot %d", potionType, beltSlot)
		return nil
	}

	br.log("Found %d %s potions in inventory", len(availablePotions), potionType)

	// Przenieś mikstury na pasek
	movedCount := 0
	for i := 0; i < len(availablePotions) && movedCount < potionsNeeded; i++ {
		potion := availablePotions[i]
		
		// Znajdź pierwszy wolny rząd w slocie
		targetRow := br.interaction.GetNextAvailableRowInBeltSlot(d.Items.Belt, beltSlot)
		
		if targetRow == -1 {
			br.log("Slot %d is full, cannot add more potions", beltSlot)
			break
		}

		br.log("Moving %s (UnitID: %d) to slot %d, row %d", potion.Name, potion.UnitID, beltSlot, targetRow)

		// Przenieś miksturę poprzez manipulację pamięcią
		if err := br.interaction.MovePotionToBelt(potion, beltSlot, targetRow); err != nil {
			br.log("Failed to move potion: %v", err)
			continue
		}

		movedCount++
		br.log("Successfully moved potion %d/%d", movedCount, potionsNeeded)

		// Krótka pauza między przeniesieniami dla stabilności
		time.Sleep(20 * time.Millisecond)

		// Odśwież dane po przeniesieniu
		d, err := br.gr.GetData()
		if err != nil {
			return fmt.Errorf("failed to refresh game data: %v", err)
		}
	}

	if movedCount > 0 {
		br.log("Successfully refilled slot %d with %d %s potions", beltSlot, movedCount, potionType)
	}

	return nil
}

// ForceRefillAll wymusza natychmiastowe uzupełnienie wszystkich slotów
// Przydatne do testowania lub ręcznego wywołania
func (br *BeltRefiller) ForceRefillAll() error {
	br.lastRefillCheck = time.Time{} // Reset timera
	return br.CheckAndRefillBelt()
}