/*
 * Copyright (c) 2019 The Gonm Author
 *
 * File for transaction
 */

package gonm

import (
	"context"
	"fmt"
	"reflect"

	"cloud.google.com/go/datastore"
	"golang.org/x/sync/errgroup"
)

type GonmTx struct {
	Transaction *datastore.Transaction
	Context     context.Context
	gonm        *Gonm
}

// RunInTransaction runs f in Transaction.
//
// Get, GetMulti, GetByKey, GetConsistency, GetMultiByKeys, GetMultiConsistency, GetPut, PutMulti, Delete, DeleteMulti, are only method that can be used.
// Also, Put and PutMulti in Gonm of Transaction do not return datastore.Key (return nil), but, all structures are complemented with IDs after transaction.
// If you want to get pending key, you should use NewTransaction or *Gonm.Transaction.Put(key, src).
func (gm *Gonm) RunInTransaction(f func(gm *Gonm) error, otps ...datastore.TransactionOption) (cmt *datastore.Commit, err error) {

	gmtx := &Gonm{Context: gm.Context}
	cmt, err = gm.Client.RunInTransaction(gm.Context, func(tx *datastore.Transaction) error {
		gmtx.Transaction = tx
		return f(gmtx)
	}, otps...)

	if err != nil {
		gm.Errors = append(gm.Errors, gmtx.Errors...)
		return nil, err
	}

	for _, key := range gmtx.tkeys {
		gm.cache.delete(key)
	}

	for _, pending := range gmtx.pending {
		key := cmt.Key(pending.pkey)
		gm.cache.delete(key)
		if err := setStructKey(pending.dst, key); err != nil {
			return cmt, gm.stackError(err)
		}
	}

	return cmt, nil
}

// NewTransaction starts a new Transaction.
// Get, GetMulti, GetByKey, GetMultiByKeys, GetPut, PutMulti, Delete, and DeleteMulti are only method that can be used.
func (gm *Gonm) NewTransaction(otps ...datastore.TransactionOption) (gmtx *GonmTx, err error) {
	if gm.Transaction != nil {
		return nil, gm.stackError(ErrInTransaction)
	}
	t, err := gm.Client.NewTransaction(gm.Context, otps...)
	if err != nil {
		return nil, err
	}
	return &GonmTx{Transaction: t, Context: gm.Context, gonm: &Gonm{Transaction: t, cache: gm.cache}}, nil
}

// Commit applies the enqueued operations atomically.
// Also, all cache is clear when this method run.
func (gmtx *GonmTx) Commit() (cm *datastore.Commit, err error) {
	cm, err = gmtx.Transaction.Commit()
	if err != nil {
		return nil, err
	}
	gmtx.gonm.cache.clear()
	return cm, nil
}

// Delete is similar as Gonm.Delete
func (gmtx *GonmTx) Delete(dst interface{}) error {
	return gmtx.gonm.Delete(dst)
}

// DeleteMulti is similar as Gonm.DeleteMulti
func (gmtx *GonmTx) DeleteMulti(dst interface{}) error {
	return gmtx.gonm.DeleteMulti(dst)
}

// Get is similar as Gonm.Get
func (gmtx *GonmTx) Get(dst interface{}) error {
	return gmtx.gonm.Get(dst)
}

// GetMulti is similar as Gonm.GetMulti
func (gmtx *GonmTx) GetMulti(dst interface{}) error {
	return gmtx.gonm.GetMulti(dst)
}

// Mutation is similar as Gonm.Mutation
func (gmtx *GonmTx) Mutate(gmuts ...*Mutation) (ret []*datastore.Key, err error) {
	return gmtx.gonm.Mutate(gmuts...)
}

// Put is similar as Gonm.Put, but this method return datastore.PendingKey.
//
// This method do not change incomple key to complete key.
// If you want to insert ID, you may use Commit.commit(pendingKey)
func (gmtx *GonmTx) Put(src interface{}) (*datastore.PendingKey, error) {
	v := reflect.ValueOf(src)
	if v.Kind() != reflect.Ptr {
		return nil, gmtx.gonm.stackError(fmt.Errorf("gonm: expected pointer to a struct, got %#v", src))
	}
	ks, err := gmtx.PutMulti([]interface{}{src})
	if err != nil {
		if me, ok := err.(datastore.MultiError); ok {
			return nil, me[0]
		}
		return nil, err
	}
	return ks[0], nil
}

// PutMulti is a bach version of Put.
func (gmtx *GonmTx) PutMulti(src interface{}) ([]*datastore.PendingKey, error) {
	keys, err := extractKeys(src, true) // allow incomplete keys on a Put request
	if err != nil {
		return nil, gmtx.gonm.stackError(err)
	}

	v := reflect.Indirect(reflect.ValueOf(src))
	goroutines := (len(keys)-1)/datastorePutMultiMaxItems + 1
	var pendingKeys []*datastore.PendingKey

	var eg errgroup.Group
	for i := 0; i < goroutines; i++ {
		i := i
		eg.Go(func() error {
			lo := i * datastorePutMultiMaxItems
			hi := (i + 1) * datastorePutMultiMaxItems
			if hi > len(keys) {
				hi = len(keys)
			}

			pkeys, err := gmtx.Transaction.PutMulti(keys[lo:hi], v.Slice(lo, hi).Interface())

			if err != nil {
				merr, ok := err.(datastore.MultiError)
				if !ok {
					return gmtx.gonm.stackError(err)
				}
				for _, err := range merr {
					err = gmtx.gonm.stackError(err)
				}
				return err
			}

			gmtx.gonm.m.Lock()
			defer gmtx.gonm.m.Unlock()
			pendingKeys = append(pendingKeys, pkeys...)
			return nil

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return pendingKeys, err
	}

	return pendingKeys, nil
}

// Rollback abandons a pending Transaction.
func (gmtx *GonmTx) Rollback() (err error) {
	return gmtx.Transaction.Rollback()
}
