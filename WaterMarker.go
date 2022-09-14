package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/nfnt/resize"
)

var (
	paramOpacity   *int     = flag.Int("opacity", 70, "Watermark opacity between 0 and 100")
	paramLocation  *string  = flag.String("location", "right", "Location of watermark [left, right]")
	paramScale     *float64 = flag.Float64("scale", 0.2, "Specify the size of the watermark as a portion of the image (between 0 and 1)")
	paramWatermark *string  = flag.String("watermark", "watermark.png", "Name of PNG image to be used as watermark")
	paramSourceDir *string  = flag.String("source", "photos", "Source directory (location to find un-watermarked photos)")
	paramTargetDir *string  = flag.String("target", "watermarked", "Target directory (location to put watermarked photos")
	paramForce     *bool    = flag.Bool("force", false, "Force overwrite of target directory if it already exists")
)

func main() {
	flag.Parse()
	fmt.Println("**************************************************************************")
	fmt.Println("*                                                                        *")
	fmt.Println("*      WaterMarker v1.0 - Written by Tjeerd Bakker (ICheered) in Go      *")
	fmt.Println("*                                                                        *")
	fmt.Println("**************************************************************************")
	fmt.Println("")
	fmt.Println("For help: run the program from command line with the -h flag")
	fmt.Println("Having issues? Please let me know at Tjeerd992@gmail.com")
	fmt.Println("")

	fmt.Println("Using following parameters:")
	fmt.Printf("- Opacity:          %d\n", *paramOpacity)
	fmt.Printf("- Location:         %s\n", *paramLocation)
	fmt.Printf("- Scale:            %1.1f\n", *paramScale)
	fmt.Printf("- Watermark:        %s\n", *paramWatermark)
	fmt.Printf("- Source directory: %s\n", *paramSourceDir)
	fmt.Printf("- Target directory: %s\n", *paramTargetDir)

	if _, err := os.Stat(*paramWatermark); errors.Is(err, os.ErrNotExist) {
		// Watermark file does not exist
		log.Fatalf("ERROR: Watermark file '%s' does not exist in this directory\n", *paramWatermark)
	}

	if !strings.HasSuffix(*paramWatermark, ".png") {
		// Watermark is not a PNG
		log.Fatalf("ERROR: Watermark file '%s' is not a PNG file\n", *paramWatermark)
	}

	if _, err := os.Stat(*paramSourceDir); os.IsNotExist(err) {
		// Source folder does not exist
		log.Fatalf("ERROR: Source folder (folder containing images) '%s' does not exist in this directory\n", *paramSourceDir)
	}

	if _, err := os.Stat(*paramTargetDir); err == nil {
		// Target dir already exists
		fmt.Printf("WARNING: Target folder '%s' already exists in this directory. \n", *paramTargetDir)
		if *paramForce {
			fmt.Println("         Using --force, so will overwrite existing files")
		} else {
			fmt.Println("         Use --force to overwrite existing files")
			fmt.Println("         Exiting to avoid overwriting existing files.")
			os.Exit(1)
		}
	} else {
		os.Mkdir(*paramTargetDir, 0755)
	}
	fmt.Print("\n--------------------------------------\n")

	watermark := openImage(*paramWatermark, "png")
	mask := image.NewUniform(color.Alpha{uint8(*paramOpacity * 255)})
	files := getFiles(*paramSourceDir)

	fmt.Printf("Starting: Processing %d files\n\n", len(files))

	var wg sync.WaitGroup
	wg.Add(len(files))
	start := time.Now()
	for _, file := range files {
		go func(file os.FileInfo, watermark image.Image, mask image.Image, watermarkLocation string, watermarkScale float64, sourceDir string, targetDir string) {
			defer wg.Done()
			if !(strings.HasSuffix(file.Name(), ".jpg")) && !(strings.HasSuffix(file.Name(), ".jpeg")) {
				fmt.Printf("Skipping photo '%s' because it is not a .jpg or .jpeg\n", file.Name())
				return
			}

			srcImage := openImage(path.Join(sourceDir, file.Name()), "jpeg")

			imgSize := srcImage.Bounds()

			scaledWatermark := resize.Resize(0, uint(watermarkScale*float64(imgSize.Dy())), watermark, resize.NearestNeighbor)

			wmSize := scaledWatermark.Bounds()
			canvas := image.NewRGBA(imgSize)
			var watermarkOffset image.Point
			if watermarkLocation == "left" {
				watermarkOffset = image.Point{0, imgSize.Max.Y - wmSize.Max.Y}
			} else if watermarkLocation == "right" {
				watermarkOffset = image.Point{imgSize.Max.X - wmSize.Max.X, imgSize.Max.Y - wmSize.Max.Y}
			}

			draw.Draw(canvas, imgSize, srcImage, image.Point{0, 0}, draw.Src)
			draw.DrawMask(canvas, imgSize.Add(watermarkOffset), scaledWatermark, image.Point{0, 0}, mask, image.Point{0, 0}, draw.Over)

			saveImage(canvas, targetDir, file.Name())
		}(file, watermark, mask, *paramWatermark, *paramScale, *paramSourceDir, *paramTargetDir)
	}
	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("\nAll done! Editted %d files in %s", len(files), elapsed)
	fmt.Print("\n--------------------------------------\n")
	fmt.Println("")
	fmt.Println("Press any key to exit")
	fmt.Scanln()
}

func getFiles(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	return files
}

func saveImage(img image.Image, pname, fname string) {
	fpath := path.Join(pname, fname)
	outputFile, err := os.Create(fpath)
	if err != nil {
		log.Fatalf("failed to create file: %s", err)
	}
	defer outputFile.Close()

	opt := jpeg.Options{
		Quality: 95,
	}
	if err := jpeg.Encode(outputFile, img, &opt); err != nil {
		log.Fatalf("failed to encode watermarked image: %v", err)
	}
}

func openImage(fname string, ftype string) image.Image {
	inputfile, err := os.Open(fname)
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}
	defer inputfile.Close()

	var srcimage image.Image
	switch ftype {
	case "jpeg":
		if srcimage, err = jpeg.Decode(inputfile); err != nil {
			log.Fatalf("failed to decode jpeg image: %s", err)
		}
	case "png":
		if srcimage, err = png.Decode(inputfile); err != nil {
			log.Fatalf("failed to decode png: %s", err)
		}
	default:
		log.Fatalf("file type: %q is not supported", ftype)
	}
	return srcimage
}
