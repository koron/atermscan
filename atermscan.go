package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/jackpal/gateway"
	"github.com/koron-go/atermsearch"
	"golang.org/x/sync/semaphore"
)

type Scanner struct {
	verbose       bool
	parallelNum   int
	deviceTimeout time.Duration
}

func parseAsPrefix(s string) (netip.Prefix, error) {
	prefix, err1 := netip.ParsePrefix(s)
	if err1 == nil {
		return prefix, nil
	}

	// Try as a netip.Addr.
	addr, err2 := netip.ParseAddr(s)
	if err2 != nil {
		// TODO: return the error which merged err1 and err2.
		return netip.Prefix{}, err1
	}

	// Convert a netip.Addr to a netip.Prefix taking into account IPv4 classes.
	if !addr.Is4() {
		return netip.Prefix{}, errors.New("an IPv6 address without prefix")
	}
	b := addr.As4()
	var bits int
	switch {
	case b[0] < 0x80:
		bits = 8
	case b[0] < 0xc0:
		bits = 16
	default:
		bits = 24
	}
	return addr.Prefix(bits)
}

func (s Scanner) Scan(ctx context.Context, cidrOrAddr string) (<-chan *atermsearch.Device, error) {
	if cidrOrAddr == "" {
		gateway, err := gateway.DiscoverGateway()
		if err != nil {
			return nil, fmt.Errorf("failed to determine the default gateway: %w", err)
		}
		cidrOrAddr = gateway.String()
		if s.verbose {
			slog.Info("discovered default gateway", "address", gateway)
		}
	}
	prefix, err := parseAsPrefix(cidrOrAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}
	if s.verbose {
		slog.Info("scanning...", "target", prefix)
	}
	ch := make(chan *atermsearch.Device)
	go s.scanAsync(ctx, prefix, ch)
	return ch, nil
}

func (s Scanner) semaphoreWeight() int64 {
	if s.parallelNum <= 0 {
		return 256
	}
	return int64(s.parallelNum)
}

func (s Scanner) scanAsync(ctx context.Context, prefix netip.Prefix, ch chan<- *atermsearch.Device) {
	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(s.semaphoreWeight())
	for ip := prefix.Addr(); prefix.Contains(ip); ip = ip.Next() {
		wg.Add(1)
		err := sem.Acquire(ctx, 1)
		if err != nil {
			slog.Warn("semaphore.Acquire failed", "ip", ip, "error", err)
			return
		}
		go func(ip netip.Addr) {
			s.search(ctx, ch, ip)
			sem.Release(1)
			wg.Done()
		}(ip)
	}
	wg.Wait()
	close(ch)
}

func (s Scanner) search(ctx0 context.Context, ch chan<- *atermsearch.Device, ip netip.Addr) {
	ctx, cancel := context.WithTimeout(ctx0, s.deviceTimeout)
	defer cancel()
	dev, err := atermsearch.Search(ctx, ip.String())
	if err != nil {
		if s.verbose {
			slog.Info("not aterm device", "ip", ip, "reason", err)
		}
		return
	}
	ch <- dev
}

const defaultTimeout = 5 * time.Second

func main() {
	var (
		verbose  bool
		timeout  time.Duration
		parallel int
		address  string
	)
	flag.BoolVar(&verbose, "verbose", false, `verbose message`)
	flag.DurationVar(&timeout, "timeout", defaultTimeout, `timeout in second per addresses`)
	flag.IntVar(&parallel, "parallel", 0, `maximum number of addresses to check simultaneously (default: 256)`)
	flag.StringVar(&address, "address", "", `an address or CIDR to scan (default: the network where the default gateway exists)`)
	flag.Parse()

	var scanner Scanner
	if verbose {
		scanner.verbose = verbose
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	scanner.deviceTimeout = timeout

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	devices, err := scanner.Scan(ctx, address)
	if err != nil {
		slog.Error("failed to start scan", "error", err)
		os.Exit(1)
	}

	const format = "%-15s\t%-15s\t%-24s\n"
	fmt.Printf(format, "Address", "Product Name", "Mode")
	for dev := range devices {
		fmt.Printf(format, dev.Address, dev.ProductName, dev.SystemMode.Name)
	}
	if err := ctx.Err(); err != nil {
		slog.Warn("scan is terminated", "error", err)
		os.Exit(2)
	}
}
