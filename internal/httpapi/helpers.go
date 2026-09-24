package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// readJSON decodifica o corpo JSON da requisicao com limite de tamanho.
func readJSON(r *http.Request, v any) error {
	return readJSONMax(r, v, 1<<20)
}

// readJSONMax e o readJSON com limite configuravel (ex.: importacao de planilha).
func readJSONMax(r *http.Request, v any, max int64) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, max))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
