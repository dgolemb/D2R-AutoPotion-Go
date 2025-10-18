package data

import (
	"strings"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/stat"
)

type Items struct {
	Belt     Belt
	AllItems []Item
}

func (i Items) Find(name item.Name, locations ...item.Location) (Item, bool) {
	for _, it := range i.AllItems {
		if strings.EqualFold(string(it.Name), string(name)) {
			// If no locations are specified, return the first item found
			if len(locations) == 0 {
				return it, true
			}

			for _, l := range locations {
				if it.Location == l {
					return it, true
				}
			}
		}
	}

	return Item{}, false
}

func (i Items) ByLocation(locations ...item.Location) []Item {
	var items []Item

	for _, it := range i.AllItems {
		for _, l := range locations {
			if it.Location == l {
				items = append(items, it)
			}
		}
	}

	return items
}

type UnitID int

type Item struct {
	UnitID
	Name       item.Name
	Quality    item.Quality
	Position   Position
	Location   item.Location
	Ethereal   bool
	IsHovered  bool
	Stats      map[stat.ID]stat.Data
	Identified bool
}

func (i Item) Type() string {
	t, _ := item.TypeForItemName(string(i.Name))

	return t
}

func (i Item) IsPotion() bool {
	return i.IsHealingPotion() || i.IsManaPotion() || i.IsRejuvPotion()
}

// IsHealingPotion sprawdza czy przedmiot jest miksturą zdrowia
func (i Item) IsHealingPotion() bool {
	nameLower := strings.ToLower(string(i.Name))
	return strings.Contains(nameLower, "healingpotion") ||
		strings.Contains(nameLower, "healing potion") ||
		string(i.Name) == "MinorHealingPotion" ||
		string(i.Name) == "LightHealingPotion" ||
		string(i.Name) == "HealingPotion" ||
		string(i.Name) == "GreaterHealingPotion" ||
		string(i.Name) == "SuperHealingPotion"
}

// IsManaPotion sprawdza czy przedmiot jest miksturą many
func (i Item) IsManaPotion() bool {
	nameLower := strings.ToLower(string(i.Name))
	return strings.Contains(nameLower, "manapotion") ||
		strings.Contains(nameLower, "mana potion") ||
		string(i.Name) == "MinorManaPotion" ||
		string(i.Name) == "LightManaPotion" ||
		string(i.Name) == "ManaPotion" ||
		string(i.Name) == "GreaterManaPotion" ||
		string(i.Name) == "SuperManaPotion"
}

// IsRejuvPotion sprawdza czy przedmiot jest miksturą odnowy (rejuvenation)
func (i Item) IsRejuvPotion() bool {
	nameLower := strings.ToLower(string(i.Name))
	return strings.Contains(nameLower, "rejuv") ||
		strings.Contains(nameLower, "rejuvenation") ||
		string(i.Name) == "RejuvenationPotion" ||
		string(i.Name) == "FullRejuvenationPotion"
}

func (i Item) IsFromQuest() bool {
	for _, q := range item.QuestItems {
		if strings.EqualFold(string(i.Name), q) {
			return true
		}
	}

	return false
}

// GetPotionCountInSlot zlicza ilość mikstur w danym slocie paska (kolumnie)
func (b Belt) GetPotionCountInSlot(slot int) int {
	count := 0
	for _, item := range b.Items {
		// Position.X to numer slotu (kolumny) od 0 do 3
		if item.Position.X == slot {
			count++
		}
	}
	return count
}

// GetItemsInBeltSlot zwraca wszystkie przedmioty w danym slocie paska
func (b Belt) GetItemsInBeltSlot(slot int) []Item {
	var items []Item
	for _, item := range b.Items {
		if item.Position.X == slot {
			items = append(items, item)
		}
	}
	return items
}

// IsSlotFull sprawdza czy slot paska jest pełny (4 mikstury)
func (b Belt) IsSlotFull(slot int) bool {
	return b.GetPotionCountInSlot(slot) >= 4
}

// GetLowestRowInSlot zwraca najniższy (najmniejszy Y) rząd zajęty w slocie
// Przydatne do znajdowania "dna" stosu mikstur
func (b Belt) GetLowestRowInSlot(slot int) int {
	lowestRow := -1
	for _, item := range b.Items {
		if item.Position.X == slot {
			if lowestRow == -1 || item.Position.Y < lowestRow {
				lowestRow = item.Position.Y
			}
		}
	}
	return lowestRow
}

// GetHighestRowInSlot zwraca najwyższy (największy Y) rząd zajęty w slocie
func (b Belt) GetHighestRowInSlot(slot int) int {
	highestRow := -1
	for _, item := range b.Items {
		if item.Position.X == slot {
			if item.Position.Y > highestRow {
				highestRow = item.Position.Y
			}
		}
	}
	return highestRow
}