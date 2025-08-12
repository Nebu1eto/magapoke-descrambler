package main

import (
	"fmt"
	"image"
	"image/draw"
	"sort"
)

// NOTE: struct for representing shuffled tile
type ShuffledItem struct {
	RandomValue uint32
	Index       int
}

func generateShuffleMap(seed uint32, size int) []int {
	prng := NewXorShift32(seed)
	items := make([]ShuffledItem, size)
	for idx := range items {
		items[idx] = ShuffledItem{
			RandomValue: prng.Next(),
			Index:       idx,
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].RandomValue < items[j].RandomValue
	})

	shuffledIndices := make([]int, size)
	for idx, item := range items {
		shuffledIndices[idx] = item.Index
	}

	return shuffledIndices
}

// NOTE: it uses 4x4 grid, maybe it can be changed when original logic is changed
func DescrambleImage(scrambledImg image.Image, seed uint32) (image.Image, error) {
	const divisions = 4
	const gridSize = divisions * divisions

	bounds := scrambledImg.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	const y = 8
	tileWidth := (imgWidth / y / divisions) * y
	tileHeight := (imgHeight / y / divisions) * y

	if tileWidth == 0 || tileHeight == 0 {
		return nil, fmt.Errorf("err: image or tile size is invalid (w:%d, h:%d)", tileWidth, tileHeight)
	}

	// make new image (map[destIndex] = sourceIndex)
	destImg := image.NewRGBA(bounds)
	shuffledMap := generateShuffleMap(seed, gridSize)

	// reconstruct tile position
	for destIndex := 0; destIndex < gridSize; destIndex++ {
		sourceIndex := shuffledMap[destIndex]
		destGridX := destIndex % divisions
		destGridY := destIndex / divisions

		sourceGridX := sourceIndex % divisions
		sourceGridY := sourceIndex / divisions

		sourcePoint := image.Point{
			X: sourceGridX * tileWidth,
			Y: sourceGridY * tileHeight,
		}

		destRect := image.Rect(
			destGridX*tileWidth,
			destGridY*tileHeight,
			(destGridX+1)*tileWidth,
			(destGridY+1)*tileHeight,
		)

		draw.Draw(destImg, destRect, scrambledImg, sourcePoint, draw.Src)
	}

	return destImg, nil
}
