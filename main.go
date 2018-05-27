package main

import (
	"encoding/binary"
	"image"
	"log"
	"os"
	"time"

	"github.com/MaxHalford/halfgone"
	"github.com/disintegration/imaging"
)

const maxChunkHeight = 3000

func isBlankLine(y int, img *image.Gray, bounds image.Rectangle) bool {

	for x := 0; x < bounds.Max.X; x++ {
		if img.GrayAt(x, y).Y == 0 {
			return false
		}
	}

	return true
}

func nextBreakPoint(offset int, img *image.Gray, bounds image.Rectangle, maxChunkHeight int) int {
	chunkHeight := bounds.Max.Y - offset
	if chunkHeight < maxChunkHeight {
		return bounds.Max.Y
	}

	maxConsecutiveBlankRows := 0
	breakpoint := offset + maxChunkHeight
	currentBlankRows := 0

	for y := 0; y < maxChunkHeight; y++ {
		if isBlankLine(y+offset, img, bounds) {
			currentBlankRows++
			if currentBlankRows >= maxConsecutiveBlankRows {
				maxConsecutiveBlankRows = currentBlankRows
				breakpoint = y + offset + 1
			}
		} else {
			currentBlankRows = 0
		}
	}

	return breakpoint
}

func main() {

	// Init
	buffer := make([]byte, 0, 10e3)
	buffer = append(buffer, 0x1b, 0x40, 0x1b, 0x33, 50)

	img, err := halfgone.LoadImage(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	img = imaging.Resize(img, 576, 0, imaging.Lanczos)
	gray := halfgone.ImageToGray(img)
	dithered := halfgone.FloydSteinbergDitherer{}.Apply(gray)

	bounds := dithered.Bounds()

	roundedX := (bounds.Max.X + (bounds.Max.X % 8)) / 8
	bytesWidth := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytesWidth, uint16(roundedX))

	offsetY := 0
	for offsetY < bounds.Max.Y {
		breakpoint := nextBreakPoint(offsetY, dithered, bounds, maxChunkHeight)
		if breakpoint-offsetY == 288 {
			breakpoint++
		}

		bytesHeight := make([]byte, 2)
		binary.LittleEndian.PutUint16(bytesHeight, uint16(breakpoint-offsetY))

		buffer = append(buffer, 0x1d, 0x76, 0x30, 0x00, bytesWidth[0], bytesWidth[1], bytesHeight[0], bytesHeight[1])

		for y := offsetY; y < breakpoint; y++ {
			for x := 0; x < roundedX; x++ {
				b := byte(0)

				for ix := 0; ix < 8; ix++ {
					var value byte

					if x*8+ix >= bounds.Max.X || y >= bounds.Max.Y {
						value = 0
					} else {
						pixel := dithered.GrayAt(x*8+ix, y)
						if pixel.Y == 0 {
							value = 1
						} else {
							value = 0
						}
					}

					b += value << uint(7-ix)
				}

				buffer = append(buffer, b)
			}
		}

		offsetY = breakpoint
	}

	// Cut & end
	buffer = append(buffer, 10, 0x1d, 0x56, 0x42, 0x30, 0xfa)

	file, _ := os.Create("/dev/usb/lp0")
	file.Write(buffer)
	time.Sleep(5 * time.Second)
}
