package gonm

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/datastore"
	"github.com/stretchr/testify/assert"
)

func TestGonm_RunInTransaction(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	gm, err := FromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer gm.Close()

	t.Run("simple Transaction", func(t *testing.T) {
		putModel := []testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
		}
		deleteModel := []testModel{
			{ID: 3, Name: "Michael"},
			{ID: 4, Name: "Tom"},
		}
		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		if _, err = gm.PutMulti(deleteModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		getModel := []*testModel{{ID: 1}, {ID: 2}}
		_, err = gm.RunInTransaction(func(gm *Gonm) error {

			if err = gm.GetMulti(getModel); err != nil {
				return err
			}
			getModel[0].Name = "Hanako"
			getModel = append(getModel, &testModel{Name: "Jack"})
			if _, err = gm.PutMulti(getModel); err != nil {
				return err
			}

			deleteModel = []testModel{{ID: 3}, {ID: 4}}
			if err = gm.DeleteMulti(deleteModel); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.NotZero(getModel[2].ID, "complete pkey")

		getModel = []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal("Hanako", getModel[0].Name, "gostore GetMulti when commit")

		getModel = []*testModel{{ID: 3}, {ID: 4}}
		err = gm.GetMulti(getModel)
		assert.Error(err, datastore.ErrNoSuchEntity)
	})

	t.Run("rollback Transaction", func(t *testing.T) {
		putModel := []testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
		}
		deleteModel := []*testModel{
			{ID: 3, Name: "Michael"},
			{ID: 4, Name: "Tom"},
		}
		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		if _, err = gm.PutMulti(deleteModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		getModel := []*testModel{{ID: 1}, {ID: 2}}
		_, err = gm.RunInTransaction(func(gm *Gonm) error {
			if err = gm.GetMulti(getModel); err != nil {
				return err
			}
			assert.Equal(putModel[0].Name, getModel[0].Name, "gostore GetMulti in Transaction")

			getModel[0].Name = "Hanako"
			getModel = append(getModel, &testModel{Name: "Jack"})
			if _, err = gm.PutMulti(putModel); err != nil {
				return err
			}

			deleteModelTx := []*testModel{{ID: 3}, {ID: 4}}
			if err = gm.DeleteMulti(deleteModelTx); err != nil {
				return err
			}

			return fmt.Errorf("test callback")
		})
		assert.Errorf(err, "test callback")
		assert.Zero(getModel[2].ID, "incomplete pkey")

		getModel = []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(putModel[0].Name, getModel[0].Name, "gostore GetMulti when callback")

		getModel = []*testModel{{ID: 3}, {ID: 4}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(deleteModel, getModel, "gostore GetDelete when callback")
	})
}

func TestGonm_NewTransaction(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	gm, err := FromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer gm.Close()

	t.Run("simple Transaction", func(t *testing.T) {
		putModel := []testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
		}
		deleteModel := []testModel{
			{ID: 3, Name: "Michael"},
			{ID: 4, Name: "Tom"},
		}
		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		if _, err = gm.PutMulti(deleteModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		getModel := []*testModel{{ID: 1}, {ID: 2}}
		{
			gmtx, err := gm.NewTransaction()
			if err != nil {
				t.Fatal(gm.printStackErrs(err))
			}
			if err = gmtx.GetMulti(getModel); err != nil {
				t.Fatal(gmtx.printStackErrs(err))
			}
			getModel[0].Name = "Hanako"
			getModel = append(getModel, &testModel{Name: "Jack"})
			if _, err = gmtx.PutMulti(getModel); err != nil {
				t.Fatal(gmtx.printStackErrs(err))
			}

			deleteModel = []testModel{{ID: 3}, {ID: 4}}
			if err = gmtx.DeleteMulti(deleteModel); err != nil {
				t.Fatal(gmtx.printStackErrs(err))
			}
			if _, err := gmtx.Commit(); err != nil {
				t.Fatal(gmtx.printStackErrs(err))
			}
		}

		getModel = []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal("Hanako", getModel[0].Name, "gostore GetMulti when commit")

		getModel = []*testModel{{ID: 3}, {ID: 4}}
		err = gm.GetMulti(getModel)
		assert.Error(err, datastore.ErrNoSuchEntity)
	})

	t.Run("rollback Transaction", func(t *testing.T) {
		putModel := []testModel{
			{ID: 1, Name: "Michael"},
			{ID: 2, Name: "Tom"},
		}
		deleteModel := []*testModel{
			{ID: 3, Name: "Michael"},
			{ID: 4, Name: "Tom"},
		}
		if _, err = gm.PutMulti(putModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		if _, err = gm.PutMulti(deleteModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}

		getModel := []*testModel{{ID: 1}, {ID: 2}}
		{
			gmtx, err := gm.NewTransaction()
			if err != nil {
				t.Fatal(gm.printStackErrs(err))
			}
			if err = gmtx.GetMulti(getModel); err != nil {
				t.Fatal(gmtx.printStackErrs(err))
			}
			assert.Equal(putModel[0].Name, getModel[0].Name, "gostore GetMulti in Transaction")

			getModel[0].Name = "Hanako"
			getModel = append(getModel, &testModel{Name: "Jack"})
			if _, err = gmtx.PutMulti(putModel); err != nil {
				t.Fatal(gmtx.printStackErrs(err))
			}

			deleteModelTx := []*testModel{{ID: 3}, {ID: 4}}
			if err = gmtx.DeleteMulti(deleteModelTx); err != nil {
				t.Fatal(gmtx.printStackErrs(err))
			}
			if err := gmtx.Rollback(); err != nil {
				t.Fatal(err)
			}
		}

		getModel = []*testModel{{ID: 1}, {ID: 2}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(putModel[0].Name, getModel[0].Name, "gostore GetMulti when callback")

		getModel = []*testModel{{ID: 3}, {ID: 4}}
		if err = gm.GetMulti(getModel); err != nil {
			t.Fatal(gm.printStackErrs(err))
		}
		assert.Equal(deleteModel, getModel, "gostore GetDelete when callback")
	})
}
