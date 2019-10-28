package gonm_test

import (
	"context"
	"fmt"
	"github.com/komem3/gonm"

	"cloud.google.com/go/datastore"
)

type User struct {
	ID   int64
	Name string
}

var dsClient *datastore.Client

func ExampleKind() {
	type User struct {
		ID   int64
		Name string
	}
	fmt.Println(gonm.Kind(User{}))
	// Output: User
}

func ExampleKindWithTag() {
	type User struct {
		ID   int64
		Kind string `datastore:"-" gonm:"kind,user"`
	}

	kind, err := gonm.KindWithTag(User{})
	if err != nil {
		// TODO: Handle error.
	}
	fmt.Println(kind)
	// Output: user
}

func ExampleGonm_AllocateID() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	user := &User{Name: "Hanako"}
	key, err := gm.AllocateID(user)
	if err != nil {
		// TODO: Handle error.
	}

	// TODO: Use key.
	_ = key
}

func ExampleGonm_AllocateIDs() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	users := []*User{
		{Name: "Michel"},
		{Name: "Hanako"},
	}
	keys, err := gm.AllocateIDs(users)
	if err != nil {
		// TODO: Handle error.
	}

	// TODO: Use keys.
	_ = keys
}

func ExampleGonm_Delete() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	if err := gm.Delete(&User{ID: 1}); err != nil {
		// TODO: Handle error.
	}
}

func ExampleGonm_DeleteMulti() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)
	users := []*User{{ID: 1}, {ID: 2}}
	if err := gm.DeleteMulti(users); err != nil {
		// TODO: Handle error.
	}
}

func ExampleGonm_Get() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	user := &User{ID: 2}
	if err := gm.Get(user); err != nil {
		// TODO: Handle error.
	}
}

func ExampleGonm_GetAll() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	var users []*User
	keys, err := gm.GetAll(datastore.NewQuery("User"), &users)

	if err != nil {
		// TODO: Handle error.
	}
	for i, key := range keys {
		fmt.Println(key)
		fmt.Println(users[i])
	}
}

func ExampleGonm_GetKeysOnly() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	q := datastore.NewQuery("User").Limit(2)

	keys, cursor, err := gm.GetKeysOnly(q)
	if err != nil {
		// TODO: Handle error.
	}

	dst := make([]User, len(keys))
	if err := gm.GetMultiByKeys(keys, dst); err != nil {
		// TODO: Handle error.
	}

	_ = cursor
}

func ExampleGonm_GetMulti() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	users := []*User{{ID: 1}, {ID: 2}}
	if err := gm.GetMulti(users); err != nil {
		// TODO: Handle error.
	}
}

func ExampleGonm_Mutate() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	_, err := gm.Mutate(
		gonm.NewInsert(&User{ID: 1, Name: "Jack"}),
		gonm.NewUpsert(&User{ID: 2, Name: "Michael"}),
		gonm.NewUpdate(&User{ID: 3, Name: "Tom"}),
		gonm.NewDelete(&User{ID: 4}),
	)
	if err != nil {
		// TODO: Handle error.
	}
}

func ExampleGonm_NewTransaction() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	tx, err := gm.NewTransaction() // returns GonmTx instead of Gonm
	if err != nil {
		// TODO: Handle error.
	}

	users := []*User{{ID: 1}, {ID: 2}}
	if err := tx.GetMulti(users); err != nil {
		// TODO: Handle error.
	}

	users[0].Name = "Change"
	if _, err = tx.PutMulti(users); err != nil {
		// TODO: Handle error.
	}

	if _, err := tx.Commit(); err != nil {
		// TODO: Handle error.
	}
}

func ExampleGonm_Put() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	if _, err := gm.Put(&User{Name: "Tom"}); err != nil {
		// TODO: Handle error.
	}
}

func ExampleGonm_PutMulti() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	users := []*User{
		{Name: "Tom"},
		{Name: "Jack"},
	}
	if _, err := gm.PutMulti(users); err != nil {
		// TODO: Handle error.
	}
}

func ExampleGonm_Run() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	q := datastore.NewQuery("User").Limit(2)
	it, err := gm.Run(q)
	if err != nil {
		// TODO: Handle error.
	}
	_ = it
}

func ExampleGonm_RunInTransaction() {
	ctx := context.Background()
	gm := gonm.FromContext(ctx, dsClient)

	users := []*User{{ID: 1}, {ID: 2}}
	_, err := gm.RunInTransaction(func(gmtx *gonm.Gonm) error {
		if err := gmtx.GetMulti(users); err != nil {
			return err
		}

		users[0].Name = "Change"
		if _, err := gmtx.PutMulti(users); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		// TODO: Handle error.
	}
}
