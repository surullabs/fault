// Copyright 2014, Surul Software Labs GmbH
// All rights reserved.

/*
Package fault provides utilities to help with package internal error handling.

It supports a simple idiom for reducing the amount of package internal error
handling. It allows you to panic with a Fault instance in package internal code.
This is then recovered using a defer for a all exported methods, which then return
an error extracted from the fault. As an example, if you were to be reading data
from a file and then writing it you could use

	func ExportedMethod() (err error) {
		// Set up the recovery. err will be automatically populated and all
		// non-fault panics will be propogated.
		defer func() { fault.Recover(&err, recover()) } ()

		// If there is an error in ReadFile the method will automatically return
		// the error.
		data := fault.CheckReturn(ioutil.ReadFile("filename")).([]byte)
		// If yourFn returns false the function will return an error
		// formatted as "condition is not true: yourData"
		fault.Check(yourFn(data), "condition is not true: %s", string(data))
	}

It also provides access to an ErrorChain class which can be used to chain errors together.
Errors can be transparently checked for existence in a chain by calling the Contains method.

Please look at the tests for more sample usage.
*/
package fault

import (
	"fmt"
	"strings"
)

// Fault is an interface for representing a package internal fault.
type Fault interface {
	Fault() error // Error tied to this fault
}

// ErrorChain is a list of errors and can be used to chain errors together.
type ErrorChain struct {
	chain []error
}

// Errors returns all errors in the chain
func (c *ErrorChain) Errors() []error { return c.chain }

// Error will return a string representation of all errors.
func (c *ErrorChain) Error() string {
	errors := make([]string, len(c.chain))
	for i, err := range c.chain {
		errors[i] = err.Error()
	}
	return strings.Join(errors, "; ")
}

// Chain appends the error provided to the current chain. If the
// err is a chain then all errors in the chain are appended.
func (c *ErrorChain) Chain(err error) {
	if err == nil {
		return
	}

	if c.chain == nil {
		c.chain = make([]error, 0)
	}

	switch e := err.(type) {
	case *ErrorChain:
		if e.chain != nil {
			c.chain = append(c.chain, e.chain...)
		}
	default:
		c.chain = append(c.chain, e)
	}
}

// Chain will chain a list of errors passed in. The errors can
// be of type *ErrorChain in which case their chains will be appended.
func Chain(errs ...error) *ErrorChain {
	if len(errs) == 0 {
		return nil
	}

	chain := &ErrorChain{}
	if errs[0] != nil {
		switch e := errs[0].(type) {
		case *ErrorChain:
			chain.chain = e.chain
		default:
			chain.chain = []error{e}
		}
	}

	for _, err := range errs[1:] {
		chain.Chain(err)
	}
	return chain
}

// Contains will return true in the following cases:
//
// 	* chain.Error() == target.Error()
// 	* chain is an ErrorChain and one of the errors is target
// 	* Contains(target, chain) returns true
func Contains(chain, target error) bool {
	if chain == nil || target == nil {
		return false
	}
	if chain.Error() == target.Error() {
		return true
	}
	if chainErr, isChain := chain.(*ErrorChain); isChain {
		for _, err := range chainErr.chain {
			if err.Error() == target.Error() {
				return true
			}
		}
	}
	if chainErr, isChain := target.(*ErrorChain); isChain {
		for _, err := range chainErr.chain {
			if err.Error() == chain.Error() {
				return true
			}
		}
	}
	return false
}

// Recover is used to recover from faults that were expressed through
// panics.
//
// The expected usage is as follows
//
//	func MyExportedFunc() (err error) {
//		defer func() { fault.Recover(&err, recover()) }()
//		// On error raise a fault. err will be automatically
//		// populated
//		panic(myFault)
//	}
//
// The first argument must hold a pointer to an error which will
// be set to an error generated from second argument if that is a Fault.
// If the second argument is nil nothing will be done. If it is non-nil
// and does not implement Fault a panic will be re-raised.
//
func Recover(errPtr *error, panicked interface{}) {
	if panicked == nil {
		return
	} else if fault, faulty := panicked.(Fault); faulty {
		*errPtr = Chain(*errPtr, fault.Fault())
		return
	}
	panic(panicked)
}

type errorFault struct {
	err error
}

func (e errorFault) Fault() error { return e.err }

// Check will panic with a fault if the condition provided is false
// The fault error will be the result of a call to fmt.Errorf(format, args...)
func Check(condition bool, format string, args ...interface{}) {
	if !condition {
		panic(errorFault{err: fmt.Errorf(format, args...)})
	}
}

// CheckReturn will panic if the error provided is not nil. It will return
// the first argument if not
func CheckReturn(i interface{}, err error) interface{} {
	if err != nil {
		panic(errorFault{err: err})
	}
	return i
}

// CheckError is equivalent to a call to CheckReturn(nil, err)
func CheckError(err error) { CheckReturn(nil, err) }

// CheckOutput functions exactly as CheckReturn, with the only
// difference being that the output is included in the error message.
// This can be useful when debugging use of os/exec package for instance.
func CheckOutput(i interface{}, err error) interface{} {
	if err != nil {
		panic(errorFault{err: &ErrorChain{chain: []error{err, fmt.Errorf("output: %v", i)}}})
	}
	return i
}
