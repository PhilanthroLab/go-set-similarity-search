package SetSimilaritySearch

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	// Download from https://github.com/ekzhu/set-similarity-search-benchmarks
	allPairsContainmentBenchmarkFilename  = "canada_us_uk_opendata.inp.gz"
	allPairsContainmentBenchmarkResult    = "canada_us_uk_opendata_all_pairs_containment.csv"
	allPairsContainmentBenchmarkThreshold = 0.9
	allPairsContainmentBenchmarkMinSize   = 10
)

// Read set similarity search benchmark files from
// https://github.com/ekzhu/set-similarity-search-benchmarks
func readGzippedTransformedSets(filename string,
	firstLineInfo bool, minSize int) (sets [][]int) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}
	defer gz.Close()
	sets = make([][]int, 0)
	scanner := bufio.NewScanner(gz)
	scanner.Buffer(nil, 1024*1024*1024*4)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), "\n")
		if firstLineInfo && len(sets) == 0 {
			// Initialize the sets using the info given by the first line
			count, err := strconv.Atoi(strings.Split(line, " ")[0])
			if err != nil {
				panic(err)
			}
			sets = make([][]int, 0, count)
			firstLineInfo = false
			continue
		}
		raw := strings.Split(strings.Split(line, "\t")[1], ",")
		if len(raw) < minSize {
			continue
		}
		set := make([]int, len(raw))
		for i := range set {
			set[i], err = strconv.Atoi(raw[i])
			if err != nil {
				panic(err)
			}
		}
		sets = append(sets, set)
		if len(sets)%100 == 0 {
			fmt.Printf("\rRead %d sets so far", len(sets))
		}
	}
	fmt.Println()
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return sets
}

func BenchmarkOpenDataAllPairContainment(b *testing.B) {
	log.Printf("Reading transformed sets from %s",
		allPairsContainmentBenchmarkFilename)
	start := time.Now()
	sets := readGzippedTransformedSets(allPairsContainmentBenchmarkFilename,
		/*firstLineInfo=*/ true,
		allPairsContainmentBenchmarkMinSize)
	log.Printf("Finished reading %d transformed sets in %s", len(sets),
		time.Now().Sub(start).String())
	log.Printf("Building search index")
	start = time.Now()
	searchIndex, err := NewSearchIndex(sets, "containment",
		allPairsContainmentBenchmarkThreshold)
	if err != nil {
		b.Fatal(err)
	}
	log.Printf("Finished building search index in %s",
		time.Now().Sub(start).String())
	out, err := os.Create(allPairsContainmentBenchmarkResult)
	if err != nil {
		b.Fatal(err)
	}
	defer out.Close()
	w := csv.NewWriter(out)
	log.Printf("Begin querying")
	start = time.Now()
	var count int
	for i, set := range sets {
		results := searchIndex.Query(set)
		for _, result := range results {
			if result.X == i {
				continue
			}
			w.Write([]string{
				strconv.Itoa(i),
				strconv.Itoa(result.X),
				strconv.FormatFloat(result.Similarity, 'f', 4, 64),
			})
		}
		count++
		if count%100 == 0 {
			fmt.Printf("\rQueried %d sets so far", count)
		}
	}
	fmt.Println()
	log.Printf("Finished querying in %s", time.Now().Sub(start).String())
	w.Flush()
	if err := w.Error(); err != nil {
		b.Fatal(err)
	}
	log.Printf("Results written to %s", allPairsContainmentBenchmarkResult)
}
