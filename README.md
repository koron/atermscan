# koron/atermscan

[![PkgGoDev](https://pkg.go.dev/badge/github.com/koron/atermscan)](https://pkg.go.dev/github.com/koron/atermscan)
[![Actions/Go](https://github.com/koron/atermscan/workflows/Go/badge.svg)](https://github.com/koron/atermscan/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/koron/atermscan)](https://goreportcard.com/report/github.com/koron/atermscan)

`aterscan` is a command to search for [Aterm devices](https://www.aterm.jp/product/atermstation/).
It is reimplementation of [Aterm search tool](https://www.aterm.jp/web/model/aterm_search.html) in Go.

## Getting Started

``` console
$ go install github.com/koron/atermscan@latest

$ atermscan
Address         Product Name    Mode
192.168.0.4     WG2600HP2       Bridge
192.168.0.3     WG2600HP2       Bridge
192.168.0.6     WG2600HP3       Bridge
192.168.0.5     WG2600HP3       Bridge
192.168.0.8     WG1200HS4       Wireless LAN Adapter
```
