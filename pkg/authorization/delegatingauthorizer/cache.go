package delegatingauthorizer

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

var (
	cacheTime = 5 * time.Second
)

type Cache struct {
	cache *lru.Cache[string, cacheEntry]
}

func NewCache() *Cache {
	cache, _ := lru.New[string, cacheEntry](256)
	return &Cache{
		cache: cache,
	}
}

type cacheEntry struct {
	authorized authorizer.Decision
	reason     string

	exp time.Time
}

func (c *Cache) Set(a authorizer.Attributes, authorized authorizer.Decision, reason string) {
	c.cache.Add(getCacheKey(a), cacheEntry{
		authorized: authorized,
		reason:     reason,
		exp:        time.Now().Add(cacheTime),
	})
}

func (c *Cache) Get(a authorizer.Attributes) (authorized authorizer.Decision, reason string, exists bool) {
	// check if in cache
	now := time.Now()
	entry, ok := c.cache.Get(getCacheKey(a))
	if ok && entry.exp.After(now) {
		return entry.authorized, entry.reason, true
	}

	return authorizer.DecisionNoOpinion, "", false
}

func getCacheKey(a authorizer.Attributes) string {
	parts := []string{}
	if a.GetUser() != nil {
		parts = append(parts, a.GetUser().GetName(), a.GetUser().GetUID(), strings.Join(a.GetUser().GetGroups(), ","))
	}
	if a.IsResourceRequest() {
		parts = append(parts, a.GetAPIGroup(), a.GetAPIVersion(), a.GetResource(), a.GetSubresource(), a.GetVerb(), a.GetNamespace(), a.GetName())
	} else {
		parts = append(parts, a.GetPath(), a.GetVerb())
	}

	// hash the string
	h := sha256.New()
	h.Write([]byte(strings.Join(parts, "#")))
	return hex.EncodeToString(h.Sum(nil))
}
