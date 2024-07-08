package testutils

import (
	"net/http"
	"testing"
)

// LogNotFoundHandler returns a http.HandlerFunc that returns a 404 response and logs the request to the test log.
func LogNotFoundHandler(t *testing.T) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		t.Logf("Unregistered request (404): %s %s", req.Method, req.URL.Path)
		res.WriteHeader(http.StatusNotFound)
		_, _ = res.Write([]byte(`not found`))
	}
}
