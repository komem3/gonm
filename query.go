/*
 * Copyright (c) 2019 The Gonm Author
 *
 * File for query
 */

package gonm

import (
	"reflect"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

// Run runs the given query.
// If Transaction gonm use this method, return ErrInTransaction
func (gm *Gonm) Run(q *datastore.Query) (*datastore.Iterator, error) {
	if gm.Transaction != nil {
		return nil, gm.stackError(ErrInTransaction)
	}
	return gm.Client.Run(gm.Context, q), nil
}

// GetAll runs the provided query and returns all keys that match that query,
// as well as appending the values to dst.
func (gm *Gonm) GetAll(q *datastore.Query, dst interface{}) (keys []*datastore.Key, err error) {
	if gm.Transaction != nil {
		return nil, gm.stackError(ErrInTransaction)
	}

	keys, err = gm.Client.GetAll(gm.Context, q, dst)
	if err != nil {
		return nil, gm.stackError(err)
	}

	// query get no much object or keysOnly query
	if len(keys) == 0 || dst == nil {
		return keys, err
	}

	v := reflect.Indirect(reflect.ValueOf(dst))
	// This conditional expression is insurance
	if v.Len() != len(keys) {
		return keys, nil
	}

	for i, key := range keys {
		vi := v.Index(i)
		if vi.Kind() == reflect.Struct {
			vi = vi.Addr()
		}
		if err = setStructKey(vi.Interface(), key); err != nil {
			return keys, gm.stackError(err)
		}
	}
	return keys, nil
}

// GetKeysOnly run q.KeysOnly().
//
// this method return key and cursor. That`s why assuming that combining this method with GetByKey,
func (gm *Gonm) GetKeysOnly(q *datastore.Query) (keys []*datastore.Key, cursor datastore.Cursor, err error) {
	if gm.Transaction != nil {
		return nil, datastore.Cursor{}, gm.stackError(ErrInTransaction)
	}

	t := gm.Client.Run(gm.Context, q.KeysOnly())
	for {
		key, err := t.Next(nil)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return keys, datastore.Cursor{}, gm.stackError(err)
		}
		keys = append(keys, key)
	}

	cursor, err = t.Cursor()
	if err != nil {
		return keys, cursor, gm.stackError(err)
	}
	return keys, cursor, nil
}
