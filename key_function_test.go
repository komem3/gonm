package gonm

import (
	"testing"

	"cloud.google.com/go/datastore"
	"github.com/stretchr/testify/assert"
)

type testModel2 struct {
	IDOther int64  `datastore:"-" gonm:"id"`
	Kind    string `datastore:"-" gonm:"kind,test"`
	Name    string
	Parent  *datastore.Key `datastore:"-"`
}

type testModel3 struct{}

func TestExtractKey(t *testing.T) {
	assert := assert.New(t)

	test := testModel{}
	_, err := extractKeys(test, false)
	assert.Error(err, "gonm: value must be a slice or pointer-to-slice")

	testList := []testModel{{}, {}}
	key, err := extractKeys(testList, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(2, len(key))
	assert.Equal("/testModel,0", key[0].String())

	_, err = extractKeys(testList, false)
	assert.Error(err, "gonm: empty id on put")
}

func TestGetStructKey(t *testing.T) {
	assert := assert.New(t)
	test := testModel{}

	key, err := getStructKey(test)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal("/testModel,0", key.String(), "pkey name is struct name")
	assert.Equal(true, key.Incomplete(), "pkey is incomplete")

	test = testModel{ID: 2}
	key, err = getStructKey(test)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal("/testModel,2", key.String(), "pkey name is struct name")
	assert.Equal(false, key.Incomplete(), "pkey is not incomplete")

	test2 := &testModel2{Kind: "testModeller", Parent: datastore.IDKey("kind", 1, nil)}
	key, err = getStructKey(test2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal("/kind,1/testModeller,0", key.String(), "pkey name is Kind name")

	test2 = &testModel2{}
	key, err = getStructKey(test2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal("/test,0", key.String(), "pkey name is default Kind name")

	test3 := testModel3{}
	_, err = getStructKey(test3)
	assert.Error(err, "gonm: At least one ID or id tag in testModel3")
}

func TestSetStructKey(t *testing.T) {
	parentKey := datastore.IDKey("test", 2, nil)
	key := datastore.IDKey("test", 1, parentKey)
	test := &testModel2{}

	if err := setStructKey(test, key); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "test", test.Kind, "kind property set kind of pkey")
	assert.Equal(t, int64(1), test.IDOther, "id property set id of pkey")
	assert.Equal(t, parentKey, test.Parent, "parent property set parent pkey of pkey")
}

func TestKind(t *testing.T) {
	test := &testModel2{}
	assert.Equal(t, "testModel2", Kind(test), "struct name is kind name")
	kind, err := KindWithTag(test)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "test", kind, "struct tag is kind name")

	test2 := testModel2{}
	assert.Equal(t, "testModel2", Kind(test2), "struct name is kind name")
	kind, err = KindWithTag(test2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "test", kind, "struct tag is kind name")
}
