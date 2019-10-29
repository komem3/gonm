/*
 * Copyright (c) 2019 The Gonm Author
 *
 * Gonm automatically assigns a key from the structure, and Gonm use datastore package with it.
 */

package gonm

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"cloud.google.com/go/datastore"
	"golang.org/x/sync/errgroup"
)

var datastorePutMultiMaxItems = 500

// Gonm is main struct
type Gonm struct {
	// Client store generated datastore.Client. Datastore.Client close when Gonm.Close.
	// In transaction, Client is nil.
	Client *datastore.Client

	// Transaction store generated datastore.Transaction.
	// In not transaction, Transaction is nil.
	Transaction *datastore.Transaction

	// Errors store occurred error in method of Gonm.
	Errors datastore.MultiError

	Context context.Context
	cache   *cache
	pending []*pendingStruct
	m       sync.Mutex
}

type pendingStruct struct {
	pkey *datastore.PendingKey
	dst  interface{}
}

// FromContext generate Gonm from Context.
func FromContext(ctx context.Context, dsClient *datastore.Client) *Gonm {
	return &Gonm{
		Context: ctx,
		Client:  dsClient,
		cache:   newCache(),
	}
}

// AllocateID is accepts a incomplete keys and
// returns a complete keys that are guaranteed to be valid in the datastore.
//
// Also, all structures are complemented with ID.
func (gm *Gonm) AllocateID(dst interface{}) (*datastore.Key, error) {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return nil, gm.stackError(fmt.Errorf("gonm: expected pointer to a struct, got %#v", dst))
	}
	keys, err := gm.AllocateIDs([]interface{}{dst})
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, gm.stackError(fmt.Errorf("gonm: not allocate id"))
	}
	return keys[0], nil
}

// AllocateIDs is accepts a slice of incomplete keys and
// returns a slice of complete keys that are guaranteed to be valid in the datastore.
//
// Also, all structures are complemented with IDs.
func (gm *Gonm) AllocateIDs(dst interface{}) ([]*datastore.Key, error) {
	if gm.Transaction != nil {
		return nil, gm.stackError(ErrInTransaction)
	}
	keys, err := extractKeys(dst, true)
	if err != nil {
		return nil, gm.stackError(err)
	}
	keys, err = gm.Client.AllocateIDs(gm.Context, keys)
	if err != nil {
		return nil, gm.stackError(err)
	}

	v := reflect.Indirect(reflect.ValueOf(dst))
	for i, key := range keys {
		if err := setStructKey(v.Index(i).Interface(), key); err != nil {
			return nil, gm.stackError(err)
		}
	}
	return keys, nil
}

// Close close the Gonm
func (gm *Gonm) Close() error {
	if gm.Transaction != nil {
		return fmt.Errorf("gonm: Close do not use in Transaction")
	}
	if err := gm.Client.Close(); err != nil {
		return err
	}
	gm.cache = nil
	gm.Errors = nil
	return nil
}

// Delete deletes the entity for the given *S.
func (gm *Gonm) Delete(dst interface{}) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return gm.stackError(fmt.Errorf("gonm: expected pointer to a struct, got %#v", dst))
	}
	err := gm.DeleteMulti([]interface{}{dst})
	if err != nil {
		if me, ok := err.(datastore.MultiError); ok {
			return me[0]
		}
		return err
	}
	return nil
}

// DeleteMulti deletes the entity for the given []*S or []S.
func (gm *Gonm) DeleteMulti(dst interface{}) error {
	keys, err := extractKeys(dst, false) // allow incomplete keys on a Put request
	if err != nil {
		return gm.stackError(err)
	}

	goroutines := (len(keys)-1)/datastorePutMultiMaxItems + 1
	var multiError datastore.MultiError

	var eg errgroup.Group
	for i := 0; i < goroutines; i++ {
		i := i
		eg.Go(func() error {
			lo := i * datastorePutMultiMaxItems
			hi := (i + 1) * datastorePutMultiMaxItems
			if hi > len(keys) {
				hi = len(keys)
			}

			for _, key := range keys[lo:hi] {
				gm.cache.delete(key)
			}

			var err error
			if gm.Transaction != nil {
				err = gm.Transaction.DeleteMulti(keys[lo:hi])
			} else {
				err = gm.Client.DeleteMulti(gm.Context, keys[lo:hi])
			}
			if err != nil {
				if merr, ok := err.(datastore.MultiError); ok {
					multiError = append(multiError, merr...)
				}
				return gm.stackError(err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

// Get loads the entity based on dst's key into dst.
//
// If there is no such entity for the key, get returns datastore.ErrNoSuchEntity.
// Dst must be a *S, and returning datastore.ErrFieldMissMatch if dst is not struct pointer
func (gm *Gonm) Get(dst interface{}) error {
	return gm.get(dst, true)
}

// GetConsistency is get method without cache.
func (gm *Gonm) GetConsistency(dst interface{}) error {
	return gm.get(dst, false)
}

func (gm *Gonm) get(dst interface{}, cache bool) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return gm.stackError(fmt.Errorf("gonm: expected pointer to a struct, got %#v", dst))
	}

	var err error
	if cache {
		err = gm.GetMulti([]interface{}{dst})
	} else {
		err = gm.GetMultiConsistency([]interface{}{dst})
	}

	if err != nil {
		if me, ok := err.(datastore.MultiError); ok {
			return me[0]
		}
		return err
	}

	return nil
}

// GetByKey is getting object from datastore by datastore.Key.
//
// Usage is almost the same as datastore.Client.Get.
// This method use cache.
func (gm *Gonm) GetByKey(key *datastore.Key, dst interface{}) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return gm.stackError(fmt.Errorf("gonm: expected pointer to a struct, got %#v", dst))
	}

	if err := gm.GetMultiByKeys([]*datastore.Key{key}, []interface{}{dst}); err != nil {
		if me, ok := err.(datastore.MultiError); ok {
			return me[0]
		}
		return err
	}

	return nil
}

// GetMulti is a batch version of Get.
//
// Dst must have type *[]S, *[]*S or *[]P.
func (gm *Gonm) GetMulti(dst interface{}) error {
	keys, err := extractKeys(dst, false)
	if err != nil {
		return gm.stackError(err)
	}
	return gm.GetMultiByKeys(keys, dst)
}

// GetMultiConsistency is GetMulti method without cache.
func (gm *Gonm) GetMultiConsistency(dst interface{}) error {
	keys, err := extractKeys(dst, false)
	if err != nil {
		return gm.stackError(err)
	}
	return gm.getMultiByKeysConsistency(keys, dst)
}

// GetMultiByKeys is getting object from datastore by datastore.Key.
//
// Usage is almost the same as datastore.Client.GetMulti.
// this method use cache
func (gm *Gonm) GetMultiByKeys(keys []*datastore.Key, dst interface{}) error {
	if gm.Transaction != nil {
		return gm.getMultiByKeysConsistency(keys, dst)
	}

	v := reflect.Indirect(reflect.ValueOf(dst))

	var getKeys []*datastore.Key
	var dstList []interface{}

	for i, key := range keys {
		vi := v.Index(i)

		if vi.Kind() == reflect.Struct {
			vi = vi.Addr()
		}

		if data, ok := gm.cache.get(key); ok {
			if vi.Kind() == reflect.Interface {
				vi = vi.Elem()
			}
			dv := reflect.ValueOf(data)
			// []interface{}{*S}, []S, []*S, []interface{}{*S[0]} are all different object
			switch {
			case vi.CanSet() && vi.Kind() == dv.Kind():
				vi.Set(dv)
			case !vi.CanSet() && dv.Kind() == reflect.Ptr:
				vi.Elem().Set(dv.Elem())
			case !vi.CanSet():
				vi.Elem().Set(dv)
			case vi.Kind() == reflect.Ptr && dv.Kind() != reflect.Ptr:
				vi.Elem().Set(dv)
			case vi.Kind() != reflect.Ptr && dv.Kind() == reflect.Ptr:
				vi.Set(dv.Elem())
			default:
				// this case should not occur
				return fmt.Errorf("gonm: unexpected cache comibnation")
			}
		} else {
			getKeys = append(getKeys, key)
			dstList = append(dstList, vi.Interface())
		}
	}

	return gm.getMultiByKeysConsistency(getKeys, dstList)
}

// getMultiByKeysConsistency is simple wrapper of datastore Client GetMulti
func (gm *Gonm) getMultiByKeysConsistency(keys []*datastore.Key, dst interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(dst))
	goroutines := (len(keys)-1)/datastorePutMultiMaxItems + 1
	var multiError datastore.MultiError

	var eg errgroup.Group
	for i := 0; i < goroutines; i++ {
		i := i
		eg.Go(func() error {
			lo := i * datastorePutMultiMaxItems
			hi := (i + 1) * datastorePutMultiMaxItems
			if hi > len(keys) {
				hi = len(keys)
			}

			var err error
			if gm.Transaction != nil {
				for _, key := range keys[lo:hi] {
					gm.cache.delete(key)
				}
				err = gm.Transaction.GetMulti(keys[lo:hi], v.Slice(lo, hi).Interface())
				if err != nil {
					if merr, ok := err.(datastore.MultiError); ok {
						multiError = append(multiError, merr...)
					}
					return gm.stackError(err)
				}
			} else {
				err = gm.Client.GetMulti(gm.Context, keys[lo:hi], v.Slice(lo, hi).Interface())
				if err != nil {
					if merr, ok := err.(datastore.MultiError); ok {
						multiError = append(multiError, merr...)
					}
					for _, key := range keys[lo:hi] {
						gm.cache.delete(key)
					}
					return gm.stackError(err)
				}

				for i, key := range keys[lo:hi] {
					vi := v.Index(lo + i).Interface()
					gm.cache.set(key, vi)
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		if len(multiError) > 0 {
			return multiError
		}
		return err
	}

	return nil
}

// Put method receive *S and put *S into datastore.
//
// If src has no ID filed, this method return ErrNoIDField.
// Also, the structure is complemented with ID after this method.
// This method return nil as *datastore.Key when success in Transaction.
func (gm *Gonm) Put(src interface{}) (*datastore.Key, error) {
	v := reflect.ValueOf(src)
	if v.Kind() != reflect.Ptr {
		return nil, gm.stackError(fmt.Errorf("gonm: expected pointer to a struct, got %#v", src))
	}
	ks, err := gm.PutMulti([]interface{}{src})
	if err != nil {
		if me, ok := err.(datastore.MultiError); ok {
			return nil, me[0]
		}
		return nil, err
	}
	return ks[0], nil
}

// PutMulti is a batch version of Put.
// Put receive []*S and put []*S into datastore.
//
// Also, all structures are complemented with IDs after this method.
func (gm *Gonm) PutMulti(src interface{}) ([]*datastore.Key, error) {
	keys, err := extractKeys(src, true) // allow incomplete keys on a Put request
	if err != nil {
		return nil, gm.stackError(err)
	}

	v := reflect.Indirect(reflect.ValueOf(src))
	goroutines := (len(keys)-1)/datastorePutMultiMaxItems + 1
	var multiError datastore.MultiError

	var eg errgroup.Group
	for i := 0; i < goroutines; i++ {
		i := i
		eg.Go(func() error {
			lo := i * datastorePutMultiMaxItems
			hi := (i + 1) * datastorePutMultiMaxItems
			if hi > len(keys) {
				hi = len(keys)
			}

			var (
				rkeys []*datastore.Key
				pkeys []*datastore.PendingKey
				err   error
			)
			if gm.Transaction != nil {
				pkeys, err = gm.Transaction.PutMulti(keys[lo:hi], v.Slice(lo, hi).Interface())
				if err != nil {
					if merr, ok := err.(datastore.MultiError); ok {
						multiError = append(multiError, merr...)
					}
					for _, key := range keys[lo:hi] {
						if !key.Incomplete() {
							gm.cache.delete(key)
						}
					}
					return gm.stackError(err)
				}

				for i, key := range keys[lo:hi] {
					vi := v.Slice(lo, hi)
					if key.Incomplete() {
						gm.m.Lock()
						gm.pending = append(gm.pending,
							&pendingStruct{
								pkey: pkeys[i],
								dst:  vi.Index(i).Interface(),
							})
						gm.m.Unlock()
					} else {
						gm.cache.delete(key)
					}
				}

			} else {
				rkeys, err = gm.Client.PutMulti(gm.Context, keys[lo:hi], v.Slice(lo, hi).Interface())
				if err != nil {
					if merr, ok := err.(datastore.MultiError); ok {
						multiError = append(multiError, merr...)
					}
					for _, key := range keys[lo:hi] {
						if !key.Incomplete() {
							gm.cache.delete(key)
						}
					}
					return gm.stackError(err)
				}

				for i, key := range keys[lo:hi] {
					vi := v.Index(lo + i).Interface()
					if key.Incomplete() {
						if err := setStructKey(vi, rkeys[i]); err != nil {
							return err
						}
						keys[lo+i] = rkeys[i]
					}
					gm.cache.set(rkeys[i], vi)
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		if len(multiError) > 0 {
			return keys, multiError
		}
		return keys, err
	}

	return keys, nil
}
