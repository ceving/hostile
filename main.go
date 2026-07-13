package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ferdypruis/iso3166"
	"github.com/openrdap/rdap"
)

// Number of parallel workers for network queries
const workerCount = 10

// Cache for already queried IPs to minimize rate-limits and latency
var (
	cacheMutex sync.RWMutex
	ipCache    = make(map[string]map[string]interface{})
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: location-lookup IP_ADDRESS_ATTRIBUTE\n")
		os.Exit(1)
	}
	ipKey := os.Args[1]

	// Configure RDAP client with a custom HTTP timeout.
	// Bootstrap is initialized automatically on first use when left nil.
	client := &rdap.Client{
		HTTP: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Channels for parallel processing
	linesIn := make(chan string, workerCount*2)
	linesOut := make(chan string, workerCount*2)

	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(client, ipKey, linesIn, linesOut)
		}()
	}

	// Separate goroutine for output (order is not strictly preserved, but very fast)
	done := make(chan struct{})
	go func() {
		for out := range linesOut {
			fmt.Println(out)
		}
		close(done)
	}()

	// Read STDIN line by line
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		linesIn <- scanner.Text()
	}

	close(linesIn)
	wg.Wait()
	close(linesOut)
	<-done

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Cant read STDIN: %v\n", err)
		os.Exit(1)
	}
}

func worker(client *rdap.Client, ipKey string, linesIn <-chan string, linesOut chan<- string) {
	for line := range linesIn {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Parse JSON line into a generic map
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &record); err != nil {
			// Pass through unchanged if not valid JSON
			linesOut <- trimmed
			continue
		}

		// Read IP attribute
		ipVal, ok := record[ipKey]
		if !ok {
			linesOut <- trimmed
			continue
		}

		ipStr, ok := ipVal.(string)
		if !ok || ipStr == "" {
			linesOut <- trimmed
			continue
		}

		// 1. Cache lookup
		cacheMutex.RLock()
		cachedData, found := ipCache[ipStr]
		cacheMutex.RUnlock()

		if found {
			for k, v := range cachedData {
				record[k] = v
			}
			outputJSON(record, linesOut, trimmed)
			continue
		}

		// 2. RDAP network query.
		// QueryIP takes only the IP string; the 5s timeout is set on the HTTP client above.
		ipNet, err := client.QueryIP(ipStr)

		enrichment := make(map[string]interface{})

		if err == nil && ipNet != nil {
			// Extract ISO country code
			if ipNet.Country != "" {
				enrichment["iso3166"] = ipNet.Country
				if c, err := iso3166.FromAlpha2(ipNet.Country); err == nil {
					enrichment["country"] = c.Name() // z.B. "China"
				} else {
					enrichment["country"] = ipNet.Country // Fallback auf den Code
				}
			}

			// Determine Autonomous System (ASN) from linked entities.
			// RDAP stores ASN associations in entities whose Handle starts with "AS".
			for _, entity := range ipNet.Entities {
				if strings.HasPrefix(strings.ToUpper(entity.Handle), "AS") {
					enrichment["asn"] = entity.Handle
					break
				}
			}
		}

		// Store in cache
		cacheMutex.Lock()
		ipCache[ipStr] = enrichment
		cacheMutex.Unlock()

		// Enrich record
		for k, v := range enrichment {
			record[k] = v
		}

		outputJSON(record, linesOut, trimmed)
	}
}

func outputJSON(record map[string]interface{}, linesOut chan<- string, fallback string) {
	updatedBytes, err := json.Marshal(record)
	if err != nil {
		linesOut <- fallback
		return
	}
	linesOut <- string(updatedBytes)
}
