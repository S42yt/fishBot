package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	hook "github.com/robotn/gohook"
)

type BotState int

const (
	IDLE    BotState = 1
	FISHING BotState = 2
)

var (
	RED_LOWER   = color.RGBA{R: 200, G: 0, B: 0, A: 255}
	RED_UPPER   = color.RGBA{R: 255, G: 100, B: 100, A: 255}
	WHITE_LOWER = color.RGBA{R: 220, G: 220, B: 220, A: 255}
	WHITE_UPPER = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

const (
	IDLE_TIMEOUT      = 3 * time.Minute
	TICK_INTERVAL     = 25 * time.Millisecond
	BITE_CHECK_MARGIN = 2
)

type Bot struct {
	roi          image.Rectangle
	state        BotState
	lastCastTime time.Time
	isPaused     bool
	mu           sync.Mutex
}

func NewBot(roi image.Rectangle) *Bot {
	return &Bot{
		roi:      roi,
		state:    IDLE,
		isPaused: false,
	}
}

func (b *Bot) Run() {
	fmt.Println("🎣 Werfe Angel zum Start aus...")
	robotgo.Click("right")
	time.Sleep(2 * time.Second)
	b.lastCastTime = time.Now()

	fmt.Println("🔎 Warte auf Angel-Runde...")

	for {
		b.Tick()
		time.Sleep(TICK_INTERVAL)
	}
}

func (b *Bot) Tick() {

	b.mu.Lock()
	paused := b.isPaused
	b.mu.Unlock()

	if paused {
		return
	}

	img, err := screenshot.CaptureRect(b.roi)
	if err != nil {
		log.Printf("Fehler beim Erstellen des Screenshots: %v", err)
		time.Sleep(1 * time.Second)
		return
	}

	redBox, redFound := findBoundingBox(img, isRed)

	isUIActive := redFound && redBox.Dx() > 8 && redBox.Dy() > 8

	switch b.state {
	case IDLE:
		if isUIActive {
			fmt.Println("\n✅ Angel-Runde erkannt! Starte aktives Angeln...")
			b.state = FISHING
		} else if time.Since(b.lastCastTime) > IDLE_TIMEOUT {
			fmt.Println("\n⏰ Timeout! Nichts passiert. Werfe zur Sicherheit neu aus...")
			robotgo.Click("right")
			b.lastCastTime = time.Now()
			time.Sleep(2 * time.Second)
		}
	case FISHING:
		if isUIActive {

			if analyzeImageForBite(img, redBox) {
				fmt.Println("🐠 Biss-Signal erkannt! Klicke...")
				robotgo.Click("right")
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			fmt.Println("🎣 Angel-Rude beendet. Werfe neu aus und warte...")
			robotgo.Click("right")
			b.lastCastTime = time.Now()
			b.state = IDLE
			time.Sleep(2 * time.Second)
		}
	}
}

func (b *Bot) listenForPauseToggle() {
	fmt.Println("\n▶️  Globale Hotkey-Überwachung für Taste 'P' wird gestartet...")

	hook.Register(hook.KeyDown, []string{"p"}, func(e hook.Event) {
		b.mu.Lock()
		b.isPaused = !b.isPaused
		if b.isPaused {

			fmt.Print("\r⏸️  Bot pausiert. Drücke 'P' zum Fortsetzen.                     ")
		} else {
			fmt.Print("\r▶️  Bot wird fortgesetzt...                                    ")
		}
		b.mu.Unlock()
	})

	s := hook.Start()

	defer hook.End()

	<-hook.Process(s)
}

func analyzeImageForBite(img *image.RGBA, redBox image.Rectangle) bool {
	return isSurroundedByWhite(img, redBox)
}

func findBoundingBox(img *image.RGBA, colorCheckFunc func(color.RGBA) bool) (image.Rectangle, bool) {
	minX, minY, maxX, maxY := -1, -1, -1, -1
	found := false
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if colorCheckFunc(img.At(x, y).(color.RGBA)) {
				if !found {
					minX, minY, maxX, maxY = x, y, x, y
					found = true
				} else {
					minX = min(minX, x)
					minY = min(minY, y)
					maxX = max(maxX, x)
					maxY = max(maxY, y)
				}
			}
		}
	}
	if !found {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX, maxY), true
}

func isSurroundedByWhite(img *image.RGBA, box image.Rectangle) bool {
	checkRect := image.Rect(box.Min.X-BITE_CHECK_MARGIN, box.Min.Y-BITE_CHECK_MARGIN, box.Max.X+BITE_CHECK_MARGIN, box.Max.Y+BITE_CHECK_MARGIN)
	var perimeterPixels, whitePerimeterPixels int

	for y := checkRect.Min.Y; y <= checkRect.Max.Y; y++ {
		for x := checkRect.Min.X; x <= checkRect.Max.X; x++ {
			if x > box.Min.X && x < box.Max.X && y > box.Min.Y && y < box.Max.Y {
				continue
			}
			if !image.Pt(x, y).In(img.Bounds()) {
				continue
			}

			perimeterPixels++
			if isWhite(img.At(x, y).(color.RGBA)) {
				whitePerimeterPixels++
			}
		}
	}

	if perimeterPixels == 0 {
		return false
	}

	return float64(whitePerimeterPixels)/float64(perimeterPixels) > 0.5
}

func isColorInRange(c, lower, upper color.RGBA) bool {
	return c.R >= lower.R && c.R <= upper.R &&
		c.G >= lower.G && c.G <= upper.G &&
		c.B >= lower.B && c.B <= upper.B
}

func isRed(c color.RGBA) bool {
	return isColorInRange(c, RED_LOWER, RED_UPPER)
}

func isWhite(c color.RGBA) bool {
	return isColorInRange(c, WHITE_LOWER, WHITE_UPPER)
}

func setupROI() (image.Rectangle, error) {
	fmt.Println("\n--- Bereich für Angel-Leiste definieren ---")
	fmt.Print("Positioniere deine Maus an der OBEREN-LINKEN Ecke der Leiste und drücke Enter...")
	fmt.Scanln()
	x1, y1 := robotgo.GetMousePos()
	fmt.Printf("Obere-linke Ecke gespeichert: (%d, %d)\n", x1, y1)

	fmt.Print("Positioniere deine Maus nun an der UNTEREN-RECHTEN Ecke der Leiste und drücke Enter...")
	fmt.Scanln()
	x2, y2 := robotgo.GetMousePos()
	fmt.Printf("Untere-rechte Ecke gespeichert: (%d, %d)\n", x2, y2)

	rect := image.Rect(min(x1, x2), min(y1, y2), max(x1, x2), max(y1, y2))
	if rect.Dx() == 0 || rect.Dy() == 0 {
		return image.Rectangle{}, fmt.Errorf("der definierte Bereich hat keine Größe (Breite oder Höhe ist 0)")
	}
	return rect, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	fmt.Println("--- Go Fishing Bot (v17 - Schnell & Globaler Hotkey) ---")

	roi, err := setupROI()
	if err != nil {
		log.Fatalf("Fehler bei der ROI-Einrichtung: %v", err)
	}
	fmt.Println("\n✅ Bereich erfolgreich definiert!")

	bot := NewBot(roi)

	go bot.listenForPauseToggle()

	fmt.Println(">>> Bot startet JETZT. Drücke 'P' zum Pausieren und im Terminal STRG+C zum Beenden.")

	bot.Run()
}
