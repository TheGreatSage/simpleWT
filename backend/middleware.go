package backend

import (
	"log"
	"net/http"
	"slices"
	"time"
)

type Middleware func(http.Handler) http.Handler
type Chain []Middleware

func (c Chain) ThenFunc(h http.HandlerFunc) http.Handler {
	if h == nil {
		return c.Then(nil)
	}
	return c.Then(h)
}

func (c Chain) Then(h http.Handler) http.Handler {
	if h == nil {
		h = http.DefaultServeMux
	}
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}

func (c Chain) Append(mw ...Middleware) Chain {
	nChain := make([]Middleware, 0, len(c)+len(mw))
	nChain = append(nChain, c...)
	nChain = append(nChain, mw...)
	return nChain
}

// WithLogging Simple Log time
func WithLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s took %b", r.Method, r.URL.Path, time.Since(start))
	})
}

// WithRecovery catches panics in request handling
func WithRecovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				log.Printf("panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// WithCORS Adds basic cors headers
// TODO: Figure out CORS Headers
// Honestly don't know what i'm doing here.
func WithCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
