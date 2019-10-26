/*
 * Copyright (c) 2019 The Gonm Author
 *
 * File for error
 */

package gonm

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrInTransaction is returned when not available method in transaction
	ErrInTransaction = errors.New("gonm: transaction gonm is not available this method")
	// ErrNoIDField is returned when struct do not have ID field in tag
	ErrNoIdFiled = errors.New("gonm: At least one ID or id tag")
)

func (gm *Gonm) stackError(err error) error {
	if err == nil {
		panic("gonm: err is nil")
	}
	gm.m.Lock()
	defer gm.m.Unlock()

	gm.Errors = append(gm.Errors, errors.WithStack(err))
	return err
}

func printErr(err error) string {
	if err == nil {
		panic("gonm: err is nil")
	}
	return fmt.Sprintf("%+v\n", err)
}

func (gm *Gonm) printStackErrs(err error) string {
	if err == nil {
		panic("gonm: err is nil")
	}
	str := printErr(err)
	for _, e := range gm.Errors {
		if e == err {
			continue
		}
		str += printErr(e)
	}
	return str
}

func (gmtx *GonmTx) printStackErrs(err error) string {
	return gmtx.gonm.printStackErrs(err)
}
