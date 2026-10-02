// Command vapid-gen gera um par de chaves VAPID (Web Push) e imprime em
// formato .env:
//
//	go run ./cmd/vapid-gen
//
// Cole a saida no .env (VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY). As chaves NAO
// podem mudar depois: trocar invalida todas as inscricoes existentes.
package main

import (
	"fmt"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("gerar chaves: %v", err)
	}
	fmt.Println("VAPID_PUBLIC_KEY=" + publicKey)
	fmt.Println("VAPID_PRIVATE_KEY=" + privateKey)
	fmt.Println("VAPID_SUBJECT=mailto:contato@erpchosen.com.br")
}
