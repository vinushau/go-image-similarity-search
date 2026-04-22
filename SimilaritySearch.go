package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Stores name and the histogram
type Histogram struct {
	Name string
	Histogram []int
}

var waitGroup sync.WaitGroup

func main() {
	if len(os.Args) < 3 { 
		fmt.Println(os.Args)
		fmt.Printf("Usage: go run similaritySearch.go queryImageFilename imageDatasetDirectory")
		os.Exit(1)
	}
	qImage := "queryImages/" + os.Args[1]
	datasetlis := os.Args[2] 
	//list to hold all possible thread count (K) values
	threadCounts := []int {1,2,4,16,64,256,1048}
	dImages, err := getListOfImages(datasetlis) // extract images from image data folder
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	
	for _,valueOfK := range(threadCounts) {
		histogramChan := make(chan Histogram) // create channel to send and get hist information
		isComplete := make(chan bool, 1) // channel for main
		go func() {
			defer close(histogramChan) 
			// iterates through all the files in every sublist after splitting it into k sublists
			for _, dImage := range getKSublists(dImages, valueOfK) {
				waitGroup.Add(1)
				go computeHistograms(dImage, 3, histogramChan)
			}
			waitGroup.Wait() 
			close(isComplete) 
		}()
		
		// compute histogram of query
		qHistogram, err := computeHistogram(qImage, 3)
	
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	
		// measure time taken
		now := time.Now()
		// read and compare the intersection values of the query hist and data image
		read(qHistogram, histogramChan)
		// print time taken to read data
		fmt.Printf("Time taken (%d): %v\n", valueOfK, time.Since(now))
		<-isComplete 
	}

}

// getKSublists splits the list of images into k sublists
func getKSublists(datasetImageLis []string, valueOfK int) [][]string {
	len := len(datasetImageLis)
	kSubLists := make([][]string, valueOfK)
	remainingImages := len % valueOfK 
	lenOfSublist := len / valueOfK 
	// iterate through every k slice produced
	for i := 0; i < valueOfK; i++ {
		lastIdx := (i+1)*lenOfSublist 
		firstIdx := i*lenOfSublist     
		if i < remainingImages {
			lastIdx++ 
		}
		if lastIdx > len {
			lastIdx = len 
		}
		kSubLists[i] = datasetImageLis[firstIdx:lastIdx]
	}
	return kSubLists 
}

// readImage reads image data using the given path
func readImage(imagePath string) (image.Image, error) {
	imageFile, err := os.Open(imagePath) // open the image file
	if err != nil {
		return nil, err
	}
	defer imageFile.Close()
	image, err := jpeg.Decode(imageFile)
	if err != nil {
		return nil, err
	}
	// return the image
	return image, nil
}

// Gets list of all jpg images in image dataset
func getListOfImages(datasetlis string) ([]string, error) {
	datasetImageLis := []string{}
	filesLis, err := os.ReadDir(datasetlis)
	if err != nil {
		log.Fatal(err)
	}

	// iterate through every file and add only the jpg images from the folder to the list
	for _, file := range filesLis {
		if strings.HasSuffix(file.Name(), ".jpg") {
			// append the .jpg image to list
			datasetImageLis = append(datasetImageLis, datasetlis + "/" + file.Name())
		}
	}
	// return the list of .jpg images
	return datasetImageLis, nil
}

func computeHistogram(path string, bitDepth int) (Histogram, error) {
	// read image using the file image path
	image, err := readImage(path)
	if err != nil {
		return Histogram{}, err
	}
	newHistogram := makeHist(image, bitDepth)
	return Histogram{path, newHistogram}, nil
}

// compute the histograms given an array of file names
func computeHistograms(path []string, bitDepth int, histChannel chan<- Histogram) {
	defer waitGroup.Done()
	// iterate through every image file and create a hist for each file
	for i := 0; i < len(path); i++ {
		image := path[i]
		hist, err := computeHistogram(image, bitDepth)
		if err != nil {
			fmt.Println(err)
			return
		}
		histChannel <- hist
	}
}

func makeHist(Image image.Image, x int) []int {
	hist := make([]int, bins(x))
	for i := Image.Bounds().Min.X; i < Image.Bounds().Max.X; i++ {
		for j := Image.Bounds().Min.Y; j < Image.Bounds().Max.Y; j++ {
			R, G, B, _ := Image.At(i, j).RGBA()
			R >>= uint(16 - x)
			G >>= uint(16 - x)
			B >>= uint(16 - x)
			idx := int((R << uint32(2*x)) + (G << uint32(x)) + B)
			hist[idx]++
		}
	}

	// returns the final histogram
	return hist
}

func normHist(hist Histogram) []float64 {
	image, err := readImage(hist.Name)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	pixels := image.Bounds().Max.X * image.Bounds().Max.Y
	normHist := make([]float64, len(hist.Histogram))
	for i := 0; i < len(hist.Histogram); i++ {
		normHist[i] = (float64(hist.Histogram[i])) / float64(pixels)
	}
	return normHist
}

func compared(query, dataSet []float64) float64 {
	compare := 0.0
	for i := 0; i < len(query); i++ {
		compare += min(query[i], dataSet[i])
	}
	return compare
}

func read(queryHistogram Histogram, hChan chan Histogram) {
    Qnorm := normHist(queryHistogram)
    image := make(map[string]float64)

    for currentHistogram := range hChan {
        data := normHist(currentHistogram)
        image[currentHistogram.Name] = compared(Qnorm, data)
    }

    topFive(image)
}


func topFive(similarityScores map[string]float64) {
    var comparisonResults []struct {
        FileName   string
        CompareVal float64
    }

    for file, val := range similarityScores {
        comparisonResults = append(comparisonResults, struct {
            FileName   string
            CompareVal float64
        }{file, val})
    }

    sort.SliceStable(comparisonResults, func(n, m int) bool {
        return comparisonResults[n].CompareVal > comparisonResults[m].CompareVal
    })
}

func min(x, y float64) float64 {
	if x > y {
		return y
	}
	return x
}

	func bins(x int) int {
	return 1 << uint(3*x)
}