package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const (
	maxTaskJSONBodyBytes     int64 = 256 * 1024
	maxSettingsJSONBodyBytes int64 = 64 * 1024
	maxMessageJSONBodyBytes  int64 = 128 * 1024
	maxDingTalkJSONBodyBytes int64 = 64 * 1024
)

func decodeLimitedJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64, badMessage string) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "请求体过大")
			return false
		}
		writeError(w, http.StatusBadRequest, badMessage)
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, badMessage)
		return false
	}
	return true
}
