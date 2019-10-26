package gonm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"cloud.google.com/go/datastore"
)

func TestGonm_GetAll(t *testing.T) {

	ctx := context.Background()

	var err error
	gm := FromContext(ctx, testDsClient)

	putModel := []*testModel{
		{ID: 1, Name: "Michael"},
		{ID: 2, Name: "Tom"},
	}
	if _, err = gm.PutMulti(putModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}

	var getModel []testModel
	q := datastore.NewQuery(Kind(testModel{})).Limit(2)
	if _, err := gm.GetAll(q, &getModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}

	assert.Equal(t, 2, len(getModel), "gonm GetAll")
	assert.NotZero(t, getModel[0].ID, "gonm GetAll and complete ID")
	assert.NotZero(t, getModel[1].ID, "gonm GetAll and complete ID")
}

func TestGonm_GetKeysOnly(t *testing.T) {
	ctx := context.Background()

	var err error
	gm := FromContext(ctx, testDsClient)

	putModel := []*testModel{
		{ID: 1, Name: "Michael"},
		{ID: 2, Name: "Tom"},
		{ID: 3, Name: "Jack"},
		{ID: 4, Name: "Hanako"},
	}
	if _, err = gm.PutMulti(putModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}

	gm.CacheClear()

	q := datastore.NewQuery(Kind(testModel{})).Limit(2)

	keys, cursor, err := gm.GetKeysOnly(q)
	dst := make([]testModel, len(keys))
	if err := gm.GetMultiByKeys(keys, dst); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.NotEqual(t, "", dst[0].Name, "index 0, KeysOnly query and GetMultiByKey")
	assert.NotEqual(t, "", dst[1].Name, "index 1, KeysOnly query and GetMultiByKey")

	q = q.Start(cursor)
	keys, cursor, err = gm.GetKeysOnly(q)
	secondDst := make([]testModel, len(keys))
	if err := gm.GetMultiByKeys(keys, secondDst); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.NotEqual(t, dst, secondDst, "GetKeysOnly cursor set start")

	q = q.End(cursor)
	keys, _, err = gm.GetKeysOnly(q)
	thirdDst := make([]testModel, len(keys))
	if err := gm.GetMultiByKeys(keys, thirdDst); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.Equal(t, secondDst, thirdDst, "GetKeysOnly cursor set end")
}
