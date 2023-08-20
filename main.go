package main
import (
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
	"runtime"
	"strconv"
)

type Part struct {
	Data  []byte
	Index int
}


const (
	// Sample download file provided by Vodafone
	// to test internet
	url     = "http://212.183.159.230/5MB.zip"
	  
	// Number of Goroutines to spawn for download.
	//workers = 5
  )

 func MaxParallelism() int {
    maxProcs := runtime.GOMAXPROCS(0)
    numCPU := runtime.NumCPU()
    if maxProcs < numCPU {
        return maxProcs
    }
    return numCPU
}

  func download(index, size int, c chan Part, workers int) { //readCh chan int

	client := &http.Client{}
	
	// calculate offset by multiplying
	// index with size
	start := index * size
	
	// Write data range in correct format
	// I'm reducing one from the end size to account for
	// the next chunk starting there
	dataRange := fmt.Sprintf("bytes=%d-%d", start, start+size-1)
	
	// if this is downloading the last chunk
	// rewrite the header. It's an easy way to specify
	// getting the rest of the file
	if index == workers-1 {
		dataRange = fmt.Sprintf("bytes=%d-", start)
	}

	log.Println(dataRange)

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		// code to restart download
		return
	}

	req.Header.Add("Range", dataRange)

	resp, err := client.Do(req)

	if err != nil {
		// code to restart download
		return
	}

	defer resp.Body.Close()
	// chunk := []byte{}
	// chunk_size := 100
	// for i := 0; i < size; i+=chunk_size {
	// 	n, err := io.ReadAtLeast(resp.Body, chunk, chunk_size)
	// }
	

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		// code to restart download
		return
	}

	c <- Part{Index: index, Data: body}
}

func main() {

	var size int
	workers := MaxParallelism()
	partAmount := workers * 100
	log.Println("Workers amount: ", workers)
	results := make(chan Part, workers)
	parts := make([][]byte, partAmount)

	client := &http.Client{}

	req, err := http.NewRequest("HEAD", url, nil)

	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}


	if header, ok := resp.Header["Content-Length"]; ok {
		fileSize, err := strconv.Atoi(header[0])

		if err != nil {
			log.Fatal("File size could not be determined : ", err)
		}

		size = fileSize / partAmount

	} else {
		log.Fatal("File size was not provided!")
	}



	total_counter := 0
	nextPart := 0
	for ; nextPart < workers; nextPart++ {
		go download(nextPart, size, results, partAmount)
	}
	  
	for part := range results {
		total_counter++;
		log.Printf("Downloaded %d parts out of %d, part index: %d", total_counter, partAmount, part.Index)
		parts[part.Index] = part.Data
		if total_counter == partAmount {
		  break
		} else if nextPart < partAmount { // FIX: Added a check that we are not exceeding the amount of chunks 
			go download(nextPart, size, results, partAmount)
			nextPart++
		}
	}

	file := []byte{}

	for _, part := range parts {
		file = append(file, part...)
	}

	// Set permissions accordingly, 0700 may not
	// be the best choice
	err = ioutil.WriteFile("./data.zip", file, 0700)

	if err != nil {
		log.Fatal(err)
	}
}