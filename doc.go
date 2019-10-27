// Copyright (c) 2019 The Gonm Author

/*
Package gonm automatically assigns a key from the interface, and autocahing interface local memory.

gonm wrapped Google Cloud Datastore.
I used https://godoc.org/cloud.google.com/go/datastore and https://godoc.org/github.com/mjibson/goon as a reference

Concept

Gonm generate key from ID of structure property, and this key will use for get or put.
All structures are complemented with ID of structure property after these method are used.
Also, gonm stores the results in a local cache by default. Therefore, the same fetch can be performed at high speed.

It is simple to use gonm.
	type User struct {
		 ID   int64 `datastore:"-"`
		 Name string
	}
	gm := gonm.FromContext(ctx, dsClient)
	user := &User{ID: 1}
	err := gm.Get(user)

Properties

A key consists of an optional parent key, and parent key generate Parent of structure property.
ID assumes int64 and string, and Parent assumes *datastore.Key like database api.

example: create child-parent relationship

	type User struct {
		 ID int64 `datastore:"-"`
		 Name string
		 Parent *datastore.Key `datastore:"-"`
	}
	gm := gom.FromContext(ctx, dsClient)

	parent := &User{Name: "Father"}
	key, err := gm.Put(parent)
	if err != nil {
	    // TODO: Handle error.
	}

	child := &User{Name: "Jack", Parent: key}
	_, err := gm.Put(child)

If you want to use other property as key id, you need to put id tag in structure tag.
The same applies to the parent key and key name. For parent Key, you need to put parent tag in structure.
For Key kind, you need to put kind tag in structure.
Gonm returns ErrNoIdFiled when the id cannot be obtained from the received structure.

	type CustomUser struct {
		Id string `datastore:"-" gonm:"id"`
		Kind string `datastore:"-" gonm:"kind"`
		Key *datastore.Key `datastore:"-" gonm:"parent"`
		Name string
	}

Check https://godoc.org/cloud.google.com/go/datastore#hdr-Properties to lean more about datastore properties.


The PropertyLoadSaver Interface

Of course you can use PropertyLoadSaver Interface. However, be careful because gonm wraps the datastore.

 get flow:  Get of gonm -> key generate -> call get api -> Load of PropertyLoadSaver -> complemented with ID
 put flow:  Put of gonm -> key generate -> Save of PropertyLoadSaver -> call put api -> complemented with ID

Check https://godoc.org/cloud.google.com/go/datastore#hdr-The_PropertyLoadSaver_Interface to lean more about PropertyLoadSaver Interface.


Queries

Queries of gonm is very similar datastore queries. Gonm use datastore.Query.
Gonm support Run and GetAll, but I reconmmend using GetKeysOnly.

	q := datastore.NewQuery("User").Limit(2)

	keys, cursor, err := gm.GetKeysOnly(q)
	dst := make([]User, len(keys))
	if err := gm.GetMultiByKeys(keys, dst); err != nil {
	   // TODO: Handle error.
	}


Transactions

Gonm.RunInTransaction runs a function in a transaction.

	Users := []*User{{ID: 1}, {ID: 2}}
	_, err = gm.RunInTransaction(func(gm *Gonm) error {

		if err = gm.GetMulti(Users); err != nil {
			return err
		}
		Users[0].Name = "Hanako"
		if _, err = gm.PutMulti(Users); err != nil {
			return err
		}
		return nil
	})


Google Cloud Datastore Emulator

To install and set up the emulator and its environment variables,
see the documentation at https://cloud.google.com/datastore/docs/tools/datastore-emulator.


*/
package gonm
