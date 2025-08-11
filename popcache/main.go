package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/yzp0n/ncdn/httprps"
	"github.com/yzp0n/ncdn/popcache/cache"
	"github.com/yzp0n/ncdn/types"
)

var originURLStr = flag.String("originURL", "http://localhost:8888", "Origin server URL")
var listenAddr = flag.String("listenAddr", ":8889", "Address to listen on")
var nodeId = flag.String("nodeId", "unknown_node", "Name of the node")

func main() {
	flag.Parse()

	originURL, err := url.Parse(*originURLStr)
	if err != nil {
		log.Fatalf("Failed to parse origin URL %q: %v", *originURLStr, err)
	}

	start := time.Now()
	store := cache.NewCacheStore()

	mux := http.NewServeMux()
	rps := httprps.NewMiddleware(mux)
	http.Handle("/", rps)

	mux.HandleFunc("/statusz", func(w http.ResponseWriter, r *http.Request) {
		s := types.PoPStatus{
			Id:     *nodeId,
			Uptime: time.Since(start).Seconds(),
			Load:   rps.GetRPS(),
		}
		bs, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal PoP status: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(bs)
	})
	mux.HandleFunc("/latencyz", func(w http.ResponseWriter, r *http.Request) {
		// return 204
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if cacheData := store.Get(r); cacheData != nil && !cacheData.RequiresRevalidate(r) {
			log.Printf("Cache hit for %s", r.URL.String())

			w.WriteHeader(cacheData.Res.StatusCode)
			for k, values := range cacheData.Res.Header {
				for _, v := range values {
					w.Header().Add(k, v)
				}
			}
			_, _ = io.Copy(w, cacheData.Res.Body)

			return
		}

		rp := &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				r.SetXForwarded()
				r.Out.Header.Set("X-NCDN-PoPCache-NodeId", *nodeId)
				r.SetURL(originURL)
			},
			ModifyResponse: func(res *http.Response) error {
				cacheData, err := cache.New(res)
				if err != nil {
					log.Printf("Failed to create cache: %v", err)
					return err
				}

				if cacheData != nil {
					log.Printf("Cache miss for %s, caching response", r.URL.String())
					store.Put(r, cacheData)
				} else {
					log.Printf("Cache miss for %s, not caching response", r.URL.String())
				}
				return nil
			},
		}
		rp.ServeHTTP(w, r)
	})

	log.Printf("Listening on %s...", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
