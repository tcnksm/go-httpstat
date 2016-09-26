package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/tcnksm/go-httpstat"
)

func main() {
	req, err := http.NewRequest("GET", "https://github.com", nil)
	if err != nil {
		log.Fatal(err)
	}

	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	end := time.Now()

	log.Printf("DNS lookup: %d ms", int(result.DNSLookup/time.Millisecond))
	log.Printf("TCP connection: %d ms", int(result.TCPConnection/time.Millisecond))
	log.Printf("TLS handshake: %d ms", int(result.TLSHandshake/time.Millisecond))
	log.Printf("Server processing: %d ms", int(result.ServerProcessing/time.Millisecond))
	log.Printf("Content transfer: %d ms", int(result.ContentTransfer(time.Now())/time.Millisecond))
	fmt.Println()

	log.Printf("Name Lookup: %d ms", int(result.NameLookup/time.Millisecond))
	log.Printf("Connect: %d ms", int(result.Connect/time.Millisecond))
	log.Printf("Pre Transfer: %d ms", int(result.Pretransfer/time.Millisecond))
	log.Printf("Start Transfer: %d ms", int(result.StartTransfer/time.Millisecond))
	log.Printf("Total: %d ms", int(result.Total(end)/time.Millisecond))
}
