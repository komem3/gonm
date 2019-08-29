/*
 * Copyright (c) 2019 The Gonm Author
 *
 * File for managing cache.
 */

package gonm

import (
	"sync"

	"cloud.google.com/go/datastore"
)

type cache struct {
	hashMap *sync.Map
}

// CacheClear clear local cache
func (gm *Gonm) CacheClear() {
	gm.cache.clear()
}

func newCache() *cache {
	return &cache{&sync.Map{}}
}

func (c *cache) delete(key *datastore.Key) {
	c.hashMap.Delete(key.Encode())
}

func (c *cache) get(key *datastore.Key) (value interface{}, ok bool) {
	return c.hashMap.Load(key.Encode())
}

func (c *cache) set(key *datastore.Key, value interface{}) {
	c.hashMap.Store(key.Encode(), value)
}

func (c *cache) clear() {
	c.hashMap = &sync.Map{}
}

// TODO: This file implements access to redis memcache database
