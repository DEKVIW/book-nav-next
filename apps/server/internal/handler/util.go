package handler

import (
	"encoding/json"
	"net/http"

	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
)

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if e, ok := apperr.As(err); ok {
		status := http.StatusBadRequest
		switch e.Code {
		case apperr.Unauthorized:
			status = http.StatusUnauthorized
		case apperr.Forbidden:
			status = http.StatusForbidden
		case apperr.NotFound:
			status = http.StatusNotFound
		case apperr.Conflict:
			status = http.StatusConflict
		case apperr.Internal:
			status = http.StatusInternalServerError
		case apperr.Validation:
			status = http.StatusBadRequest
		}
		response.Fail(w, status, string(e.Code), e.Message)
		return
	}
	response.Internal(w, err.Error())
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmtSscanf(v, &n); err != nil || n <= 0 {
		return def
	}
	return n
}

func fmtSscanf(s string, n *int) (int, error) {
	var x int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errInvalid
		}
		x = x*10 + int(ch-'0')
	}
	*n = x
	return 1, nil
}

var errInvalid = errStr("invalid")

type errStr string

func (e errStr) Error() string { return string(e) }
