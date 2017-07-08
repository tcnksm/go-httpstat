// Package httpstat traces HTTP latency infomation (DNSLookup_duration, TCP Connection and so on) on any golang HTTP request.
// It uses `httptrace` package. Just create `go-httpstat` powered `context.Context` and give it your `http.Request` (no big code modification is required).
package httpstat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// Result stores httpstat info.
type Result struct {
	// The following are duration for each phase
	DNSLookup_duration        time.Duration
	TCPConnection_duration    time.Duration
	TLSHandshake_duration     time.Duration
	ServerProcessing_duration time.Duration
	ContentTransfer_duration  time.Duration
	Total_duration            time.Duration

	// The followings are timeline of request
	NameLookup_done    time.Duration
	Connect_done       time.Duration
	Pretransfer_done   time.Duration
	StartTransfer_done time.Duration
	End_done           time.Duration

	//t0 time.Time
	//t1 time.Time
	//t2 time.Time
	//t3 time.Time
	//t4 time.Time
	//t5 time.Time // need to be provided from outside

	dnsStart      time.Time
	dnsDone       time.Time
	tcpStart      time.Time
	tcpDone       time.Time
	tlsStart      time.Time
	tlsDone       time.Time
	serverStart   time.Time
	serverDone    time.Time
	transferStart time.Time
	trasferDone   time.Time // need to be provided from outside

	// isTLS is true when connection seems to use TLS
	isTLS bool

	// isReused is true when connection is reused (keep-alive)
	isReused bool
}

func (r *Result) durations() map[string]time.Duration {
	return map[string]time.Duration{
		"DNSLookup_duration":        r.DNSLookup_duration,
		"TCPConnection":    r.TCPConnection_duration,
		"TLSHandshake":     r.TLSHandshake_duration,
		"ServerProcessing": r.ServerProcessing_duration,
		"ContentTransfer":  r.ContentTransfer_duration,
		"Total":            r.Total_duration,
	}
}
func (r *Result) timeline() map[string]time.Duration {
	return map[string]time.Duration{
		"NameLookup":    r.NameLookup_done,
		"Connect":       r.Connect_done,
		"Pretransfer":   r.Pretransfer_done,
		"StartTransfer": r.StartTransfer_done,
		"End":         r.End_done,
	}
}

// ContentTransfer returns the duration of content transfer time.
// It is from first response byte to the given time. The time must
// be time after read body (go-httpstat can not detect that time).
//func (r *Result) ContentTransfer(t time.Time) time.Duration {
	//return t.Sub(r.t4)
//}

// Total returns the duration of total http request.
// It is from dns lookup start time to the given time. The
// time must be time after read body (go-httpstat can not detect that time).
//func (r *Result) Total(t time.Time) time.Duration {
	//return t.Sub(r.t0)
//}

// Format formats stats result.
func (r Result) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			var buf bytes.Buffer
			fmt.Fprintf(&buf, "DNS lookup:        %4d ms\n",
				int(r.DNSLookup_duration/time.Millisecond))
			fmt.Fprintf(&buf, "TCP connection:    %4d ms\n",
				int(r.TCPConnection_duration/time.Millisecond))
			fmt.Fprintf(&buf, "TLS handshake:     %4d ms\n",
				int(r.TLSHandshake_duration/time.Millisecond))
			fmt.Fprintf(&buf, "Server processing: %4d ms\n",
				int(r.ServerProcessing_duration/time.Millisecond))

			if !r.trasferDone.IsZero() {
				fmt.Fprintf(&buf, "Content transfer:  %4d ms\n\n",
					int(r.ContentTransfer_duration/time.Millisecond))
			} else {
				fmt.Fprintf(&buf, "Content transfer:  %4s ms\n\n", "-")
			}

			fmt.Fprintf(&buf, "Name Lookup:    %4d ms\n",
				int(r.NameLookup_done/time.Millisecond))
			fmt.Fprintf(&buf, "Connect:        %4d ms\n",
				int(r.Connect_done/time.Millisecond))
			fmt.Fprintf(&buf, "Pre Transfer:   %4d ms\n",
				int(r.Pretransfer_done/time.Millisecond))
			fmt.Fprintf(&buf, "Start Transfer: %4d ms\n",
				int(r.StartTransfer_done/time.Millisecond))

			if !r.trasferDone.IsZero() {
				fmt.Fprintf(&buf, "Total:          %4d ms\n",
					int(r.Total_duration/time.Millisecond))
			} else {
				fmt.Fprintf(&buf, "Total:          %4s ms\n", "-")
			}
			io.WriteString(s, buf.String())
			return
		}

		fallthrough
	case 's', 'q':
		d := r.durations()
		t := r.timeline()
		list := make([]string, 0, len(d) + len(t))
		for k, v := range d {
			// Handle when End function is not called
			if (k == "ContentTransfer" || k == "Total") && r.trasferDone.IsZero() {
				list = append(list, fmt.Sprintf("%s: - ms", k))
				continue
			}
			list = append(list, fmt.Sprintf("%s: %d ms", k, v/time.Millisecond))
		}
		for k, v := range t {
			// Handle when End function is not called
			if k == "End" && r.trasferDone.IsZero() {
				list = append(list, fmt.Sprintf("%s: - ms", k))
				continue
			}
			list = append(list, fmt.Sprintf("%s: %d ms", k, v/time.Millisecond))
		}
		io.WriteString(s, strings.Join(list, ", "))
	}

}

// WithHTTPStat is a wrapper of httptrace.WithClientTrace. It records the
// time of each httptrace hooks.
func WithHTTPStat(ctx context.Context, r *Result) context.Context {
	return withClientTrace(ctx, r)
}
