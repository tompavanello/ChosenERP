package httpapi

import (
	"strings"
	"sync"
	"time"
)

// ttlCache e um cache em memoria com expiracao por entrada. Serve para evitar
// uma transacao por request apenas para ler entitlement de plano e permissoes
// do papel (dados que mudam raramente).
//
// O estado e local ao processo. Com uma unica instancia da API isso e o
// suficiente; se um dia houver mais replicas, o TTL curto limita a defasagem.
type ttlCache[V any] struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]ttlItem[V]
}

type ttlItem[V any] struct {
	value   V
	expires time.Time
}

func newTTLCache[V any](ttl time.Duration) *ttlCache[V] {
	return &ttlCache[V]{ttl: ttl, items: make(map[string]ttlItem[V])}
}

func (c *ttlCache[V]) Get(key string) (V, bool) {
	var zero V
	if c == nil {
		return zero, false
	}
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expires) {
		if ok {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return zero, false
	}
	return item.value, true
}

func (c *ttlCache[V]) Set(key string, value V) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.items[key] = ttlItem[V]{value: value, expires: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

// DeletePrefix invalida entradas de um tenant (ex.: mudanca de plano/papel).
func (c *ttlCache[V]) DeletePrefix(prefix string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	for k := range c.items {
		if strings.HasPrefix(k, prefix) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// Clear esvazia o cache inteiro (ex.: alteracao no catalogo de planos).
func (c *ttlCache[V]) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.items = make(map[string]ttlItem[V])
	c.mu.Unlock()
}
