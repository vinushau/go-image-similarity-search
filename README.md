# go-image-similarity-search

Finds the 5 most visually similar images to a query image by computing and comparing color histograms across an image dataset using multiple goroutines. The program also benchmarks performance across 7 different thread counts automatically.

## How It Works

1. The query image histogram and all dataset image histograms are computed from scratch on every run (no pre-computed files are used).
2. The color space is reduced to 3 bits per channel (512 bins) via right bit-shifts.
3. Histograms are normalized so all bins sum to 1.0.
4. Similarity is measured using histogram intersection, a score of 1.0 means identical color distributions, 0.0 means no colors in common.
5. The 5 dataset images with the highest intersection scores are reported.

## Project Structure

```
.
├── SimilaritySearch.go       # Main program
├── queryImages/              # Query images must be placed here
│   ├── q01.jpg
│   ├── q02.jpg
│   └── ...
├── imageDataset2_15_20/      # Dataset images (jpg)
│   ├── img0001.jpg
│   └── ...
└── README.md
```

## Requirements

- [Go](https://go.dev/dl/) 1.18 or later


## Running the Program

```bash
go run SimilaritySearch.go <queryImageFilename> <imageDatasetDirectory>
```

### Examples

**Windows:**
```bash
go run SimilaritySearch.go q12.jpg .\imageDataset2_15_20
```

**macOS / Linux:**
```bash
go run SimilaritySearch.go q12.jpg ./imageDataset2_15_20
```

> The query image `q12.jpg` must exist at `queryImages/q12.jpg` relative to where you run the command.

## Expected Output

The program automatically benchmarks **all 7 thread counts** (K = 1, 2, 4, 16, 64, 256, 1048) for every run and prints the time taken for each:

```
Time taken (1): 3.241s
Time taken (2): 1.872s
Time taken (4): 1.103s
Time taken (16): 654ms
Time taken (64): 489ms
Time taken (256): 501ms
Time taken (1048): 534ms
```

## Concurrency Model

| Component | Role |
|---|---|
| `computeHistograms` | Goroutine that processes a slice of dataset images and sends each Histogram to the channel |
| `computeHistogram` | Computes and returns the histogram for a single image |
| `getKSublists` | Splits the dataset image list into K equal slices for parallel processing |
| `sync.WaitGroup` | Ensures all goroutines finish before the histogram channel is closed |
| `histogramChan` | Channel through which computed histograms are streamed to the main goroutine |
| `isComplete` | Signals when all goroutines are done and the channel is safe to close |

The query image histogram is computed in the main goroutine while dataset goroutines run concurrently.

## Key Functions

| Function | Signature | Description |
|---|---|---|
| `computeHistogram` | `(path string, bitDepth int) (Histogram, error)` | Loads a JPEG and computes its reduced color histogram |
| `computeHistograms` | `(path []string, bitDepth int, histChannel chan<- Histogram)` | Goroutine that computes histograms for a slice of images and sends to channel |
| `makeHist` | `(image image.Image, x int) []int` | Builds the raw histogram by iterating over all pixels |
| `normHist` | `(hist Histogram) []float64` | Normalizes histogram bins by total pixel count |
| `compared` | `(query, dataSet []float64) float64` | Computes histogram intersection score |
| `read` | `(queryHistogram Histogram, hChan chan Histogram)` | Reads channel, compares all histograms, calls topFive |
| `topFive` | `(similarityScores map[string]float64)` | Sorts results and returns the 5 most similar images |
| `getKSublists` | `(datasetImageLis []string, valueOfK int) [][]string` | Splits image list into K slices for parallel goroutines |
| `bins` | `(x int) int` | Returns histogram size: `2^(3*x)` bins |

## Histogram Index Formula

For a pixel with reduced channels `[R', G', B']` at bit depth `D`:

```
index = (R' << (2*D)) + (G' << D) + B'
```

> **Note:** Go's `image` package stores pixel values as `uint32` in the range 0–65535. The code right-shifts by `16 - D` (instead of `8 - D`) to bring values into the correct reduced range before computing the index.
