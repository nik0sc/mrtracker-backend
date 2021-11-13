package smrt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	endpointPlatform = "https://connectv3.smrt.wwprojects.com/smrt/api/train_arrival_time_by_platform/?platform="
)

func GetOnePlatform(ctx context.Context, maxTries int, platform string) (*NextTrains, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpointPlatform+url.QueryEscape(platform), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fakeUA)
	out := make(map[string]*NextTrains)

retryLoop:
	for maxTries > 0 {
		maxTries--
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		var buf []byte
		if resp.ContentLength > 0 {
			buf = make([]byte, resp.ContentLength)
			var n int
			n, err = resp.Body.Read(buf)
			if int64(n) != resp.ContentLength {
				err = errors.New("short response")
				continue
			}
			if err != nil {
				continue
			}
		} else {
			buf, err = io.ReadAll(resp.Body)
			if err != nil {
				continue
			}
		}

		err = resp.Body.Close()
		if err != nil {
			continue
		}

		//log.Printf(">> [%s] -> %s", platform, buf)

		err = json.Unmarshal(buf, &out)
		if err != nil {
			// sometimes the returned result is an array of NextTrains,
			// it's bogus in that case
			continue
		}

		if !out["results"].Valid() {
			err = fmt.Errorf("invalid response: %v", out["results"])
			time.Sleep(100 * time.Millisecond) // TODO configurable
			continue retryLoop
		}

		err = nil
		break
	}

	if err != nil {
		return nil, err
	} else {
		return out["results"], nil
	}
}

func GetNPlatform(ctx context.Context, numWorkers, maxTries int, platforms ...string) (map[string]*NextTrains, error) {
	// channel capacity is arbitrary
	workCh := make(chan string, numWorkers)
	resultCh := make(chan interface{}, numWorkers)
	var fanInOutWg, workerWg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerWg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer workerWg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case platform, ok := <-workCh:
					if !ok {
						return
					}
					result, err := GetOnePlatform(ctx, maxTries, platform)
					if err != nil {
						cancel()
						resultCh <- err
					} else {
						resultCh <- result
					}

				}
			}
		}()
	}

	fanInOutWg.Add(1)
	go func() {
		defer func() {
			fanInOutWg.Done()
			close(workCh)
		}()
		for _, platform := range platforms {
			select {
			case <-ctx.Done():
				return
			case workCh <- platform:
			}
		}
	}()

	fanInOutWg.Add(1)
	var err error
	out := make(map[string]*NextTrains)
	go func() {
		defer fanInOutWg.Done()

		// closed when all workers have exited
		for re := range resultCh {
			switch r := re.(type) {
			case *NextTrains:
				out[r.PlatformID] = r
			case error:
				err = r
			default:
				panic("wrong type on result channel")
			}
		}
	}()

	workerWg.Wait()
	close(resultCh)
	fanInOutWg.Wait()

	return out, err
}
