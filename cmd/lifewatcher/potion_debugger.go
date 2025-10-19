package lifewatcher

import (
	"fmt"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/memory"
)

// PotionDebugger - narzędzie do debugowania mikstur i txtFileNo
type PotionDebugger struct {
	gr *memory.GameReader
}

func NewPotionDebugger(gr *memory.GameReader) *PotionDebugger {
	return &PotionDebugger{gr: gr}
}

// DebugPotions wyświetla szczegóły wszystkich mikstur z ich txtFileNo
func (pd *PotionDebugger) DebugPotions() {
	d, err := pd.gr.GetData()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\n╔══════════════════════════════════════╗")
	fmt.Println("║    POTION TXT FILE NO DEBUGGER      ║")
	fmt.Println("╚══════════════════════════════════════╝")

	fmt.Println("\n📦 INVENTORY POTIONS:")
	hpCount, manaCount, rejuvCount := 0, 0, 0
	
	for _, itm := range d.Items.AllItems {
		if itm.Location != item.LocationInventory {
			continue
		}

		// Określ typ mikstury na podstawie nazwy
		potionType := "OTHER"
		if itm.IsHealingPotion() {
			potionType = "HP"
			hpCount++
		} else if itm.IsManaPotion() {
			potionType = "MANA"
			manaCount++
		} else if itm.IsRejuvPotion() {
			potionType = "REJUV"
			rejuvCount++
		} else {
			continue // Pomiń przedmioty niebędące miksturami
		}

		// Odczytaj txtFileNo bezpośrednio z pamięci
		itemAddress := pd.getItemAddress(itm.UnitID)
		if itemAddress == 0 {
			continue
		}
		
		itemDataBuffer := pd.gr.Process.ReadBytesFromMemory(itemAddress, 144)
		txtFileNo := memory.ReadUIntFromBuffer(itemDataBuffer, 0x04, memory.Uint32)

		fmt.Printf("  [%s] %s | TxtFileNo: %d | Grid(%d,%d)\n",
			potionType, itm.Name, txtFileNo, itm.Position.X, itm.Position.Y)
	}

	fmt.Println("\n🎒 BELT POTIONS:")
	for _, itm := range d.Items.Belt.Items {
		potionType := "OTHER"
		if itm.IsHealingPotion() {
			potionType = "HP"
		} else if itm.IsManaPotion() {
			potionType = "MANA"
		}

		itemAddress := pd.getItemAddress(itm.UnitID)
		if itemAddress == 0 {
			continue
		}
		
		itemDataBuffer := pd.gr.Process.ReadBytesFromMemory(itemAddress, 144)
		txtFileNo := memory.ReadUIntFromBuffer(itemDataBuffer, 0x04, memory.Uint32)

		fmt.Printf("  [%s] %s | TxtFileNo: %d | Slot(%d,%d)\n",
			potionType, itm.Name, txtFileNo, itm.Position.X, itm.Position.Y)
	}

	fmt.Println("\n📊 SUMMARY:")
	fmt.Printf("  HP Potions in inventory: %d\n", hpCount)
	fmt.Printf("  Mana Potions in inventory: %d\n", manaCount)
	fmt.Printf("  Rejuv Potions in inventory: %d\n", rejuvCount)
	
	fmt.Println("\n✅ Expected TxtFileNo ranges:")
	fmt.Println("  HP:    586-590 (Minor→Super)")
	fmt.Println("  Mana:  591-595 (Minor→Super)")
	fmt.Println("  Rejuv: 515-516 (Normal→Full)")
	fmt.Println("═══════════════════════════════════════\n")
}

func (pd *PotionDebugger) getItemAddress(unitID data.UnitID) uintptr {
	baseAddr := pd.gr.Process.ModuleBaseAddressPtr + pd.gr.Offset.UnitTable + (4 * 1024)
	unitTableBuffer := pd.gr.Process.ReadBytesFromMemory(baseAddr, 128*8)

	for i := 0; i < 128; i++ {
		itemOffset := 8 * i
		itemUnitPtr := uintptr(memory.ReadUIntFromBuffer(unitTableBuffer, uint(itemOffset), memory.Uint64))
		for itemUnitPtr > 0 {
			currentUnitID := pd.gr.Process.ReadUInt(itemUnitPtr+0x08, memory.Uint32)
			if data.UnitID(currentUnitID) == unitID {
				return itemUnitPtr
			}
			itemUnitPtr = uintptr(pd.gr.Process.ReadUInt(itemUnitPtr+0x150, memory.Uint64))
		}
	}
	return 0
}