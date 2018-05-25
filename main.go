package main

import (
	"encoding/binary"
	"log"
	"os"

	"github.com/MaxHalford/halfgone"
	"github.com/disintegration/imaging"
)

const maxChunkHeight = 400

func main() {

	// Init
	buffer := make([]byte, 0, 10e3)
	buffer = append(buffer, 0x1b, 0x40)

	img, err := halfgone.LoadImage(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	if img.Bounds().Max.X >= 576 {
		img = imaging.Resize(img, 576, 0, imaging.Lanczos)
	}

	gray := halfgone.ImageToGray(img)
	dithered := halfgone.FloydSteinbergDitherer{}.Apply(gray)

	bounds := dithered.Bounds()
	roundedX := (bounds.Max.X + (bounds.Max.X % 8)) / 8
	bytesWidth := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytesWidth, uint16(roundedX))

	offsetY := 0
	for offsetY < bounds.Max.Y {
		chunkHeight := bounds.Max.Y - offsetY
		if chunkHeight > maxChunkHeight {
			chunkHeight = maxChunkHeight
		}

		bytesHeight := make([]byte, 2)
		binary.LittleEndian.PutUint16(bytesHeight, uint16(chunkHeight))

		buffer = append(buffer, 0x1d, 0x76, 0x30, 0x00, bytesWidth[0], bytesWidth[1], bytesHeight[0], bytesHeight[1])

		for y := 0; y < chunkHeight; y++ {
			for x := 0; x < roundedX; x++ {
				b := byte(0)

				for ix := 0; ix < 8; ix++ {
					var value byte

					if x*8+ix > bounds.Max.X {
						value = 0
					} else {
						pixel := dithered.GrayAt(x*8+ix, offsetY+y)
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

		// buffer = append(buffer, 10)
		offsetY += chunkHeight
	}

	// Cut & end
	buffer = append(buffer, 10, 0x1d, 0x56, 0x42, 0x30, 0xfa)

	os.Stdout.Write(buffer)
}
