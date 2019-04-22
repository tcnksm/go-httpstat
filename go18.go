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

	// This means result is empty (it does nothing).
	// Skip setting value(contentTransfer and total will be zero).
	if r.dnsStart.IsZero() {
		return
	}

	r.contentTransfer = r.trasferDone.Sub(r.transferStart)
	r.total = r.trasferDone.Sub(r.dnsStart)
}

// ContentTransfer returns the duration of content transfer time.
// It is from first response byte to the given time. The time must
// be time after read body (go-httpstat can not detect that time).
func (r *Result) ContentTransfer(t time.Time) time.Duration {
	return t.Sub(r.serverDone)
}

// Total returns the duration of total http request.
// It is from dns lookup start time to the given time. The
// time must be time after read body (go-httpstat can not detect that time).
func (r *Result) Total(t time.Time) time.Duration {
	return t.Sub(r.dnsStart)
}

func withClientTrace(ctx context.Context, r *Result) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.dnsStart = time.Now()
			r.t0 = r.dnsStart
		},

		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.dnsDone = time.Now()

			r.DNSLookup = r.dnsDone.Sub(r.dnsStart)
			r.NameLookup = r.dnsDone.Sub(r.dnsStart)
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

			r.TCPConnection = r.tcpDone.Sub(r.tcpStart)
			r.Connect = r.tcpDone.Sub(r.dnsStart)
		},

		TLSHandshakeStart: func() {
			r.isTLS = true
			r.tlsStart = time.Now()
		},

		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.tlsDone = time.Now()

			r.TLSHandshake = r.tlsDone.Sub(r.tlsStart)
			r.Pretransfer = r.tlsDone.Sub(r.dnsStart)
		},

		GotConn: func(i httptrace.GotConnInfo) {
			// Handle when keep alive is used and connection is reused.
			// DNSStart(Done) and ConnectStart(Done) is skipped
			if i.Reused {
				r.isReused = true
			}
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.serverStart = time.Now()

			// When client doesn't use DialContext or using old (before go1.7) `net`
			// pakcage, DNS/TCP/TLS hook is not called.
			if r.dnsStart.IsZero() && r.tcpStart.IsZero() {
				now := r.serverStart

				r.dnsStart = now
				r.dnsDone = now
				r.tcpStart = now
				r.tcpDone = now
			}

			// When connection is re-used, DNS/TCP/TLS hook is not called.
			if r.isReused {
				now := r.serverStart

				r.dnsStart = now
				r.dnsDone = now
				r.tcpStart = now
				r.tcpDone = now
				r.tlsStart = now
				r.tlsDone = now
			}

			if r.isTLS {
				return
			}

			r.TLSHandshake = r.tcpDone.Sub(r.tcpDone)
			r.Pretransfer = r.Connect
		},

		GotFirstResponseByte: func() {
			r.serverDone = time.Now()

			r.ServerProcessing = r.serverDone.Sub(r.serverStart)
			r.StartTransfer = r.serverDone.Sub(r.dnsStart)

			r.transferStart = r.serverDone
		},
	})
}
