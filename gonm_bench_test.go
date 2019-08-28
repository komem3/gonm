package gonm

import (
	"context"
	"testing"

	"golang.org/x/sync/errgroup"

	"cloud.google.com/go/datastore"
)

const modelNum = 10000

func setupModel(prepare bool) []*testModel {
	test := make([]*testModel, modelNum)
	for i := 0; i < modelNum; i++ {
		if prepare {
			test[i] = &testModel{ID: int64(i + 1), Name: string(i)}
		} else {
			test[i] = &testModel{ID: int64(i + 1)}
		}
	}
	return test
}

func BenchmarkSimpleGet(b *testing.B) {
	ctx := context.Background()

	gm, err := FromContext(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer gm.Close()

	var keys []*datastore.Key
	putModel := setupModel(true)
	for i := 0; i < len(putModel); i++ {
		keys = append(keys, datastore.IDKey("test", int64(i+1), nil))
	}

	goroutines := (len(keys)-1)/500 + 1
	for i := 0; i < goroutines; i++ {
		lo := i * 500
		hi := (i + 1) * 500
		if hi > len(keys) {
			hi = len(keys)
		}
		if _, err = gm.Client.PutMulti(gm.Context, keys[lo:hi], putModel[lo:hi]); err != nil {
			b.Fatal(err)
		}
	}

	getModel := setupModel(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < len(getModel); i++ {
			keys[i] = datastore.IDKey("test", int64(i+1), nil)
		}
		if err = gm.Client.GetMulti(gm.Context, keys, getModel); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGonm_GetConsistency(b *testing.B) {
	ctx := context.Background()

	gm, err := FromContext(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer gm.Close()

	putModel := setupModel(true)

	if _, err = gm.PutMulti(putModel); err != nil {
		b.Fatal(err)
	}

	getModel := setupModel(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err = gm.GetMultiConsistency(getModel); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGonm_Get(b *testing.B) {
	ctx := context.Background()

	gm, err := FromContext(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer gm.Close()

	putModel := setupModel(true)

	if _, err = gm.PutMulti(putModel); err != nil {
		b.Fatal(err)
	}

	getModel := setupModel(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err = gm.GetMulti(getModel); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSimplePut(b *testing.B) {
	ctx := context.Background()

	gm, err := FromContext(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer gm.Close()

	var eg errgroup.Group
	putModel := setupModel(true)

	b.ResetTimer()
	goroutines := (len(putModel)-1)/500 + 1
	for i := 0; i < goroutines; i++ {
		i := i
		eg.Go(func() error {
			var keys []*datastore.Key
			lo := i * 500
			hi := (i + 1) * 500
			if hi > len(putModel) {
				hi = len(putModel)
			}
			for i := lo; i < hi; i++ {
				keys = append(keys, datastore.IDKey("test", int64(i+1), nil))
			}
			if _, err = gm.Client.PutMulti(gm.Context, keys, putModel[lo:hi]); err != nil {
				return err
			}
			return nil
		})
	}

	if err = eg.Wait(); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkGonm_Put(b *testing.B) {
	ctx := context.Background()

	gm, err := FromContext(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer gm.Close()

	putModel := setupModel(true)

	b.ResetTimer()
	if _, err = gm.PutMulti(putModel); err != nil {
		b.Fatal(err)
	}
}
