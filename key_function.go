/*
 * Copyright (c) 2019 The Gonm Authors
 *
 * File for Analyze structures and manipulate keys.
 */

package gonm

import (
	"fmt"
	"reflect"
	"strings"

	"cloud.google.com/go/datastore"
)

func extractKeys(src interface{}, putRequest bool) (key []*datastore.Key, err error) {
	v := reflect.Indirect(reflect.ValueOf(src))
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("gonm: value must be a slice or pointer-to-slice")
	}
	l := v.Len()

	keys := make([]*datastore.Key, l)
	for i := 0; i < l; i++ {
		vi := v.Index(i)
		key, err := getStructKey(vi.Interface())
		if err != nil {
			return nil, err
		}
		if !putRequest && key.Incomplete() {
			return nil, fmt.Errorf("gonm: cannot find a pkey for struct - %v", vi.Interface())
		}
		keys[i] = key
	}
	return keys, nil
}

// getStructKey returns the pkey of the struct based in its reflected or
// specified kind and id. The second return parameter is true if src has a
// string id.
func getStructKey(src interface{}) (key *datastore.Key, err error) {
	v := reflect.Indirect(reflect.ValueOf(src))
	t := v.Type()
	k := t.Kind()

	if k != reflect.Struct {
		err = fmt.Errorf("gonm: Expected struct, got instead: %v", k)
		return
	}

	var parent *datastore.Key
	var stringID string
	var intID int64
	var kind string
	var hasKeyField bool

	for i := 0; i < v.NumField(); i++ {
		tf := t.Field(i)
		vf := v.Field(i)

		tag := tf.Tag.Get("gonm")
		tagValues := strings.Split(tag, ",")
		var tagValue string
		if len(tagValues) > 0 {
			tagValue = tagValues[0]
		}

		switch {
		case tagValue == "id" || tf.Name == "ID":
			switch vf.Kind() {
			case reflect.Int64:
				if intID != 0 || stringID != "" {
					err = fmt.Errorf("gonm: Only one field may be marked id")
					return
				}
				intID = vf.Int()
			case reflect.String:
				if intID != 0 || stringID != "" {
					err = fmt.Errorf("gonm: Only one field may be marked id")
					return
				}
				stringID = vf.String()
			default:
				err = fmt.Errorf("gonm: ID field must be int64 or string in %v", t.Name())
				return
			}
			hasKeyField = true

		case tagValue == "kind":
			if vf.Kind() == reflect.String {
				if kind != "" {
					err = fmt.Errorf("gonm: Only one field may be marked kind")
					return
				}
				kind = vf.String()
				if kind == "" && len(tagValues) > 1 && tagValues[1] != "" {
					kind = tagValues[1]
				}
			}

		case tagValue == "parent" || tf.Name == "Parent":
			dskeyType := reflect.TypeOf(&datastore.Key{})
			if vf.Type().ConvertibleTo(dskeyType) {
				if parent != nil {
					err = fmt.Errorf("gonm: Only one field may be marked parent")
					return
				}
				parent = vf.Convert(dskeyType).Interface().(*datastore.Key)
			}
		}
	}

	if !hasKeyField {
		return nil, ErrNoIDField
	}

	// if kind has not been manually set, fetch it from src's type
	if kind == "" {
		kind = t.Name()
	}

	switch {
	case intID != 0:
		return datastore.IDKey(kind, intID, parent), nil
	case stringID != "":
		return datastore.NameKey(kind, stringID, parent), nil
	default:
		return datastore.IncompleteKey(kind, parent), nil
	}
}

func setStructKey(src interface{}, key *datastore.Key) error {
	v := reflect.ValueOf(src)
	t := v.Type()
	k := t.Kind()

	if k != reflect.Ptr {
		return fmt.Errorf("gonm: Expected pointer to struct, got instead: %v", k)
	}

	v = reflect.Indirect(v)
	t = v.Type()
	k = t.Kind()

	if k != reflect.Struct {
		return fmt.Errorf(fmt.Sprintf("gonm: Expected struct, got instead: %v", k))
	}

	idSet := false
	kindSet := false
	parentSet := false
	for i := 0; i < v.NumField(); i++ {
		tf := t.Field(i)
		vf := v.Field(i)

		if !vf.CanSet() {
			continue
		}

		tag := tf.Tag.Get("gonm")
		tagValues := strings.Split(tag, ",")
		var tagValue string
		if len(tagValues) > 0 {
			tagValue = tagValues[0]
		}

		switch {
		case tagValue == "id" || tf.Name == "ID":
			if idSet {
				return fmt.Errorf("gonm: Only one field may be marked id")
			}

			if vf.Kind() == reflect.Int64 {
				vf.SetInt(key.ID)
				idSet = true
			}

		case tagValue == "kind":
			if kindSet {
				return fmt.Errorf("gonm: Only one field may be marked kind")
			}
			if vf.Kind() == reflect.String {
				vf.Set(reflect.ValueOf(key.Kind))
				kindSet = true
			}

		case tagValue == "parent" || tf.Name == "Parent":
			if parentSet {
				return fmt.Errorf("gonm: Only one field may be marked parent")
			}
			dskeyType := reflect.TypeOf(&datastore.Key{})
			vfType := vf.Type()
			if vfType.ConvertibleTo(dskeyType) {
				vf.Set(reflect.ValueOf(key.Parent).Convert(vfType))
				parentSet = true
			}
		}
	}

	if !idSet {
		return ErrNoIDField
	}

	return nil
}

// Key generate *datastore.Key from src.
func Key(src interface{}) (*datastore.Key, error) {
	return getStructKey(src)
}

// KindWithTag generate *datastore.Key.Kind from src.
//
// If you do not need to look kind tag, you should use Key.
func KindWithTag(src interface{}) (string, error) {
	key, err := getStructKey(src)
	if err != nil {
		return "", err
	}
	return key.Kind, nil
}

// Kind return struct name of src.
// This method is simple without looking tag of struct
func Kind(src interface{}) string {
	v := reflect.Indirect(reflect.ValueOf(src))
	t := v.Type()
	return t.Name()
}
