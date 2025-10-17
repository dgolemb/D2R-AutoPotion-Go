package data

import (
	"strings"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
)

const (
	HealingPotion      PotionType = "HealingPotion"
	ManaPotion         PotionType = "ManaPotion"
	RejuvenationPotion PotionType = "RejuvenationPotion"
)

type Belt struct {
	Items []Item
	Name  item.Name
}

func (b Belt) GetFirstPotion(potionType PotionType) (Position, bool) {
	for _, i := range b.Items {
		// Ensure potion is in row 0 and one of the four columns
		if strings.Contains(string(i.Name), string(potionType)) && i.Position.Y == 0 && (i.Position.X == 0 || i.Position.X == 1 || i.Position.X == 2 || i.Position.X == 3) {
			return i.Position, true
		}
	}

	return Position{}, false
}

// GetPotionCountInSlot zwraca ilość mikstur w danym slocie (kolumnie) paska
func (b Belt) GetPotionCountInSlot(slotX int) int {
	count := 0
	for _, i := range b.Items {
		if i.Position.X == slotX && i.Position.Y >= 0 {
			count++
		}
	}
	return count
}

// GetSlotsByPotionType zwraca listę slotów zawierających określony typ mikstury
func (b Belt) GetSlotsByPotionType(potionType PotionType) []int {
	slots := make(map[int]bool)
	for _, i := range b.Items {
		if strings.Contains(string(i.Name), string(potionType)) && i.Position.Y == 0 {
			slots[i.Position.X] = true
		}
	}

	result := make([]int, 0)
	for slot := range slots {
		result = append(result, slot)
	}
	return result
}

// NeedsRefill sprawdza czy slot potrzebuje uzupełnienia (mniej niż 2 mikstury)
func (b Belt) NeedsRefill(slotX int) bool {
	return b.GetPotionCountInSlot(slotX) < 2
}

func (b Belt) Rows() int {
	switch b.Name {
	case "":
		return 1
	case "Sash", "LightBelt":
		return 2
	case "Belt", "HeavyBelt":
		return 3
	default:
		return 4
	}
}

type PotionType string
