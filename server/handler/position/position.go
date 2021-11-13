package position

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"go.lepak.sg/mrtracker-backend/data"
	"go.lepak.sg/mrtracker-backend/model"
	"go.lepak.sg/mrtracker-backend/smrt"
)

const (
	defaultUpdateInterval = 15 * time.Second
)

const (
	UpdateLive = iota
	UpdateRecorded
)

type handler struct {
	// sharedMap is the map of line names to position entries
	// the position entry contains some extra bookkeeping stuff
	sharedMap map[string]*entry

	// This is the context for update, when it's cancelled it will
	// cause the update goroutine to abandon all work and exit
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	tick     *time.Ticker
	interval time.Duration

	metrics *metrics
}

type entry struct {
	// rlocked by ServeHTTP, locked by update
	lock        sync.RWMutex
	position    model.Position
	data        []string
	lastUpdated time.Time
}

type result struct {
	Line        string `json:"line"`
	Positions   string `json:"positions"`
	LastUpdated uint64 `json:"last_updated"`
	Source      string `json:"source,omitempty"`
}

type machineResult struct {
	Data        []string `json:"data"`
	LastUpdated uint64   `json:"last_updated"`
	Source      string   `json:"source,omitempty"`
}

type NewParam struct {
	Ctx            context.Context
	UpdateInterval time.Duration
	Strategy       int
}

func New(p NewParam) (*handler, error) {
	if p.UpdateInterval == 0 {
		p.UpdateInterval = defaultUpdateInterval
	}
	if p.Ctx == nil {
		p.Ctx = context.Background()
	}

	h := &handler{
		sharedMap: make(map[string]*entry),
		tick:      time.NewTicker(p.UpdateInterval),
		interval:  p.UpdateInterval,
		metrics:   newMetrics(),
	}
	h.ctx, h.cancel = context.WithCancel(p.Ctx)

	for _, l := range data.GetLines() {
		h.sharedMap[l.Name] = &entry{}
	}

	h.sharedMap["dev_v1"] = &entry{}

	h.wg.Add(1)
	switch p.Strategy {
	case UpdateLive:
		go h.update()
	case UpdateRecorded:
		panic("not implemented")
	default:
		return nil, fmt.Errorf("unrecognized update strategy: %d", p.Strategy)
	}

	return h, nil
}

func MustNew(p NewParam) *handler {
	h, err := New(p)
	if err != nil {
		log.Panic(err)
	}
	return h
}

func (h *handler) update() {
	defer h.wg.Done()
	names := data.GetNames()
	running := true
	for running {
		func() {
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(h.ctx, h.interval)
			var err error

			defer func() {
				cancel()
				h.metrics.BgLatency.Observe(time.Since(startTime).Seconds())
				h.metrics.BgRequests.Inc()
				if err != nil {
					h.metrics.BgErrors.Inc()
				}

				select {
				case <-h.ctx.Done():
					log.Print("exiting update loop")
					running = false
				case <-h.tick.C:
				}
			}()

			results, err := smrt.GetN(ctx, 0, 10, names...)
			if err != nil {
				log.Printf("error: smrt scrape failed: %v", err)
				return
			}

			workingMap := make(map[string]model.Position)
			for _, l := range data.GetLines() {
				workingMap[l.Name] = smrt.ToModel(results, l.Line).ToPosition()
			}

			for _, l := range data.GetLines() {
				ent := h.sharedMap[l.Name]
				ent.lock.Lock()
				ent.position = workingMap[l.Name].Copy()
				ent.lastUpdated = time.Now()
				ent.lock.Unlock()
			}

			for k, v := range workingMap {
				workingMap[k] = v[:len(v)-1]
			}

			packed, err := model.PackBoardV1(workingMap)
			if err != nil {
				log.Printf("error: packing for dev v1: %v", err)
				return
			}

			packedHex := make([]string, len(packed))
			for i := range packedHex {
				packedHex[i] = fmt.Sprintf("%x", packed[i])
			}

			ent := h.sharedMap["dev_v1"]
			ent.lock.Lock()
			ent.data = packedHex // aliasing is ok, we are not retaining packedHex
			ent.lastUpdated = time.Now()
			ent.lock.Unlock()

			h.metrics.BgLastUpdated.SetToCurrentTime()
		}()
	}
}

func (h *handler) Stop() {
	h.cancel()
	h.wg.Wait()
	h.tick.Stop()
	// order?
}

func (h *handler) resultForDevV1() *machineResult {
	out := &machineResult{}

	ent := h.sharedMap["dev_v1"]
	if ent == nil {
		return nil
	}

	ent.lock.RLock()
	out.Data = ent.data
	out.LastUpdated = uint64(ent.lastUpdated.UnixNano() / 1000000)
	ent.lock.RUnlock()

	return out
}

func (h *handler) resultForDefault() []result {
	var out []result

	for _, l := range data.GetLines() {
		r := result{
			Line: l.Name,
		}

		ent := h.sharedMap[l.Name]
		if ent == nil {
			continue
		}

		ent.lock.RLock()
		r.Positions = ent.position.ToString()
		r.LastUpdated = uint64(ent.lastUpdated.UnixNano() / 1000000)
		ent.lock.RUnlock()
		out = append(out, r)
	}

	return out
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	var outEface interface{}
	var err error

	defer func() {
		h.metrics.Latency.Observe(time.Since(startTime).Seconds())
		h.metrics.Requests.Inc()
		if err != nil {
			h.metrics.Errors.Inc()
		}
	}()

	w.Header().Set("content-type", "application/json")

	format := r.URL.Query().Get("format")
	switch format {
	case "dev_v1":
		outEface = h.resultForDevV1()
	default:
		outEface = h.resultForDefault()
	}

	marshal, err := json.Marshal(outEface)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errstr := fmt.Sprintf("{\"error\":%q}", err.Error())
		_, err2 := w.Write([]byte(errstr))
		if err2 != nil {
			log.Printf("error: double fault in position handler: %v -> %v", err, err2)
		}
	} else {
		_, err = w.Write(marshal)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}
}
