# go-httpstat

[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[license]: https://github.com/tcnksm/go-httpstat/blob/master/LICENSE
[godocs]: http://godoc.org/github.com/tcnksm/go-httpstat

`go-httpstat` is a golang package to trace golang HTTP request latency (DNSLookup, TCP Connection and so on). Because it uses [`httptrace`](https://golang.org/pkg/net/http/httptrace/) internally, just creating `go-httpstat` powered `context` and giving it your `http.Request` kicks tracing (no big code modification is required). The original idea came from [`httpstat`](https://github.com/reorx/httpstat) command ( and Dave Cheney's [golang implementation](https://github.com/davecheney/httpstat)) 👏. This package now traces same latency infomation as them.

See usage and example on [GoDoc][godocs]. 

## Install 

Use `go get`,

```bash
$ go get github.com/tcnksm/go-httpstat
```

## Author

[Taichi Nakashima](https://github.com/tcnksm)
