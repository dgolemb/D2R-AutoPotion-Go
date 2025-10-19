package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	Config *StructConfig
)

type StructConfig struct {
	Health struct {
		HealingPotionAt     int `yaml:"healingPotionAt"`
		ManaPotionAt        int `yaml:"manaPotionAt"`
		RejuvPotionAtLife   int `yaml:"rejuvPotionAtLife"`
		RejuvPotionAtMana   int `yaml:"rejuvPotionAtMana"`
		MercHealingPotionAt int `yaml:"mercHealingPotionAt"`
		MercRejuvPotionAt   int `yaml:"mercRejuvPotionAt"`
	} `yaml:"health"`
	Bindings struct {
		PotionHP    IntList `yaml:"potionHP"`
		PotionMANA  IntList `yaml:"potionMANA"`
		PotionREJUV IntList `yaml:"potionREJUV"`
	} `yaml:"bindings"`
	Timings struct {
		RejuvInterval       float32 `yaml:"rejuvInterval"`
		HealingInterval     float32 `yaml:"healingInterval"`
		ManaInterval        float32 `yaml:"manaInterval"`
		HealingMercInterval float32 `yaml:"healingMercInterval"`
	} `yaml:"timings"`
	BeltRefill struct {
		Enabled          bool `yaml:"enabled"`
		CheckInterval    int  `yaml:"checkInterval"`
		MinPotionsInSlot int  `yaml:"minPotionsInSlot"`
	} `yaml:"beltRefill"`
	// ✅ DODAJ TĘ SEKCJĘ:
	Debug struct {
		Enabled           bool `yaml:"enabled"`
		ShowMemoryDetails bool `yaml:"showMemoryDetails"`
		ShowGameData      bool `yaml:"showGameData"`
	} `yaml:"debug"`
}

type IntList []int

func (il *IntList) UnmarshalYAML(value *yaml.Node) error {
	var single int
	if err := value.Decode(&single); err == nil {
		*il = IntList{single}
		return nil
	}

	var list []int
	if err := value.Decode(&list); err == nil {
		*il = IntList(list)
		return nil
	}

	return fmt.Errorf("unable to unmarshal IntList from YAML node kind %d", value.Kind)
}

func Load() error {
	r, err := os.Open("config/config.yaml")
	if err != nil {
		return fmt.Errorf("error loading config.yaml: %w", err)
	}

	d := yaml.NewDecoder(r)
	if err = d.Decode(&Config); err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}
	return nil
}