package gonm

import (
	"context"
	"testing"

	"cloud.google.com/go/datastore"
	"github.com/stretchr/testify/assert"
)

type testModel struct {
	ID   int64 `datastore:"-"`
	Name string
}

func TestGonm_AllocateIDs(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	var err error
	gm := FromContext(ctx, testDsClient)

	test := []*testModel{
		{Name: "Michel"},
		{Name: "Hanako"},
	}
	keys, err := gm.AllocateIDs(test)
	if err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.False(keys[0].Incomplete(), "pkey[0] complete pkey")
	assert.False(keys[1].Incomplete(), "pkey[1] complete pkey")
	assert.NotZero(test[0].ID, "test[0].ID is not zero")
	assert.NotZero(test[1].ID, "test[1].ID is not zero")

	singleTest := &testModel{Name: "Hanako"}
	key, err := gm.AllocateID(singleTest)
	if err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.False(key.Incomplete(), "pkey[0] complete pkey")
	assert.NotZero(singleTest.ID, "test[0].ID is not zero")
}

func TestGonm_DeleteMulti(t *testing.T) {
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
	if err = gm.DeleteMulti(putModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}

	err = gm.GetMulti(putModel)
	if me, ok := err.(datastore.MultiError); ok {
		assert.Error(t, datastore.ErrNoSuchEntity, me[0], "delete entry no such entry")
		assert.Error(t, datastore.ErrNoSuchEntity, me[1], "delete entry no such entry")
	}
}

func TestGonm_PutGet(t *testing.T) {
	ctx := context.Background()

	var err error
	gm := FromContext(ctx, testDsClient)

	putModel := &testModel{Name: "Tom"}
	if _, err = gm.Put(putModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.NotZero(t, putModel.ID, "complete ID")

	putModel = &testModel{ID: 1, Name: "Michael"}
	if _, err = gm.Put(putModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}

	getModel := &testModel{ID: 1}
	if err := gm.Get(getModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.Equal(t, putModel, getModel, "single put get")

	getModel = &testModel{ID: 1}
	if err := gm.GetConsistency(getModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.Equal(t, putModel, getModel, "single put consist get")

	getModel = []*testModel{
		{ID: 1, Name: "Michael"},
		{ID: 2, Name: "Tom"},
	}[0]
	if err := gm.Get(getModel); err != nil {
		t.Fatal(gm.printStackErrs(err))
	}
	assert.Equal(t, putModel, getModel, "single put get")
}

func TestGonm_PutGetMulti(t *testing.T) {
	ctx := context.Background()

	var err error
	gm := FromContext(ctx, testDsClient)

	t.Run("get multi stackError", func(t *testing.T) {
		var largeModel []*testModel
		for i := 1; i <= 1001; i++ {
			largeModel = append(largeModel, &testModel{ID: int64(i * 1000)})
		}
		err = gm.GetMulti(largeModel)
		assert.Errorf(t, err, "%s (and %d other errors)", datastore.ErrNoSuchEntity, 1000)

		merr, ok := err.(datastore.MultiError)
		if !ok {
			t.Fatalf("error is not MultiError")
		}
		for _, err := range merr {
			assert.Error(t, datastore.ErrNoSuchEntity, err, "error is ErrNoSuchEntity")
		}
	})

	t.Run("addr put addr get", func(t *testing.T) {
		putModel := []*testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
			{Name: "Jack"},
		}

		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		putModel = putModel[:len(putModel)-1]

		getModel := []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel, getModel, "gostore GetMulti")

		if err = gm.GetMultiConsistency(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel, getModel, "gostore GetMulti")

		getModel = []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel, getModel, "gostore GetMulti")
	})

	t.Run("addr put struct get", func(t *testing.T) {
		putModel := []*testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
			{Name: "Jack"},
		}

		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		getModel := []testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")

		if err = gm.GetMultiConsistency(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")

		getModel = []testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")
	})

	t.Run("struct put addr get", func(t *testing.T) {
		t.Skip()
		putModel := []testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
			{Name: "Jack"},
		}

		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		getModel := []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")

		if err = gm.GetMultiConsistency(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")

		getModel = []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")
	})

	t.Run("struct put struct get", func(t *testing.T) {
		t.Skip()
		putModel := []testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
			{Name: "Jack"},
		}

		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		getModel := []testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")

		if err = gm.GetMultiConsistency(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")

		getModel = []testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, putModel[0].Name, getModel[0].Name, "gostore GetMulti")
	})
}
