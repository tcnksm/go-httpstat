package httpstat_test

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/tcnksm/go-httpstat"
)

func Example() {
	req, err := http.NewRequest("GET", "http://deeeet.com", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create go-httpstat powered
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

	log.Printf("Name Lookup:    %d ms", int(result.NameLookup/time.Millisecond))
	log.Printf("Connect:        %d ms", int(result.Connect/time.Millisecond))
	log.Printf("Start Transfer: %d ms", int(result.StartTransfer/time.Millisecond))
	log.Printf("Total:          %d ms", int(result.Total(end)/time.Millisecond))
}
