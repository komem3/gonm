package gonm

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/datastore"
	"github.com/stretchr/testify/assert"
)

func TestGonm_Mutate(t *testing.T) {
	ctx := context.Background()

	gm, err := FromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer gm.Close()

	t.Run("normal mutate", func(t *testing.T) {
		putModel := []*testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
		}
		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		insertMode := &testModel{Name: "Miho"}
		_, err = gm.Mutate(
			NewInsert(insertMode),
			NewDelete(&testModel{ID: 1}),
			NewUpdate(&testModel{ID: 2, Name: "Taro"}),
			NewUpsert(&testModel{ID: 4, Name: "George"}),
		)
		if err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		assert.NotZero(t, insertMode.ID, "insert model ID is not zero")

		getModel := []*testModel{{ID: 2}, {ID: 4}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, "Taro", getModel[0].Name)
		assert.Equal(t, "George", getModel[1].Name)

		single := &testModel{ID: 1}
		err = gm.Get(single)
		gm.Errors = nil
		assert.Error(t, err, datastore.ErrNoSuchEntity)
	})

	t.Run("transaction mutate success", func(t *testing.T) {
		putModel := []*testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
		}
		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		insertMode := &testModel{Name: "Miho"}
		_, err := gm.RunInTransaction(func(gm *Gonm) error {
			_, err = gm.Mutate(
				NewInsert(insertMode),
				NewDelete(&testModel{ID: 1}),
				NewUpdate(&testModel{ID: 2, Name: "Taro"}),
				NewUpsert(&testModel{ID: 4, Name: "George"}),
			)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.NotZero(t, insertMode.ID, "insert model ID is not zero")

		getModel := []*testModel{{ID: 2}, {ID: 4}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, "Taro", getModel[0].Name)
		assert.Equal(t, "George", getModel[1].Name)

		single := &testModel{ID: 1}
		err = gm.Get(single)
		assert.Error(t, err, datastore.ErrNoSuchEntity)
	})

	t.Run("transaction mutate callback", func(t *testing.T) {
		putModel := []*testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
		}
		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		if err = gm.Delete(&testModel{ID: 4}); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		insertMode := &testModel{Name: "Miho"}
		_, err := gm.RunInTransaction(func(gm *Gonm) error {
			_, err = gm.Mutate(
				NewInsert(insertMode),
				NewDelete(&testModel{ID: 1}),
				NewUpdate(&testModel{ID: 2, Name: "Taro"}),
				NewUpsert(&testModel{ID: 4, Name: "George"}),
			)
			if err != nil {
				return err
			}
			return fmt.Errorf("mutate error")
		})
		assert.Error(t, err, fmt.Errorf("mutate error"))
		assert.Zero(t, insertMode.ID, "insert model ID is zero")

		getModel := []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(t, "Michael", getModel[0].Name)
		assert.Equal(t, "Tom", getModel[1].Name)

		single := &testModel{ID: 4}
		err = gm.Get(single)
		assert.Error(t, err, datastore.ErrNoSuchEntity)
	})
}
