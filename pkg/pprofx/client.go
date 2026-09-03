package pprofx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/google/pprof/profile"
)

const HeapProfilePath = "/debug/pprof/heap"

type MemStatDelta *Delta[runtime.MemStats, uint64]

type Client struct {
	baseURL string
	client  *http.Client
}

func NewPProfClient(baseURL string, c *http.Client) (*Client, error) {
	if c == nil {
		c = defaultHTTPClient()
	}

	pprofClient := &Client{
		client:  c,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}

	if err := pprofClient.Probe(context.Background()); err != nil {
		return nil, fmt.Errorf(
			"pprofx: failed to execute probe request: %w", err,
		)
	}

	return pprofClient, nil
}

func (c *Client) Probe(ctx context.Context) error {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.baseURL+HeapProfilePath, nil,
	)
	if err != nil {
		return fmt.Errorf("pprofx: could not create request: %w", err)
	}

	resp, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("pprofx: could not execute request: %w", err)
	}

	defer func(c io.Closer) {
		_ = c.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pprofx: unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) HeapProfile(ctx context.Context) (Profile, error) {
	var pfl Profile

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.baseURL+HeapProfilePath, nil,
	)
	if err != nil {
		return pfl, fmt.Errorf("could not create request: %w", err)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return pfl, fmt.Errorf("could not execute request: %w", err)
	}

	defer func(c io.Closer) {
		_ = c.Close()
	}(res.Body)

	pfl.source, err = profile.Parse(res.Body)
	if err != nil {
		return pfl, fmt.Errorf(
			"could not parse binary profile response: %w", err,
		)
	}

	return pfl, nil
}

func (c *Client) MemStats(ctx context.Context) (runtime.MemStats, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.baseURL+HeapProfilePath+"?debug=1", nil,
	)
	if err != nil {
		return runtime.MemStats{}, fmt.Errorf(
			"could not create request: %w", err,
		)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return runtime.MemStats{}, fmt.Errorf("could not execute request: %w", err)
	}

	defer func(c io.Closer) {
		_ = c.Close()
	}(res.Body)

	return c.parseMemStats(res.Body)
}

func (c *Client) MemStatsDelta(ctx context.Context, md MemStatDelta) (MemStatDelta, error) {
	stats, err := c.MemStats(ctx)
	if err != nil {
		return nil, err
	}

	if md == nil {
		return &Delta[runtime.MemStats, uint64]{current: stats}, nil
	}

	md.prev, md.current = md.current, stats

	return md, nil
}

func (c *Client) parseMemStats(r io.Reader) (runtime.MemStats, error) {
	var (
		stats         runtime.MemStats
		foundMemStats bool
	)

	bufioReader := bufio.NewReader(r)

	for {
		lineBytes, err := bufioReader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			return stats, fmt.Errorf("could not read heap stats: %w", err)
		}

		if bytes.HasPrefix(lineBytes, []byte("# runtime.MemStats")) {
			foundMemStats = true

			break
		}
	}

	if !foundMemStats {
		return stats, errors.New("could not find runtime memstats")
	}

	parsers := []func(val []byte) error{
		func(val []byte) error { return parseUintVal(val, &stats.Alloc) },
		func(val []byte) error { return parseUintVal(val, &stats.TotalAlloc) },
		func(val []byte) error { return parseUintVal(val, &stats.Sys) },
		func(val []byte) error { return parseUintVal(val, &stats.Lookups) },
		func(val []byte) error { return parseUintVal(val, &stats.Mallocs) },
		func(val []byte) error { return parseUintVal(val, &stats.Frees) },
		func(val []byte) error { return parseUintVal(val, &stats.HeapAlloc) },
		func(val []byte) error { return parseUintVal(val, &stats.HeapSys) },
		func(val []byte) error { return parseUintVal(val, &stats.HeapIdle) },
		func(val []byte) error { return parseUintVal(val, &stats.HeapInuse) },
		func(val []byte) error { return parseUintVal(val, &stats.HeapReleased) },
		func(val []byte) error { return parseUintVal(val, &stats.HeapObjects) },
		func(val []byte) error { return parseUintRatio(val, &stats.StackInuse, &stats.StackSys) },
		func(val []byte) error { return parseUintRatio(val, &stats.MSpanInuse, &stats.MSpanSys) },
		func(val []byte) error { return parseUintRatio(val, &stats.MCacheInuse, &stats.MCacheSys) },
		func(val []byte) error { return parseUintVal(val, &stats.BuckHashSys) },
		func(val []byte) error { return parseUintVal(val, &stats.GCSys) },
		func(val []byte) error { return parseUintVal(val, &stats.OtherSys) },
		func(val []byte) error { return parseUintVal(val, &stats.NextGC) },
		func(val []byte) error { return parseUintVal(val, &stats.LastGC) },
		func(val []byte) error { return parseUintArray(val, &stats.PauseNs) },
		func(val []byte) error { return parseUintArray(val, &stats.PauseEnd) },
		func(val []byte) error { return parseUintVal(val, &stats.NumGC) },
		func(val []byte) error { return parseUintVal(val, &stats.NumForcedGC) },
		func(val []byte) error { return parseFloatVal(val, &stats.GCCPUFraction) },
	}

	for idx := range parsers {
		lineBytes, err := bufioReader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return stats, fmt.Errorf("could not read heap stats: %w", err)
		}
		parser := parsers[idx]

		if err = parser(lineBytes[:len(lineBytes)-1]); err != nil {
			return stats, fmt.Errorf("could not map heap stats: %w", err)
		}
	}

	return stats, nil
}

func parseUintVal(rawValue []byte, target any) error {
	value, prefixFound := trimPrefix(rawValue)
	if !prefixFound {
		return fmt.Errorf(
			"could not parse %q as uint value: invalid raw value", rawValue,
		)
	}

	return unmarshalUint(value, target)
}

func parseFloatVal(rawValue []byte, target any) error {
	value, prefixFound := trimPrefix(rawValue)
	if !prefixFound {
		return fmt.Errorf(
			"could not parse %q as uint value: invalid raw value", rawValue,
		)
	}

	return unmarshalFloat(value, target)
}

func parseUintRatio(rawValue []byte, current, maxVal any) error {
	value, prefixFound := trimPrefix(rawValue)
	if !prefixFound {
		return fmt.Errorf(
			"could not parse %q as uint value: invalid raw value", rawValue,
		)
	}

	middleIdx := bytes.IndexByte(value, '/')
	if middleIdx == -1 || middleIdx+2 >= len(value) || middleIdx-1 <= 0 {
		return fmt.Errorf(
			"could not parse %q as ration value: invalid raw value", value,
		)
	}

	if err := unmarshalUint(value[:middleIdx-1], current); err != nil {
		return err
	}

	return unmarshalUint(value[middleIdx+2:], maxVal)
}

func parseUintArray(rawValue []byte, target *[256]uint64) error {
	value, prefixFound := trimPrefix(rawValue)
	if !prefixFound {
		return fmt.Errorf(
			"could not parse %q as uint value: invalid raw value", rawValue,
		)
	}

	if value[0] != '[' || value[len(value)-1] != ']' {
		return fmt.Errorf(
			"could not parse %q as uint array value: unexpected format",
			value,
		)
	}

	rawValues := bytes.Split(value[1:len(value)-1], []byte{' '})

	for i := range rawValues {
		if err := unmarshalUint(rawValues[i], &target[i]); err != nil {
			return err
		}
	}

	return nil
}

func trimPrefix(rawValue []byte) ([]byte, bool) {
	prefixEndIdx := bytes.IndexByte(rawValue, '=')
	if prefixEndIdx == -1 || prefixEndIdx+2 >= len(rawValue) {
		return nil, false
	}

	return rawValue[prefixEndIdx+2:], true
}

func unmarshalUint(rawValue []byte, target any) error {
	uintValue, err := strconv.ParseUint(
		unsafe.String(&rawValue[0], len(rawValue)), 10, 64,
	)
	if err != nil {
		return fmt.Errorf("could not parse uint value: %w", err)
	}

	switch v := target.(type) {
	case *uint64:
		*v = uintValue
	case *uint32:
		//nolint:gosec
		*v = uint32(uintValue)
	default:
		return fmt.Errorf("unexpected target type: %T", target)
	}

	return nil
}

func unmarshalFloat(rawValue []byte, target any) error {
	floatValue, err := strconv.ParseFloat(
		unsafe.String(&rawValue[0], len(rawValue)), 64,
	)
	if err != nil {
		return fmt.Errorf("could not parse uint value: %w", err)
	}

	switch v := target.(type) {
	case *float64:
		*v = floatValue
	default:
		return fmt.Errorf(
			"unexpected target type: expected *float64: received %T", target,
		)
	}

	return nil
}

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}
