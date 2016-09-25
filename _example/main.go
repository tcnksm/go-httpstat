package main

import (
	"log"
	"net/http"
	"time"

	"github.com/tcnksm/go-httpstat"
)

func main() {
	req, err := http.NewRequest("GET", "http://deeeet.com", nil)
	if err != nil {
		log.Fatal(err)
	}

	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	if _, err := client.Do(req); err != nil {
		log.Fatal(err)
	}

	log.Printf("DNS Lookup: %d ms", int(result.DNSLookup/time.Millisecond))
	log.Printf("TCP Connection: %d ms", int(result.TCPConnection/time.Millisecond))
}
