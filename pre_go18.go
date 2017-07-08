// +build !go1.8

package httpstat

import (
	"context"
	"net"
	"net/http/httptrace"
	"time"
)

// End sets the time when reading response is done.
// This must be called after reading response body.
func (r *Result) End(t time.Time) {
	r.trasferDone = t

	// This means there was no initial HTTP Request.
	// Skip setting value(contentTransfer and total will be zero).
	if r.dnsStart.IsZero() {
		return
	}

	r.ContentTransfer = r.trasferDone.Sub(r.transferStart)
	r.Total = r.trasferDone.Sub(r.dnsStart)
}

func withClientTrace(ctx context.Context, r *Result) context.Context {
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
			r.dnsStart = time.Now()
		},
		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.dnsDone = time.Now()
			r.DNSLookup = r.dnsDone.Sub(r.dnsStart)
			r.NameLookup = r.dnsDone.Sub(r.dnsStart)
		},

		ConnectStart: func(_, _ string) {
			r.tcpStart = time.Now()
			// When connecting to IP
			if r.dnsStart.IsZero() {
				r.dnsStart = time.Now()
				r.dnsDone = r.dnsStart
			}
		},

		ConnectDone: func(network, addr string, err error) {
			r.tcpDone = time.Now()
			r.TCPConnection = r.tcpDone.Sub(r.tcpStart)
			r.Connect = r.tcpDone.Sub(r.dnsStart)
		},

		GotConn: func(i httptrace.GotConnInfo) {
			// Handle when keep alive is enabled and connection is reused.
			// DNSStart(Done) and ConnectStart(Done) is skipped
			if i.Reused {
				r.isReused = true
				now := time.Now()
				r.dnsStart = now
				r.dnsDone = now
				r.tcpStart = now
				r.tcpDone = now
			}
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.serverStart = time.Now()

			// This means DNSStart, Done and ConnectStart is not called. This happens if client doesn't use DialContext
			// or using net package before go1.7 or if the connection is reused
			if (r.dnsStart.IsZero() && r.tcpStart.IsZero()) || (r.isReused) {
				now := r.serverStart
				r.dnsStart = now
				r.dnsDone = now
				r.tcpStart = now
				r.tcpDone = now
			}
			
			if r.isTLS {
				r.TLSHandshake = r.serverStart.Sub(r.tcpDone)
				r.Total +=r.TLSHandshake
				r.Pretransfer_done = r.serverStart.Sub(r.dnsStart)
			}

		},
		GotFirstResponseByte: func() {
			r.serverDone = time.Now()
			r.ServerProcessing = r.serverDone.Sub(r.serverStart)
			r.StartTransfer = r.serverDone.Sub(r.dnsStart)
			r.transferStart = r.serverDone
		},
	})
}
