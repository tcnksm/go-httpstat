package httpstat

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"
)

const (
	TestDomainHTTP  = "http://www.web-stat.com/"
	TestDomainHTTPS = "https://blog.golang.org/"
)

// Recommended way to instantiate a Transport
// Should cache connections
func DefaultTransport() *http.Transport {
	// It comes from std transport.go
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Just forgot about the Dialer
func NoDialerTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
}

// Before go1.7, it uses non context based Dial function.
// It doesn't support httptrace.
func OldTransport() *http.Transport {
	return &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
}

func DefaultClient() *http.Client {
	return &http.Client{
		Transport: DefaultTransport(),
	}
}

func NoDialerClient() *http.Client {
	return &http.Client{
		Transport: NoDialerTransport(),
	}
}

func OldDialerClient() *http.Client {
	return &http.Client{
		Transport: OldTransport(),
		Timeout:   time.Second * 10,
	}
}



func NewRequest(t *testing.T, urlStr string, result *Result) *http.Request {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		t.Fatal("NewRequest failed:", err)
	}

	ctx := WithHTTPStat(req.Context(), result)
	return req.WithContext(ctx)
}


func TestHTTPStat_HTTPS(t *testing.T) {
	/*
	DNSDone
	tcpDone
	tlsDone
	GotConn
	serverStart
	*/
	var result Result
	req := NewRequest(t, TestDomainHTTPS, &result)

	client := DefaultClient()
	res, err := client.Do(req)
	if err != nil {
		t.Fatal("client.Do failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		t.Fatal("io.Copy failed:", err)
	}
	res.Body.Close()
	result.End(time.Now())

	if !result.isTLS {
		t.Fatal("isTLS should be true")
	}

	for k, d := range result.durations() {
		if d <= 0*time.Millisecond {
			t.Fatalf("expect %s to be non-zero", k)
		}
	}
	
	
}


func TestHTTPStat_HTTP(t *testing.T) {
	/*
	DNSDone
	tcpDone
	GotConn
	serverStart
	*/
	var result Result
	req := NewRequest(t, TestDomainHTTP, &result)

	client := DefaultClient()
	res, err := client.Do(req)
	if err != nil {
		t.Fatal("client.Do failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		t.Fatal("io.Copy failed:", err)
	}
	res.Body.Close()
	result.End(time.Now())

	if result.isTLS {
		t.Fatal("isTLS should be false")
	}

	if got, want := result.TLSHandshake, 0*time.Millisecond; got != want {
		t.Fatalf("TLSHandshake time of HTTP = %d, want %d", got, want)
	}

	// Except TLS should be non zero
	durations := result.durations()
	delete(durations, "TLSHandshake")

	for k, d := range durations {
		if d <= 0*time.Millisecond {
			t.Fatalf("expect %s to be non-zero", k)
		}
	}
}

func TestHTTPStat_KeepAlive_HTTPS(t *testing.T) {
	/*
	GotConn
	serverStart
	*/

	req1, err := http.NewRequest("GET", TestDomainHTTPS, nil)
	if err != nil {
		t.Fatal("NewRequest failed:", err)
	}

	client := DefaultClient()
	res1, err := client.Do(req1)
	if err != nil {
		t.Fatal("Request failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res1.Body); err != nil {
		t.Fatal("Copy body failed:", err)
	}
	res1.Body.Close()

	var result Result
	req2 := NewRequest(t, TestDomainHTTPS, &result)

	// When second request, connection should be re-used.
	res2, err := client.Do(req2)
	if err != nil {
		t.Fatal("Request failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res2.Body); err != nil {
		t.Fatal("Copy body failed:", err)
	}
	res2.Body.Close()
	result.End(time.Now())

	// The following values should be zero.
	// Because connection is reused.
	durations := []time.Duration{
		result.DNSLookup,
		result.TCPConnection,
		result.TLSHandshake,
	}

	for i, d := range durations {
		if got, want := d, 0*time.Millisecond; got != want {
			t.Fatalf("#%d expect %d to be eq %d", i, got, want)
		}
	}
}

func TestHTTPStat_KeepAlive_HTTP(t *testing.T) {
	/*
	GotConn
	serverStart
	*/
	req1, err := http.NewRequest("GET", TestDomainHTTP, nil)
	if err != nil {
		t.Fatal("NewRequest failed:", err)
	}

	client := DefaultClient()
	res1, err := client.Do(req1)
	if err != nil {
		t.Fatal("Request failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res1.Body); err != nil {
		t.Fatal("Copy body failed:", err)
	}
	res1.Body.Close()

	var result Result
	req2 := NewRequest(t, TestDomainHTTP, &result)

	// When second request, connection should be re-used.
	res2, err := client.Do(req2)
	if err != nil {
		t.Fatal("Request failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res2.Body); err != nil {
		t.Fatal("Copy body failed:", err)
	}
	res2.Body.Close()
	result.End(time.Now())

	// The following values should be zero.
	// Because connection is reused.
	durations := []time.Duration{
		result.DNSLookup,
		result.TCPConnection,
	}

	for i, d := range durations {
		if got, want := d, 0*time.Millisecond; got != want {
			t.Fatalf("#%d expect %d to be eq %d", i, got, want)
		}
	}
}


func TestHTTPStat_beforeGO17(t *testing.T) {
	/*
	tlsDone  (If HTTPS)
	GotConn
	serverStart
	*/
	var result1,result2 Result
	
	// Before go1.7, it uses non context based Dial function.
	// It doesn't support httptrace.
	client := OldDialerClient()

	req1 := NewRequest(t, TestDomainHTTPS, &result1)

	res, err := client.Do(req1)
	if err != nil {
		t.Fatal("client.Do failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		t.Fatal("io.Copy failed:", err)
	}
	res.Body.Close()
	result1.End(time.Now())

	// The following values are not mesured, should be 0
	durations := []time.Duration{
		result1.DNSLookup,
		result1.TCPConnection,
	}

	for i, d := range durations {
		if got, want := d, 0*time.Millisecond; got != want {
			t.Logf("Result is \n %+v",result1 )
			t.Fatalf("#%d expect %d to be eq %d", i, got, want)
		}
	}
	
	req2 := NewRequest(t, TestDomainHTTP, &result2)

	res, err = client.Do(req2)
	if err != nil {
		t.Fatal("client.Do failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		t.Fatal("io.Copy failed:", err)
	}
	res.Body.Close()
	result2.End(time.Now())

	// The following values are not mesured, should be 0
	durations = []time.Duration{
		result2.DNSLookup,
		result2.TCPConnection,
		result2.TLSHandshake,
	}

	for i, d := range durations {
		if got, want := d, 0*time.Millisecond; got != want {
			t.Logf("Result is \n %+v",result2 )
			t.Fatalf("#%d expect %d to be eq %d", i, got, want)
		}
	}
}


func TestHTTPStat_NoDialer_HTTPS(t *testing.T) {
	/*
	DNSDone   >> First time only
	tcpDone  >> First time only
	tlsDone  >> First time only
	GotConn
	serverStart
	*/
	var result1,result2 Result
	
	// Using the Transport but without a Dialer
	client := NoDialerClient()

	req1 := NewRequest(t, TestDomainHTTPS, &result1)

	res, err := client.Do(req1)
	if err != nil {
		t.Fatal("client.Do failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		t.Fatal("io.Copy failed:", err)
	}
	res.Body.Close()
	result1.End(time.Now())

	if !result1.isTLS {
		t.Fatal("isTLS should be true")
	}
	
	durationsMap := result1.durations()
	// Without a Dialer, the DNS Lookup could be 0, so not comparing it
	delete(durationsMap, "DNSLookup")
	for k, d := range durationsMap {
		if d <= 0*time.Millisecond {
			t.Logf("Result is \n %+v",result1 )
			t.Fatalf("expect %s to be non-zero", k)
		}
	}

	
	req2 := NewRequest(t, TestDomainHTTPS, &result2)

	// When second request, connection should be re-used.
	res2, err := client.Do(req2)
	if err != nil {
		t.Fatal("Request failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res2.Body); err != nil {
		t.Fatal("Copy body failed:", err)
	}
	res2.Body.Close()
	result2.End(time.Now())

	// The following values should be zero.
	// Because connection is reused.
	durations := []time.Duration{
		result2.DNSLookup,
		result2.TCPConnection,
		result2.TLSHandshake,
	}

	for i, d := range durations {
		if got, want := d, 0*time.Millisecond; got != want {
			t.Logf("Result is \n %+v",result2 )
			t.Fatalf("#%d expect %d to be eq %d", i, got, want)
		}
	}
}

func TestHTTPStat_NoDialer_HTTP(t *testing.T) {
	/*
	DNSDone   >> First time only
	tcpDone  >> First time only
	GotConn
	serverStart
	*/
	var result1,result2 Result
	
	// Using the Transport but without a Dialer
	client := NoDialerClient()

	req1 := NewRequest(t, TestDomainHTTP, &result1)

	res, err := client.Do(req1)
	if err != nil {
		t.Fatal("client.Do failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		t.Fatal("io.Copy failed:", err)
	}
	res.Body.Close()
	result1.End(time.Now())

	if result1.isTLS {
		t.Fatal("isTLS should be false")
	}

	durationsMap := result1.durations()
	// Without a Dialer, the DNS Lookup could be 0, so not comparing it
	delete(durationsMap, "DNSLookup")
	delete(durationsMap, "TLSHandshake")
	for k, d := range durationsMap {
		if d <= 0*time.Millisecond {
			t.Logf("Result is \n %+v",result1 )
			t.Fatalf("expect %s to be non-zero", k)
		}
	}

	
	req2 := NewRequest(t, TestDomainHTTP, &result2)

	// When second request, connection should be re-used.
	res2, err := client.Do(req2)
	if err != nil {
		t.Fatal("Request failed:", err)
	}

	if _, err := io.Copy(ioutil.Discard, res2.Body); err != nil {
		t.Fatal("Copy body failed:", err)
	}
	res2.Body.Close()
	result2.End(time.Now())

	// The following values should be zero.
	// Because connection is reused.
	durations := []time.Duration{
		result2.DNSLookup,
		result2.TCPConnection,
		result2.TLSHandshake,
	}

	for i, d := range durations {
		if got, want := d, 0*time.Millisecond; got != want {
			t.Logf("Result is \n %+v",result2 )
			t.Fatalf("#%d expect %d to be eq %d", i, got, want)
		}
	}
}

func TestTotal_Zero(t *testing.T) {
	result := &Result{}
	result.End(time.Now())

	zero := 0 * time.Millisecond
	if result.Total != zero {
		t.Fatalf("Total time is %d, want %d", result.Total, zero)
	}

	if result.ContentTransfer != zero {
		t.Fatalf("Total time is %d, want %d", result.ContentTransfer, zero)
	}
}

func TestHTTPStat_Formatter(t *testing.T) {
	result := Result{
		DNSLookup:        100 * time.Millisecond,
		TCPConnection:    100 * time.Millisecond,
		TLSHandshake:     100 * time.Millisecond,
		ServerProcessing: 100 * time.Millisecond,
		ContentTransfer:  100 * time.Millisecond,
		NameLookup:    100 * time.Millisecond,
		Connect:       100 * time.Millisecond,
		Pretransfer:   100 * time.Millisecond,
		StartTransfer: 100 * time.Millisecond,
		Total:         100 * time.Millisecond,

		trasferDone: time.Now(),
	}

	want := `DNS lookup:         100 ms
TCP connection:     100 ms
TLS handshake:      100 ms
Server processing:  100 ms
Content transfer:   100 ms

Name Lookup:     100 ms
Connect:         100 ms
Pre Transfer:    100 ms
Start Transfer:  100 ms
Total:           100 ms
`
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%+v", result)
	if got := buf.String(); want != got {
		t.Fatalf("expect to be eq:\n\nwant:\n\n%s\ngot:\n\n%s\n", want, got)
	}
}
