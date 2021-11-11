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
	"time"
)

const (
	endpointStation = "https://connectv3.smrt.wwprojects.com/smrt/api/train_arrival_time_by_id/?station="
)

func GetOne(ctx context.Context, maxTries int, station string) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpointStation+url.QueryEscape(station), nil)
	if err != nil {
		return nil, err
	}
	//req.Header.Set("User-Agent", fakeUA)
	out := make(map[string]Result)

retryLoop:
	for maxTries > 0 {
		maxTries--
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

	if err != nil {
		log.Printf("error from smrt: %s", err.Error())
		return nil, err
	} else {
		return out["results"], nil
	}
}

// GetN retrieves the station arrival data from SMRT's API.
// numWorkers is the number of worker goroutines to start (and therefore also the maximum number
// of requests that can be made concurrently).
// If numWorkers <= 0, it will be increased to len(stations).
// maxTries is the maximum number of times a request for one station will be made.
// If ctx expires while GetN is still querying the SMRT API, all work will be abandoned
// and the context error will be returned.
func GetN(ctx context.Context, numWorkers, maxTries int, stations ...string) (map[string]Result, error) {
	if numWorkers <= 0 {
		numWorkers = len(stations)
	}

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
				case station, ok := <-workCh:
					if !ok {
						return
					}
					result, err := GetOne(ctx, maxTries, station)
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

	return out, err
}
