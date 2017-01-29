// Package httpstat traces HTTP latency infomation (DNSLookup, TCP Connection and so on) on any golang HTTP request.
// It uses `httptrace` package. Just create `go-httpstat` powered `context.Context` and give it your `http.Request` (no big code modification is required).
package httpstat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http/httptrace"
	"strings"
	"time"
)

// Result stores httpstat info.
type Result struct {
	// The following are duration for each phase
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	contentTransfer  time.Duration

	// The followings are timeline of request
	NameLookup    time.Duration
	Connect       time.Duration
	Pretransfer   time.Duration
	StartTransfer time.Duration
	total         time.Duration

	t0 time.Time
	t1 time.Time
	t2 time.Time
	t3 time.Time
	t4 time.Time

	t5 time.Time // need to be provided from outside

	// isTLS is true when connection seems to use TLS
	isTLS bool

	// isReused is true when connection is reused (keep-alive)
	isReused bool
}

func (r *Result) durations() map[string]time.Duration {
	return map[string]time.Duration{
		"DNSLookup":        r.DNSLookup,
		"TCPConnection":    r.TCPConnection,
		"TLSHandshake":     r.TLSHandshake,
		"ServerProcessing": r.ServerProcessing,
		"ContentTransfer":  r.contentTransfer,

		"NameLookup":    r.NameLookup,
		"Connect":       r.Connect,
		"Pretransfer":   r.Connect,
		"StartTransfer": r.StartTransfer,
		"Total":         r.total,
	}
}

// ContentTransfer returns the duration of content transfer time.
// It is from first response byte to the given time. The time must
// be time after read body (go-httpstat can not detect that time).
func (r *Result) ContentTransfer(t time.Time) time.Duration {
	return t.Sub(r.t4)
}

// Total returns the duration of total http request.
// It is from dns lookup start time to the given time. The
// time must be time after read body (go-httpstat can not detect that time).
func (r *Result) Total(t time.Time) time.Duration {
	return t.Sub(r.t0)
}

// End sets the time when reading response is done.
// This must be called after reading response body.
func (r *Result) End(t time.Time) {
	r.t5 = t

	// This means result is empty (it does nothing).
	// Skip setting value(contentTransfer and total will be zero).
	if r.t0.IsZero() {
		return
	}

	r.contentTransfer = r.t5.Sub(r.t4)
	r.total = r.t5.Sub(r.t0)
}

// Format implements fmt.Formatter interface.
func (r Result) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			var buf bytes.Buffer
			fmt.Fprintf(&buf, "DNS lookup:        %4d ms\n",
				int(r.DNSLookup/time.Millisecond))
			fmt.Fprintf(&buf, "TCP connection:    %4d ms\n",
				int(r.TCPConnection/time.Millisecond))
			fmt.Fprintf(&buf, "TLS handshake:     %4d ms\n",
				int(r.TLSHandshake/time.Millisecond))
			fmt.Fprintf(&buf, "Server processing: %4d ms\n",
				int(r.ServerProcessing/time.Millisecond))

			if !r.t5.IsZero() {
				fmt.Fprintf(&buf, "Content transfer:  %4d ms\n\n",
					int(r.contentTransfer/time.Millisecond))
			} else {
				fmt.Fprintf(&buf, "Content transfer:  %4s ms\n\n", "-")
			}

			fmt.Fprintf(&buf, "Name Lookup:    %4d ms\n",
				int(r.NameLookup/time.Millisecond))
			fmt.Fprintf(&buf, "Connect:        %4d ms\n",
				int(r.Connect/time.Millisecond))
			fmt.Fprintf(&buf, "Pre Transfer:   %4d ms\n",
				int(r.Pretransfer/time.Millisecond))
			fmt.Fprintf(&buf, "Start Transfer: %4d ms\n",
				int(r.StartTransfer/time.Millisecond))

			if !r.t5.IsZero() {
				fmt.Fprintf(&buf, "Total:          %4d ms\n",
					int(r.total/time.Millisecond))
			} else {
				fmt.Fprintf(&buf, "Total:          %4s ms\n", "-")
			}
			io.WriteString(s, buf.String())
			return
		}

		fallthrough
	case 's', 'q':
		d := r.durations()
		list := make([]string, 0, len(d))
		for k, v := range d {
			// Handle when End function is not called
			if (k == "ContentTransfer" || k == "Total") && r.t5.IsZero() {
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
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			_, port, err := net.SplitHostPort(hostPort)
			if err != nil {
				return
			}

			// Heuristic way to detect
			if port == "443" {
				r.isTLS = true
			}
		},

		DNSStart: func(i httptrace.DNSStartInfo) {
			r.t0 = time.Now()
		},
		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.t1 = time.Now()
			r.DNSLookup = r.t1.Sub(r.t0)
			r.NameLookup = r.t1.Sub(r.t0)
		},

		ConnectStart: func(_, _ string) {
			// When connecting to IP
			if r.t0.IsZero() {
				r.t0 = time.Now()
				r.t1 = r.t0
			}
		},

		ConnectDone: func(network, addr string, err error) {
			r.t2 = time.Now()
			if r.isTLS {
				r.TCPConnection = r.t2.Sub(r.t1)
				r.Connect = r.t2.Sub(r.t0)
			}
		},

		GotConn: func(i httptrace.GotConnInfo) {
			// Handle when keep alive is enabled and connection is reused.
			// DNSStart(Done) and ConnectStart(Done) is skipped
			if i.Reused {
				r.t0 = time.Now()
				r.t1 = r.t0
				r.t2 = r.t0

				r.isReused = true
			}
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.t3 = time.Now()

			// This means DNSStart, Done and ConnectStart is not
			// called. This happens if client doesn't use DialContext
			// or using net package before go1.7.
			if r.t0.IsZero() && r.t1.IsZero() && r.t2.IsZero() {
				r.t0 = time.Now()
				r.t1 = r.t0
				r.t2 = r.t0
				r.t3 = r.t0
			}

			// When connection is reused, TLS handshake is skipped.
			if r.isReused {
				r.t3 = r.t0
			}

			if r.isTLS {
				r.TLSHandshake = r.t3.Sub(r.t2)
				r.Pretransfer = r.t3.Sub(r.t0)
				return
			}

			r.TCPConnection = r.t3.Sub(r.t1)
			r.Connect = r.t3.Sub(r.t0)

			r.TLSHandshake = r.t3.Sub(r.t3)
			r.Pretransfer = r.Connect
		},
		GotFirstResponseByte: func() {
			r.t4 = time.Now()
			r.ServerProcessing = r.t4.Sub(r.t3)
			r.StartTransfer = r.t4.Sub(r.t0)
		},
	})
}
