package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
)

// JSON file schema
type InputData struct {
	ScrambleSeed uint32   `json:"scramble_seed"`
	PageList     []string `json:"page_list"`
}

// Xorshift32 PRNG
type XorShift32 struct {
	state uint32
}

func NewXorShift32(seed uint32) *XorShift32 {
	if seed == 0 {
		seed = 1
	}
	return &XorShift32{state: seed}
}

func (r *XorShift32) Next() uint32 {
	x := r.state
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	r.state = x
	return x
}

// struct for shuffled tile's index
type ShuffledItem struct {
	RandomValue uint32
	Index       int
}

func generateShuffleMap(seed uint32, size int) []int {
	prng := NewXorShift32(seed)
	items := make([]ShuffledItem, size)
	for i := 0; i < size; i++ {
		items[i] = ShuffledItem{
			RandomValue: prng.Next(),
			Index:       i,
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].RandomValue < items[j].RandomValue
	})

	shuffledIndices := make([]int, size)
	for i, item := range items {
		shuffledIndices[i] = item.Index
	}

	return shuffledIndices
}

func descrambleImage(scrambledImg image.Image, seed uint32) (image.Image, error) {
	// NOTE: it uses 4x4 grid, maybe it can be changed when original logic is changed
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

// TODO: support custom headers and mimic browser's fingerprint
func downloadImage(url string) (image.Image, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("err: failure to download image (%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("err: http status is invalid (%s)", resp.Status)
	}

	img, format, err := image.Decode(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("err: failure to decode image (%s): %w", url, err)
	}
	return img, format, nil
}

func saveImage(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("err: failure to save image (%s): %w", path, err)
	}
	defer file.Close()
	return jpeg.Encode(file, img, &jpeg.Options{Quality: 95})
}

func main() {
	log.Println("Start Descrambler")
	if len(os.Args) < 2 {
		log.Fatalf("Example: ./descrambler <json_file_path>")
	}

	jsonPath := os.Args[1]
	file, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("err: failure to read JSON file (%v)", err)
	}

	var data InputData
	if err := json.Unmarshal(file, &data); err != nil {
		log.Fatalf("err: failure to parse JSON (%v)", err)
	}

	log.Printf("Scramble Seed: %d", data.ScrambleSeed)
	log.Printf("Processing %d images", len(data.PageList))

	var wg sync.WaitGroup

	for i, pageURL := range data.PageList {
		wg.Add(1)
		go func(index int, url string) {
			defer wg.Done()

			log.Printf("[%3d] downloading...", index+1)
			scrambledImg, _, err := downloadImage(url)
			if err != nil {
				log.Printf("[%3d] failure to download - %v", index+1, err)
				return
			}

			log.Printf("[%3d] descrambling...", index+1)
			descrambledImg, err := descrambleImage(scrambledImg, data.ScrambleSeed)
			if err != nil {
				log.Printf("[%3d] failure to descramble - %v", index+1, err)
				return
			}

			outputPath := fmt.Sprintf("out_%03d.jpg", index+1)
			if err := saveImage(descrambledImg, outputPath); err != nil {
				log.Printf("[%3d] failure to save - %v", index+1, err)
				return
			}
			log.Printf("[%3d] saved image to %s", index+1, outputPath)

		}(i, pageURL)
	}

	wg.Wait()
	log.Println("Completed to Download and Descramble Images")
}
