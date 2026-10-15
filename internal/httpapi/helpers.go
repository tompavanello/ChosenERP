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
//
// Nao usamos DisallowUnknownFields: varios endpoints fazem round-trip GET->PUT
// (o cliente devolve o objeto recebido, que pode ter mais campos do que o
// subconjunto aceito na escrita), e a rejeicao por campo extra viraria 400.
func readJSONMax(r *http.Request, v any, max int64) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, max))
	return dec.Decode(v)
}
