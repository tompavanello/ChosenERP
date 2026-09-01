package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// readJSON decodifica o corpo JSON da requisição com limite de tamanho.
func readJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
