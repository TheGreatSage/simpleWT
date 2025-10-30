package backend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var expected = []byte("test\n")

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(expected)
}

func countMiddleware(i int) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			str := fmt.Sprintf("Count: %d\n", i)
			_, _ = w.Write([]byte(str))
			h.ServeHTTP(w, r)
		})
	}
}

func TestEmptyChain(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	Chain{}.ThenFunc(defaultHandler).ServeHTTP(w, req)

	resp := w.Result()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code not 200: %d", resp.StatusCode)
	}

	if bytes.Compare(body, expected) != 0 {
		t.Errorf("body not test: %s", string(body))
	}
}

func TestNils(t *testing.T) {
	v := Chain{}.Then(nil)
	if v != http.DefaultServeMux {
		t.Errorf("then not default")
	}
	v = Chain{}.ThenFunc(nil)
	if v != http.DefaultServeMux {
		t.Errorf("thenfunc not default")
	}
}

func TestCountingChain(t *testing.T) {
	count1 := countMiddleware(1)
	count2 := countMiddleware(2)
	count3 := countMiddleware(3)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	chain := Chain{count1, count2, count3}.ThenFunc(defaultHandler)

	chain.ServeHTTP(w, req)
	resp := w.Result()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code not 200: %d", resp.StatusCode)
	}
}
