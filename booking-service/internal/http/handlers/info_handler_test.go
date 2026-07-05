package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfo_Returns200(t *testing.T) {
	h := newRouter(nil, nil, nil, nil, nil)

	for _, path := range []string{"/_info", "/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := doRequest(h, req)
		assert.Equal(t, http.StatusOK, rr.Code, "path=%s", path)
	}
}
