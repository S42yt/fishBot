package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
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
	IDLE_TIMEOUT = 3 * time.Minute
)

func main() {
	fmt.Println("--- Go Fishing Bot (v15 - Stabile UI-Erkennung) ---")

	roi, err := setupROI()
	if err != nil {
		log.Fatalf("Fehler bei der ROI-Einrichtung: %v", err)
	}

	fmt.Println("\n✅ Bereich erfolgreich definiert!")
	fmt.Println(">>> Bot startet JETZT. Drücke im Terminal STRG+C zum Beenden.")

	fmt.Println("🎣 Werfe Angel zum Start aus...")
	robotgo.Click("right")
	time.Sleep(2 * time.Second)

	lastCastTime := time.Now()
	currentState := IDLE

	fmt.Println("🔎 Warte auf Angel-Runde...")

	for {
		img, err := screenshot.CaptureRect(roi)
		if err != nil {
			log.Printf("Fehler beim Erstellen des Screenshots: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		
		redBox, redFound := findRedBoundingBox(img)

		
		
		
		isUIActive := redFound && redBox.Dx() > 8 && redBox.Dy() > 8

		if currentState == IDLE {
			if isUIActive {
				fmt.Println("✅ Angel-Runde erkannt! Starte aktives Angeln...")
				currentState = FISHING
			} else {
				if time.Since(lastCastTime) > IDLE_TIMEOUT {
					fmt.Println("⏰ 3-Minuten-Timeout! Nichts passiert. Werfe zur Sicherheit neu aus...")
					robotgo.Click("right")
					lastCastTime = time.Now()
					time.Sleep(2 * time.Second)
				}
			}
		} else if currentState == FISHING {
			if isUIActive {
				
				if analyzeImageForBite(img, redBox) {
					fmt.Println("🐠 Biss-Signal erkannt! Klicke einmal...")
					robotgo.Click("right")
					time.Sleep(500 * time.Millisecond)
				}
			} else {
				fmt.Println("🎣 Angel-Runde beendet. Werfe neu aus und warte...")
				robotgo.Click("right")
				lastCastTime = time.Now()
				currentState = IDLE
				time.Sleep(2 * time.Second)
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}


func analyzeImageForBite(img *image.RGBA, redBox image.Rectangle) bool {
	return isSurroundedByWhite(img, redBox)
}

func findRedBoundingBox(img *image.RGBA) (image.Rectangle, bool) {
	return findBoundingBox(img, isRed)
}

func findBoundingBox(img *image.RGBA, colorCheckFunc func(color.RGBA) bool) (image.Rectangle, bool) {
	minX, minY := -1, -1
	maxX, maxY := -1, -1
	found := false
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if colorCheckFunc(img.At(x, y).(color.RGBA)) {
				if !found {
					minX, minY, maxX, maxY = x, y, x, y
					found = true
				} else {
					minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
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
	const checkMargin = 2
	checkRect := image.Rect(box.Min.X-checkMargin, box.Min.Y-checkMargin, box.Max.X+checkMargin, box.Max.Y+checkMargin)
	var perimeterPixels, whitePerimeterPixels int
	for y := checkRect.Min.Y; y <= checkRect.Max.Y; y++ {
		for x := checkRect.Min.X; x <= checkRect.Max.X; x++ {
			if x > checkRect.Min.X && x < checkRect.Max.X && y > checkRect.Min.Y && y < checkRect.Max.Y {
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

func isRed(c color.RGBA) bool {
	return c.R >= RED_LOWER.R && c.R <= RED_UPPER.R && c.G >= RED_LOWER.G && c.G <= RED_UPPER.G && c.B >= RED_LOWER.B && c.B <= RED_UPPER.B
}

func isWhite(c color.RGBA) bool {
	return c.R >= WHITE_LOWER.R && c.R <= WHITE_UPPER.R && c.G >= WHITE_LOWER.G && c.G <= WHITE_UPPER.G && c.B >= WHITE_LOWER.B && c.B <= WHITE_UPPER.B
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
		return image.Rectangle{}, fmt.Errorf("der definierte Bereich hat keine Größe")
	}
	return rect, nil
}

