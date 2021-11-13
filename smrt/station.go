package smrt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

const (
	endpointStation = "https://connectv3.smrt.wwprojects.com/smrt/api/train_arrival_time_by_id/?station="
)

// GetOne simply gets the arrival info for the named station.
// maxTries must be at least 1. If the context is cancelled the request is abandoned.
// The NextTrains array is returned along with the number of tries actually taken.
func GetOne(ctx context.Context, maxTries int, station string) (Result, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpointStation+url.QueryEscape(station), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", fakeUA)
	out := make(map[string]Result)

	triesLeft := maxTries

retryLoop:
	for triesLeft > 0 {
		triesLeft--
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		var buf []byte
		//if resp.ContentLength > 0 {
		//	buf = make([]byte, resp.ContentLength)
		//	var n int
		//	n, err = resp.Body.Read(buf)
		//	if int64(n) != resp.ContentLength {
		//		err = errors.New("short response")
		//		continue
		//	}
		//	if err != nil {
		//		continue
		//	}
		//} else {
		buf, err = io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		//}

		err = resp.Body.Close()
		if err != nil {
			continue
		}

		if resp.StatusCode == 404 {
			// special case: treat this as a recoverable error
			// sometimes the api will return 404 but on subsequent
			// requests the data is returned normally
			// maybe their load balancer is pointing to a stale instance?
			err = fmt.Errorf("404: %s", station)
			time.Sleep(100 * time.Millisecond) // TODO configurable
			continue
		} else if resp.StatusCode != 200 {
			err = fmt.Errorf("unrecoverable error code [%d]: %s", resp.StatusCode, station)
			break
		}

		err = json.Unmarshal(buf, &out)
		if err != nil {
			continue
		}

		for i, p := range out["results"] {
			if !p.Valid() {
				err = fmt.Errorf("invalid response: [%d] %v", i, p)
				time.Sleep(100 * time.Millisecond) // TODO configurable
				continue retryLoop
			}
		}

		err = nil
		break
	}

	tries := maxTries - triesLeft

	if err != nil {
		log.Printf("error from smrt: %s", err.Error())
		return nil, tries, err
	} else {
		return out["results"], tries, nil
	}
}

// GetN retrieves the station arrival data from SMRT's API.
// numWorkers is the number of worker goroutines to start (and therefore also the maximum number
// of requests that can be made concurrently).
// If numWorkers <= 0, it will be increased to len(stations).
// maxTries is the maximum number of times a request for one station will be made.
// If ctx expires while GetN is still querying the SMRT API, all work will be abandoned
// and the context error will be returned.
// Along with the station arrival data, a count of the total number of network requests made
// is returned.
func GetN(ctx context.Context, numWorkers, maxTries int, stations ...string) (map[string]Result, int64, error) {
	if numWorkers <= 0 {
		numWorkers = len(stations)
	}

	// channel capacity is arbitrary
	workCh := make(chan string, numWorkers)
	resultCh := make(chan interface{}, numWorkers)
	var fanInOutWg, workerWg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var totalTries int64

	workerWg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer workerWg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case station, ok := <-workCh:
					if !ok {
						return
					}
					result, tries, err := GetOne(ctx, maxTries, station)
					atomic.AddInt64(&totalTries, int64(tries))
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
		for _, station := range stations {
			select {
			case <-ctx.Done():
				return
			case workCh <- station:
			}
		}
	}()

	fanInOutWg.Add(1)
	var err error
	out := make(map[string]Result)
	go func() {
		defer fanInOutWg.Done()

		// closed when all workers have exited
		for re := range resultCh {
			switch r := re.(type) {
			case Result:
				// assumes that response mrt field is the same as input station param
				out[r[0].Mrt] = r
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

	return out, totalTries, err
}
