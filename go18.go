// +build go1.8

package httpstat

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"time"
)

// End sets the time when reading response is done.
// This must be called after reading response body.
func (r *Result) End(t time.Time) {
	r.trasferDone = t
	//r.t5 = t // for Formatter

	// This means result is empty (it does nothing).
	// Skip setting value(contentTransfer and total will be zero).
	if r.dnsStart.IsZero() {
		return
	}

	r.ContentTransfer_duration = r.trasferDone.Sub(r.transferStart)
	r.Total_duration += r.ContentTransfer_duration
	r.End_done = r.trasferDone.Sub(r.dnsStart)
}

func withClientTrace(ctx context.Context, r *Result) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.dnsStart = time.Now()
		},

		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.dnsDone = time.Now()
			
			r.DNSLookup_duration = r.dnsDone.Sub(r.dnsStart)
			r.Total_duration = r.DNSLookup_duration
			r.NameLookup_done = r.DNSLookup_duration
			
		},

		ConnectStart: func(_, _ string) {
			r.tcpStart = time.Now()

			// When connecting to IP (When no DNS lookup)
			if r.dnsStart.IsZero() {
				r.dnsStart = r.tcpStart
				r.dnsDone = r.tcpStart
			}
		},

		ConnectDone: func(network, addr string, err error) {
			r.tcpDone = time.Now()

			r.TCPConnection_duration = r.tcpDone.Sub(r.tcpStart)
			r.Connect_done = r.tcpDone.Sub(r.dnsStart)
			r.Total_duration += r.TCPConnection_duration
		},

		TLSHandshakeStart: func() {
			r.isTLS = true
			r.tlsStart = time.Now()
		},

		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.tlsDone = time.Now()

			r.TLSHandshake_duration = r.tlsDone.Sub(r.tlsStart)
			r.Total_duration +=r.TLSHandshake_duration
			r.Pretransfer_done = r.tlsDone.Sub(r.dnsStart)
		},

		GotConn: func(i httptrace.GotConnInfo) {
			// Handle when keep alive is used and connection is reused.
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

			// When client doesn't use DialContext or using old (before go1.7) `net` pakcage, DNS/TCP hook is not called.
			// When connection is re-used, DNS/TCP/TLS hook is not called.
			if (r.dnsStart.IsZero() && r.tcpStart.IsZero()) || (r.isReused) {
				now := r.serverStart

				r.dnsStart = now
				r.dnsDone = now
				r.tcpStart = now
				r.tcpDone = now
				if r.isTLS {
					r.tlsStart = now
					r.tlsDone = now
				}
			}

		},

		GotFirstResponseByte: func() {
			r.serverDone = time.Now()
			r.ServerProcessing_duration = r.serverDone.Sub(r.serverStart)
			r.StartTransfer_done = r.serverDone.Sub(r.dnsStart)
			r.Total_duration += r.ServerProcessing_duration
			r.transferStart = r.serverDone
		},
	})
}
