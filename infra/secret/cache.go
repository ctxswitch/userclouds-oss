package secret

import (
	"sync"
	"time"
)

// TODO: Get rid of the package level cache
var c = &cache{secrets: map[string]cacheObject{}}
var secretCacheDuration = time.Hour * 24

type cacheObject struct {
	Secret  string
	Expires time.Time
}

// cache is an in-memory cache of secrets
type cache struct {
	secrets      map[string]cacheObject
	secretsMutex sync.RWMutex
}

func (c *cache) Get(loc string) (string, bool) {
	c.secretsMutex.RLock()
	defer c.secretsMutex.RUnlock()

	co, ok := c.secrets[loc]
	if ok && time.Now().UTC().Before(co.Expires) {
		return co.Secret, true
	}

	delete(c.secrets, loc)
	return "", false
}

func (c *cache) Store(loc string, secret string) {
	c.secretsMutex.Lock()
	defer c.secretsMutex.Unlock()

	c.secrets[loc] = cacheObject{
		secret,
		time.Now().UTC().Add(secretCacheDuration),
	}
}

func (c *cache) Reset() {
	c.secrets = map[string]cacheObject{}
}
