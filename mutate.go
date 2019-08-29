/*
 * Copyright (c) 2019 The Gonm Author
 *
 * File for mutate.
 */

package gonm

import (
	"fmt"
	"reflect"

	"cloud.google.com/go/datastore"
)

// Mutation is wrapper of datastore.Mutation
type Mutation struct {
	mutation *datastore.Mutation
	src      interface{}
	key      *datastore.Key
	err      error
}

// Mutate run GonMutations. If this method run success, all structures are complemented with IDs.
// In transaction, Mutate return nil as []*datastore.Key when success
func (gm *Gonm) Mutate(gmuts ...*Mutation) (ret []*datastore.Key, err error) {
	muts := make([]*datastore.Mutation, len(gmuts))
	var merr []error
	for i, gmut := range gmuts {
		if gmut.err != nil {
			merr = append(merr, gmut.err)
			gm.stackError(gmut.err)
		}
		muts[i] = gmut.mutation
	}

	if len(merr) > 0 {
		return nil, merr[0]
	}

	if gm.Transaction != nil {
		pret, err := gm.Transaction.Mutate(muts...)
		if err != nil {
			return nil, gm.stackError(err)
		}

		for i, key := range pret {
			if gmuts[i].key.Incomplete() {
				gm.m.Lock()
				gm.pending = append(gm.pending,
					&pendingStruct{
						pkey: key,
						dst:  gmuts[i].src,
					})
				gm.m.Unlock()
			} else {
				gm.cache.delete(gmuts[i].key)
			}
		}
		return nil, nil
	} else {
		ret, err = gm.Client.Mutate(gm.Context, muts...)
		if err != nil {
			return nil, gm.stackError(err)
		}

		for i, gmut := range gmuts {
			if gmut.key.Incomplete() {
				if err = setStructKey(gmut.src, ret[i]); err != nil {
					return ret, gm.stackError(err)
				}
			} else {
				gm.cache.delete(ret[i])
			}
		}
	}

	return ret, nil
}

// NewDelete generate Delete Mutation.
// Dst is required *S.
func NewDelete(dst interface{}) (gmut *Mutation) {
	gmut = &Mutation{}
	key, err := getStructKey(dst)
	if err != nil {
		gmut.err = err
		return gmut
	}

	gmut.src = dst
	gmut.key = key
	gmut.mutation = datastore.NewDelete(key)

	return gmut
}

// NewInsert generate Insert Mutation. Returning an error if k exist.
// Dst is required *S.
func NewInsert(dst interface{}) (gmut *Mutation) {
	gmut = &Mutation{}
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		gmut.err = fmt.Errorf("gonm: expected pointer to a struct, got %#v", dst)
		return gmut
	}
	key, err := getStructKey(dst)
	if err != nil {
		gmut.err = err
		return gmut
	}

	gmut.src = dst
	gmut.key = key
	gmut.mutation = datastore.NewInsert(key, dst)

	return gmut
}

// NewUpdate generate Update Mutation. Returning an error if k does not exist.
// Dst is required *S.
func NewUpdate(dst interface{}) (gmut *Mutation) {
	gmut = &Mutation{}
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		gmut.err = fmt.Errorf("gonm: expected pointer to a struct, got %#v", dst)
		return gmut
	}
	key, err := getStructKey(dst)
	if err != nil {
		gmut.err = err
		return gmut
	}

	gmut.src = dst
	gmut.key = key
	gmut.mutation = datastore.NewUpdate(key, dst)

	return gmut
}

// NewUpsert generate Upsert Mutation. Returning no error whether or not k exists.
// Dst is required *S.
func NewUpsert(dst interface{}) (gmut *Mutation) {
	gmut = &Mutation{}
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		gmut.err = fmt.Errorf("gonm: expected pointer to a struct, got %#v", dst)
		return gmut
	}
	key, err := getStructKey(dst)
	if err != nil {
		gmut.err = err
		return gmut
	}

	gmut.src = dst
	gmut.key = key
	gmut.mutation = datastore.NewUpsert(key, dst)

	return gmut
}
