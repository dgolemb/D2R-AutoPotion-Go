package data

import (
	"strings"

	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/item"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/stat"
)

// ✅ POPRAWIONE ID zgodnie z items.txt D2R
const (
	// Rejuvenation Potions (515-516)
	RejuvenationPotionID     = 515
	FullRejuvenationPotionID = 516
	
	// Healing Potions (586-590)
	MinorHealingPotionID   = 586
	LightHealingPotionID   = 587
	HealingPotionID        = 588
	GreaterHealingPotionID = 589
	SuperHealingPotionID   = 590
	
	// Mana Potions (591-595)
	MinorManaPotionID   = 591
	LightManaPotionID   = 592
	ManaPotionID        = 593
	GreaterManaPotionID = 594
	SuperManaPotionID   = 595
)

type Items struct {
	Belt     Belt
	AllItems []Item
}

func (i Items) Find(name item.Name, locations ...item.Location) (Item, bool) {
	for _, it := range i.AllItems {
		if strings.EqualFold(string(it.Name), string(name)) {
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
	TxtFileNo  uint // ✅ DODANE: ID z items.txt
}

func (i Item) Type() string {
	t, _ := item.TypeForItemName(string(i.Name))
	return t
}

func (i Item) IsPotion() bool {
	return i.IsHealingPotion() || i.IsManaPotion() || i.IsRejuvPotion()
}

// ✅ Rozpoznawanie po TxtFileNo - działa niezależnie od języka!
func (i Item) IsHealingPotion() bool {
	return i.TxtFileNo >= MinorHealingPotionID && i.TxtFileNo <= SuperHealingPotionID
}

func (i Item) IsManaPotion() bool {
	return i.TxtFileNo >= MinorManaPotionID && i.TxtFileNo <= SuperManaPotionID
}

func (i Item) IsRejuvPotion() bool {
	return i.TxtFileNo == RejuvenationPotionID || i.TxtFileNo == FullRejuvenationPotionID
}

func (i Item) IsFromQuest() bool {
	for _, q := range item.QuestItems {
		if strings.EqualFold(string(i.Name), q) {
			return true
		}
	}
	return false
}