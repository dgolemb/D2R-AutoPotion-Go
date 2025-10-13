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
	Gr *memory.GameReader
}

type Manager struct {
	lastRejuv     time.Time
	lastRejuvMerc time.Time
	lastHeal      time.Time
	lastMana      time.Time
	lastMercHeal  time.Time
	lastDebugMsg  time.Time
	Timer         time.Time
	// indexes for cycling through multiple binding keys
	hpIndex    int
	manaIndex  int
	rejuvIndex int
	mercHpIndex int
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
	return &Watcher{Gr: gr}
}

func (w *Watcher) Start(ctx context.Context, manager *Manager, XP *ExperienceCalc, audioBufferL *beep.Buffer, audioBufferM *beep.Buffer, audioBufferR *beep.Buffer) error {

	d, err := w.Gr.GetData()
	if err != nil {
		fmt.Printf("\r                                              ") //clean line
		fmt.Printf("\rnot In Game\n")
		fmt.Print("\033[A")
		time.Sleep(1 * time.Second)
	}
	if err == nil {
		if XP.FirstStart {
			XP.XPbefore = d.PlayerUnit.Stats[stat.Experience]
			XP.FirstStart = false
		}
	}

	select {
	case <-ctx.Done():
		return nil
	default:
		d, err = w.Gr.GetData()
		if err != nil {
			fmt.Printf("\r                                  ") //clean line
			fmt.Printf("\rnot In Game\n")
			fmt.Print("\033[A")
			time.Sleep(1 * time.Second)
		}

		if err == nil {
			if time.Since(manager.lastDebugMsg) > (time.Second * 2) {
				if XP.XPbefore == 0 {
					XP.XPbefore = d.PlayerUnit.Stats[stat.Experience]
				}
				fmt.Printf("\r%2.0f PercentLife:%*d PercentMana:%*d", time.Since(manager.Timer).Seconds(), 3, d.PlayerUnit.HPPercent(), 3, d.PlayerUnit.MPPercent())
				manager.lastDebugMsg = time.Now()

				if time.Since(manager.Timer) > (time.Second * 15) {
					manager.Timer = time.Now()
					if XP.first30s {
						for i := 0; i < len(XP.XP); i++ {
							XP.XP[i] = d.PlayerUnit.Stats[stat.Experience] - XP.XPbefore
						}
						XP.first30s = false
					}
					diff := d.PlayerUnit.Stats[stat.Experience] - XP.XPbefore

					XP.XP_aux = [25]int{diff, XP.XP[0], XP.XP[1], XP.XP[2], XP.XP[3], XP.XP[4], XP.XP[5], XP.XP[6], XP.XP[7], XP.XP[8], XP.XP[9], XP.XP[10], XP.XP[11], XP.XP[12], XP.XP[13], XP.XP[14], XP.XP[15], XP.XP[16], XP.XP[17], XP.XP[18], XP.XP[19], XP.XP[20], XP.XP[21], XP.XP[22], XP.XP[23]}
					XP.XP = XP.XP_aux

					for i := 0; i < len(XP.XParray); i++ {
						XP.XParray[i] = 0
						for j := 0; j < i; j++ {
							XP.XParray[i] += float64((XP.XP[j] / i)) / 100000
						}
						if (i%2) > 0 && i < 7 {
							fmt.Printf(" xp_%d:%3.2fM", i, XP.XParray[i]*4)
						}
					}
					fmt.Printf(" xp_%d:%3.2fM", XP.IndexUpdated, XP.XParray[XP.IndexUpdated]*4)
					XP.XPbefore = d.PlayerUnit.Stats[stat.Experience]
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
					usedRejuv := true
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

				// Mercenary
				if d.MercHPPercent() > 0 {
					usedMercRejuv := false
					if time.Since(manager.lastRejuvMerc) > (time.Duration(config.Config.Timings.RejuvInterval)*time.Second) && d.MercHPPercent() <= config.Config.Health.MercRejuvPotionAt {
						UseMercRejuv(manager)
						usedMercRejuv := true
						if usedMercRejuv {
							manager.lastRejuvMerc = time.Now()
						}
						speaker.Play(audioBufferR.Streamer(0, audioBufferR.Len()))
					}

					if !usedMercRejuv {

						if d.MercHPPercent() <= config.Config.Health.MercHealingPotionAt && time.Since(manager.lastMercHeal) > (time.Duration(config.Config.Timings.HealingMercInterval)*time.Second) {
							UseHPMerc(manager)
							manager.lastMercHeal = time.Now()
							speaker.Play(audioBufferL.Streamer(0, audioBufferL.Len()))
						}
					}
				}
			}
		}
		return err
	}

}

func ResetXPCalc(XP *ExperienceCalc) {
	for i := range XP.XP {
		XP.XP[i] = 0
	}
	for j := range XP.XParray {
		XP.XParray[j] = 0
	}
	for k := range XP.XP {
		XP.XP[k] = 0
	}
	XP.XPbefore = 0
	XP.IndexUpdated = 0
	XP.first30s = true
	XP.FirstStart = true
	XP.Minutes = 0
	XP.Hours = 0
	f, err := os.Create("data.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err2 := f.WriteString("  0.00")
	if err2 != nil {
		log.Fatal(err2)
	}
}

func InitAudio(path string) (*beep.Buffer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		return nil, err
	}
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	streamer.Close()

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
	}
	kb.HasSHIFT(false)
	// get current key from list and advance index
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
