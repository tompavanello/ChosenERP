package payments

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

// authKey normaliza a chave de cifragem de credenciais: usa a chave dedicada
// quando informada, senao deriva do JWT secret. Sempre 32 bytes (AES-256).
func authKey(configured, jwtSecret string) []byte {
	src := configured
	if src == "" {
		src = jwtSecret
	}
	sum := sha256.Sum256([]byte("chosenerp-payments:" + src))
	return sum[:]
}

// EncryptCredentials cifra o JSON de credenciais e devolve um jsonb `{"enc": "..."}`
// pronto para gravar na coluna `credentials`.
func EncryptCredentials(key []byte, raw []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nonce, nonce, raw, nil)
	wrapped := map[string]string{"enc": base64.StdEncoding.EncodeToString(sealed)}
	return json.Marshal(wrapped)
}

// DecryptCredentials abre o jsonb `{"enc": "..."}` devolvendo o JSON original.
// Valores que nao estao nesse formato sao devolvidos como estao (compatibilidade).
func DecryptCredentials(key []byte, stored []byte) ([]byte, error) {
	if len(stored) == 0 {
		return nil, nil
	}
	var wrapper struct {
		Enc string `json:"enc"`
	}
	if err := json.Unmarshal(stored, &wrapper); err != nil || wrapper.Enc == "" {
		return stored, nil
	}
	sealed, err := base64.StdEncoding.DecodeString(wrapper.Enc)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, errors.New("credenciais corrompidas")
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
