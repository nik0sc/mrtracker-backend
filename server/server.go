package server

import (
	"context"
	"log"
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.lepak.sg/mrtracker-backend/server/handler/position"
	"go.lepak.sg/mrtracker-backend/server/handler/status"
)

/*
TODO:
 - on-demand updating from smrt api
 - fallback to recorded positions if api failed
 - possibly scrape alternative apis (eg the sg busleh one, do they use their own proxy?)
*/

// StartHttp starts the http server. It blocks until the context is cancelled, then it will shut down the server.
// It will also start a secondary server to serve prometheus metrics. We could attach pprof, expvar etc to it.
// Obviously, in the reverse proxy config, only route requests to the first addr and not the second
func StartHttp(ctx context.Context, addr string, privAddr string) {
	wg := &sync.WaitGroup{}
	privMux := http.NewServeMux()
	privMux.Handle("/metrics", promhttp.Handler())

	privMux.HandleFunc("/debug/pprof/", pprof.Index)
	privMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	privMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	privMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	privMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	privSrv := &http.Server{
		Addr:    privAddr,
		Handler: privMux,
	}

	wg.Add(1)
	go func(wg *sync.WaitGroup, promSrv *http.Server) {
		defer wg.Done()
		err := promSrv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalf("prom handler: %v", err)
		}
	}(wg, privSrv)

	mux := http.NewServeMux()
	mux.Handle("/v1/position", position.MustNew(position.NewParam{
		Ctx:            ctx,
		UpdateInterval: 0, // default
		Strategy:       position.UpdateLive,
	}))
	mux.Handle("/v1/status", status.Handler{})
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

	err = privSrv.Shutdown(context.Background())
	if err != nil {
		log.Printf("error shutting down prom server: %v", err)
	}

	wg.Wait()
}
