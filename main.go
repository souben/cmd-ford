package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	dirs          string
	recursive     bool
	humanReadable bool
)

const (
	K = 1000
	M = 1000 * K
	G = 1000 * M
	T = 1000 * G
	P = 1000 * T
)

type dirInfo struct {
	path string
	size int64
}

func init() {
	flag.StringVar(&dirs, "dirs", "", "(required) specify the directories separated by commas")
	flag.BoolVar(&recursive, "recursive", false, "(optional) traverse each of the specified directories recursively")
	flag.BoolVar(&humanReadable, "human", false, "(optional) format the size of the directories in a human-friendly format, e.g., 304K instead of 304,000 bytes")
}

func main() {
	flag.Parse()

	// Check if the user specified a set of directories or not
	if dirs == "" {
		fmt.Println("Error: Please specify a list of directories separated by commas.")
		return
	}

	// Extract the names of the directories
	paths := strings.Split(dirs, ",")
	// Store the accumulative size
	results := make(chan dirInfo)
	errs := make(chan string)
	acc := int64(0)

	for _, path := range paths {
		// Check if the file exists
		fileInfo, err := os.Stat(path)
		if err != nil {
			go func() {
				errs <- fmt.Sprintf("Error: The directory \"%s\" does not exist.\n", path)
			}()
			continue
		}

		// Check if the file is a directory
		if !fileInfo.IsDir() {
			fmt.Printf("Error: \"%s\" is not a directory. Please specify directories only!\n", fileInfo.Name())
			continue
		}

		// Calculate the directory' size
		go getDirSize(path, results)
	}

	for i := 0; i < len(paths); i++ {
		select {
		case dir := <-results:
			fmt.Printf("The size of the directory \"%s\" is %s bytes.\n", dir.path, formatSize(dir.size))
			acc += dir.size
		case err := <-errs:
			log.Println(err)
		}
	}

	close(results)

	// Print the cumulative size
	fmt.Printf("The cumulative size of all specified directories is %s bytes.\n", formatSize(acc))
}

func formatSize(size int64) string {
	if !humanReadable {
		return fmt.Sprintf("%d", size)
	}
	if size < K {
		return fmt.Sprintf("%d", size)
	} else if size < M {
		return fmt.Sprintf("%dK", size/K)
	} else if size < G {
		return fmt.Sprintf("%dM", size/M)
	} else if size < T {
		return fmt.Sprintf("%dG", size/G)
	} else if size < P {
		return fmt.Sprintf("%dT", size/T)
	}
	return fmt.Sprintf("%dP", size/P)
}

// This function will calculate the size of a given directory
// If recursive is set to true, it will also print the size of the subdirectories
func getDirSize(dir string, res chan dirInfo) {
	counter := 0
	results := make(chan dirInfo)
	acc := int64(0)

	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		// Prevent stack overflow by skipping the root directory
		if path == dir {
			return nil
		}

		if !d.IsDir() {
			fileInfo, err := d.Info()
			if err != nil {
				return nil // Ignore errors when getting file info
			}
			acc += fileInfo.Size()
			return nil
		}
		counter += 1
		go getDirSize(path, results)
		return nil
	})

	for counter > 0 {
		info := <-results
		acc += info.size
		if recursive {
			fmt.Printf("The size of the subdirectory \"%s\" is %s bytes.\n", info.path, formatSize(info.size))
		}
		counter -= 1
	}

	close(results)
	res <- dirInfo{path: dir, size: acc}
}
