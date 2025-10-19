package lifewatcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Hefero/D2R-AutoPotion-Go/cmd/config"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/stat"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/data/state"
	"github.com/Hefero/D2R-AutoPotion-Go/pkg/memory"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/micmonay/keybd_event"
)

type Watcher struct {
	Gr           *memory.GameReader
	BeltRefiller *BeltRefiller
}

type Manager struct {
	lastRejuv      time.Time
	lastRejuvMerc  time.Time
	lastHeal       time.Time
	lastMana       time.Time
	lastMercHeal   time.Time
	lastDebugMsg   time.Time
	Timer          time.Time
	hpIndex        int
	manaIndex      int
	rejuvIndex     int
	mercHpIndex    int
	mercRejuvIndex int
}

type ExperienceCalc struct {
	XP           [25]int
	XP_aux       [25]int
	XParray      [25]float64
	XPbefore     int     `default:"0"`
	IndexUpdated int     `default:"0"`
	first30s     bool    `default:"true"`
	FirstStart   bool    `default:"true"`
	Minutes      float64 `default:"0"`
	Hours        float64 `default:"0"`
}

func NewWatcher(gr *memory.GameReader) *Watcher {
	refiller := NewBeltRefiller(gr)
	refiller.SetDebugMode(config.Config.Debug.Enabled)
	
	return &Watcher{
		Gr:           gr,
		BeltRefiller: refiller,
	}
}

func (w *Watcher) Start(ctx context.Context, manager *Manager, XP *ExperienceCalc, audioBufferL *beep.Buffer, audioBufferM *beep.Buffer, audioBufferR *beep.Buffer) error {

	if config.Config.Debug.ShowGameData {
		fmt.Printf("[DEBUG] Attempting to get game data...\n")
	}
	
	d, err := w.Gr.GetData()
	if err != nil {
		fmt.Printf("\r                                              ")
		fmt.Printf("\rnot In Game\n")
		fmt.Print("\033[A")
		time.Sleep(1 * time.Second)
		return err
	}
	
	if config.Config.Debug.ShowGameData {
		fmt.Printf("[DEBUG] Successfully got game data! PlayerUnit area: %d\n", d.PlayerUnit.Area)
	}

	// Sprawdzanie i uzupełnianie paska (tylko poza miastem)
	if !d.PlayerUnit.Area.IsTown() {
		if refillErr := w.BeltRefiller.CheckAndRefillBelt(); refillErr != nil {
			if config.Config.Debug.Enabled {
				fmt.Printf("Belt refill error: %v\n", refillErr)
			}
		}
	}

	if time.Since(manager.lastDebugMsg) > (time.Second * 2) {
		fmt.Printf("\r                                                                         ")
		fmt.Printf("\r%2d PercentLife: %2d PercentMana:%3d", d.PlayerUnit.Stats[stat.Level], d.PlayerUnit.HPPercent(), d.PlayerUnit.MPPercent())
		manager.lastDebugMsg = time.Now()
	}

	// XP tracking
	if d.PlayerUnit.Stats[stat.Level] < 99 {
		XP.XP_aux[XP.IndexUpdated] = d.PlayerUnit.Stats[stat.Experience]

		if time.Since(manager.Timer) > (time.Second*30) && XP.first30s {
			XP.first30s = false
			XP.XPbefore = d.PlayerUnit.Stats[stat.Experience]
		}

		if time.Since(manager.Timer) > (time.Minute*1) && !XP.FirstStart {
			if XP.IndexUpdated != 0 {
				XP.XParray[XP.IndexUpdated] = (float64(XP.XP[XP.IndexUpdated]) - float64(XP.XP[XP.IndexUpdated-1])) / (float64(60))
			} else {
				XP.XParray[XP.IndexUpdated] = (float64(XP.XP[XP.IndexUpdated]) - float64(XP.XP[19])) / (float64(60))
			}

			if XP.IndexUpdated == 19 {
				XP.IndexUpdated = 0
			}

			XP.XP[XP.IndexUpdated] = d.PlayerUnit.Stats[stat.Experience]

			XPneeded := levelXP(d.PlayerUnit.Stats[stat.Level]+1) - XP.XPbefore
			XPfactor := XP.XParray[XP.IndexUpdated]
			if XPfactor == 0 {
				XPfactor = 1
			}
			XP.Minutes = float64(XPneeded) / float64((XP.XParray[XP.IndexUpdated] * 4 * 100000))
			XP.Hours = float64(XPneeded) / float64((XP.XParray[XP.IndexUpdated] * 4 * 100000 * 60))

			f, err := os.Create("data.txt")
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			duration := time.Duration(time.Duration(XP.Minutes) * time.Minute).Round(time.Minute).String()
			durationTrim := duration[:len(duration)-2]
			stringWrite := durationTrim + "  " + strconv.FormatFloat(XP.XParray[XP.IndexUpdated], 'f', 2, 64)
			_, err2 := f.WriteString(stringWrite)

			if err2 != nil {
				log.Fatal(err2)
			}

			fmt.Printf(" tnl:%s", stringWrite)
			if XP.IndexUpdated < 19 {
				XP.IndexUpdated++
			}
		}
		fmt.Print("\n\033[A")
	}

	if !d.PlayerUnit.Area.IsTown() {
		var healingInterval float32 = config.Config.Timings.HealingInterval

		if d.PlayerUnit.States.HasState(state.Poison) {
			healingInterval += 2
		}

		usedRejuv := false
		if time.Since(manager.lastRejuv) > (time.Duration(config.Config.Timings.RejuvInterval)*time.Second) && (d.PlayerUnit.HPPercent() <= config.Config.Health.RejuvPotionAtLife || d.PlayerUnit.MPPercent() < config.Config.Health.RejuvPotionAtMana) {
			UseRejuv(manager)
			usedRejuv = true
			if usedRejuv {
				manager.lastRejuv = time.Now()
			}
			speaker.Play(audioBufferR.Streamer(0, audioBufferR.Len()))
		}

		if !usedRejuv {
			if d.PlayerUnit.HPPercent() <= config.Config.Health.HealingPotionAt && time.Since(manager.lastHeal) > (time.Duration(healingInterval)*time.Second) {
				UseHP(manager)
				manager.lastHeal = time.Now()
				speaker.Play(audioBufferL.Streamer(0, audioBufferL.Len()))
			}

			if d.PlayerUnit.MPPercent() <= config.Config.Health.ManaPotionAt && time.Since(manager.lastMana) > (time.Duration(config.Config.Timings.ManaInterval)*time.Second) {
				UseMana(manager)
				manager.lastMana = time.Now()
				speaker.Play(audioBufferM.Streamer(0, audioBufferM.Len()))
			}
		}

		// Merc healing - używamy d.MercHPPercent()
		mercHPPercent := d.MercHPPercent()
		
		if mercHPPercent > 0 && mercHPPercent <= config.Config.Health.MercHealingPotionAt && time.Since(manager.lastMercHeal) > (time.Duration(config.Config.Timings.HealingMercInterval)*time.Second) {
			UseHPMerc(manager)
			manager.lastMercHeal = time.Now()
			speaker.Play(audioBufferL.Streamer(0, audioBufferL.Len()))
		}

		if mercHPPercent > 0 && mercHPPercent <= config.Config.Health.MercRejuvPotionAt && time.Since(manager.lastRejuvMerc) > (time.Duration(config.Config.Timings.RejuvInterval)*time.Second) {
			UseMercRejuv(manager)
			manager.lastRejuvMerc = time.Now()
			speaker.Play(audioBufferR.Streamer(0, audioBufferR.Len()))
		}
	}

	if XP.FirstStart && time.Since(manager.Timer) > (time.Second*30) {
		XP.FirstStart = false
	}

	return nil
}

func InitAudio(filename string) (*beep.Buffer, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		return nil, err
	}
	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)

	return buffer, nil
}

func getKey(key int) int {
	switch key {
	case 1:
		return keybd_event.VK_1
	case 2:
		return keybd_event.VK_2
	case 3:
		return keybd_event.VK_3
	case 4:
		return keybd_event.VK_4
	default:
		return 0
	}
}

func UseHP(m *Manager) {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return
	}
	kb.HasSHIFT(false)
	keys := config.Config.Bindings.PotionHP
	if len(keys) == 0 {
		return
	}
	key := keys[m.hpIndex%len(keys)]
	kb.SetKeys(getKey(key))
	err = kb.Launching()
	m.hpIndex = (m.hpIndex + 1) % len(keys)
}

func UseMana(m *Manager) {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return
	}
	kb.HasSHIFT(false)
	keys := config.Config.Bindings.PotionMANA
	if len(keys) == 0 {
		return
	}
	key := keys[m.manaIndex%len(keys)]
	kb.SetKeys(getKey(key))
	err = kb.Launching()
	m.manaIndex = (m.manaIndex + 1) % len(keys)
}

func UseHPMerc(m *Manager) {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return
	}
	kb.HasSHIFT(true)
	keys := config.Config.Bindings.PotionHP
	if len(keys) == 0 {
		return
	}
	key := keys[m.mercHpIndex%len(keys)]
	kb.SetKeys(getKey(key))
	err = kb.Launching()
	kb.HasSHIFT(false)
	m.mercHpIndex = (m.mercHpIndex + 1) % len(keys)
}

func UseMercRejuv(m *Manager) {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return
	}
	kb.HasSHIFT(true)
	keys := config.Config.Bindings.PotionREJUV
	if len(keys) == 0 {
		return
	}
	key := keys[m.mercRejuvIndex%len(keys)]
	kb.SetKeys(getKey(key))
	err = kb.Launching()
	kb.HasSHIFT(false)
	m.mercRejuvIndex = (m.mercRejuvIndex + 1) % len(keys)
}

func UseRejuv(m *Manager) {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return
	}
	kb.HasSHIFT(false)
	keys := config.Config.Bindings.PotionREJUV
	if len(keys) == 0 {
		return
	}
	key := keys[m.rejuvIndex%len(keys)]
	kb.SetKeys(getKey(key))
	err = kb.Launching()
	m.rejuvIndex = (m.rejuvIndex + 1) % len(keys)
}

func ResetXPCalc(XP *ExperienceCalc) {
	*XP = ExperienceCalc{}
}

// levelXP zwraca wymagane XP dla osiągnięcia danego poziomu
func levelXP(lvl int) int {
	xpTable := map[int]int{
		70: 285041630, 71: 311105466, 72: 339515048, 73: 370481492, 74: 404234916,
		75: 441026148, 76: 481128591, 77: 524840254, 78: 572485967, 79: 624419793,
		80: 681027665, 81: 742730244, 82: 809986056, 83: 883294891, 84: 963201521,
		85: 1050299747, 86: 1145236814, 87: 1248718217, 88: 1361512946, 89: 1484459201,
		90: 1618470619, 91: 1764543065, 92: 1923762030, 93: 2097310703, 94: 2286478756,
		95: 2492671933, 96: 2717422497, 97: 2962400612, 98: 3229426756, 99: 3520485254,
	}
	
	if xp, found := xpTable[lvl]; found {
		return xp
	}
	return 0
}