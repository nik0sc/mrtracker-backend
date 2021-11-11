package server

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.lepak.sg/mrtracker-backend/server/handler/position"
)

/*
TODO:
 - on-demand updating from smrt api
 - fallback to recorded positions if api failed
*/

// StartHttp starts the http server. It blocks until the context is cancelled, then it will shut down the server.
// It will also start a secondary server to serve prometheus metrics. We could attach pprof, expvar etc to it.
// Obviously, in the reverse proxy config, only route requests to the first addr and not the second
func StartHttp(ctx context.Context, addr string, promAddr string) {
	wg := &sync.WaitGroup{}
	promMux := http.NewServeMux()
	promMux.Handle("/metrics", promhttp.Handler())
	promSrv := &http.Server{
		Addr:    promAddr,
		Handler: promMux,
	}
	wg.Add(1)
	go func(wg *sync.WaitGroup, promSrv *http.Server) {
		defer wg.Done()
		err := promSrv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalf("prom handler: %v", err)
		}
	}(wg, promSrv)

	mux := http.NewServeMux()
	mux.Handle("/v1/position", position.MustNew(position.NewParam{
		Ctx:            ctx,
		UpdateInterval: 0, // default
		Strategy:       position.UpdateLive,
	}))
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	wg.Add(1)
	go func(wg *sync.WaitGroup, srv *http.Server) {
		defer wg.Done()
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalf("main handler: %v", err)
		}
	}(wg, srv)

	// block here
	<-ctx.Done()

	err := srv.Shutdown(context.Background())
	if err != nil {
		log.Printf("error shutting down main server: %v", err)
	}

	err = promSrv.Shutdown(context.Background())
	if err != nil {
		log.Printf("error shutting down prom server: %v", err)
	}

	wg.Wait()
}
