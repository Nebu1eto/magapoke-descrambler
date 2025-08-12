package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"sync"
)

// NOTE: represent the JSON schema (From /web/episode/viewer API response)
type InputData struct {
	ScrambleSeed uint32   `json:"scramble_seed"`
	PageList     []string `json:"page_list"`
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
			descrambledImg, err := DescrambleImage(scrambledImg, data.ScrambleSeed)
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
