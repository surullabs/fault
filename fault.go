// Copyright 2014, Surul Software Labs GmbH
// All rights reserved.

/*
Package fault provides utilities to help with package internal error handling.

It supports a simple idiom for reducing the amount of package internal error
handling. It allows you to panic with a Fault instance in package internal code.
This is then recovered using a defer for a all exported methods, which then return
an error extracted from the fault. As an example, if you were to be reading data
from a file and then writing it you could use

	import (
		"github.com/surullabs/fault"
	)

	var check = fault.NewChecker()

	func ExportedMethod() (err error) {
		// Set up the recovery. err will be automatically populated and all
		// non-fault panics will be propogated.
		defer check.Recover(&err)

		// If there is an error in ReadFile the method will automatically return
		// the error.
		data := check.Return(ioutil.ReadFile("filename")).([]byte)
		// If yourFn returns false the function will return an error
		// formatted as "condition is not true: yourData"
		check.Truef(yourFn(data), "condition is not true: %s", string(data))
	}

It also provides access to an ErrorChain class which can be used to chain errors together.
Errors can be transparently checked for existence in a chain by calling the Contains method.

Please look at the tests for more sample usage.
*/
package fault

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// ErrorChain is a list of errors and can be used to chain errors together.
type ErrorChain struct {
	chain []error
}

// AsError returns an error if any are present in the chain or nil if not
func (c *ErrorChain) AsError() error {
	if c == nil || len(c.chain) == 0 {
		return nil
	}
	return c
}

// String returns the same value as Error()
func (c *ErrorChain) String() string { return c.Error() }

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
func (c *ErrorChain) Append(err error) *ErrorChain {
	if err == nil {
		return c
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
	return c
}

func NewErrorChain() *ErrorChain { return &ErrorChain{} }

// Chain will chain a list of errors passed in. The errors can
// be of type *ErrorChain in which case their chains will be appended.
func Chain(errs ...error) error {
	chain := &ErrorChain{}
	for _, err := range errs {
		chain.Append(err)
	}
	return chain.AsError()
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

// Fault is an interface for representing a package internal fault.
type Fault interface {
	// Cause returns the error tied to this fault
	Cause() error
	// Error returns the result of a call to Cause().Error()
	Error() string
}

// Faulter is an interface used to generate faults from errors
type Faulter interface {
	New(err error) Fault
}

// FaultCheck is an interface providing functionality to check for faults and recover from them.
type FaultCheck interface {
	// Recover is used to recover from faults that were expressed through panics.
	// It will call recover() internally and the error variable pointed to by the argument will be populated with the fault information.
	Recover(*error)
	// RecoverPanic works exactly like recover with the exception that the second argument
	// must be the result of a call to recover()
	RecoverPanic(*error, interface{})
	// True will panic with a fault if the condition provided is false
	// The fault error string will be the second argument
	True(bool, string)
	// Truef behaves like Check with the error string as the result of a call to fmt.Errorf(format, args...)
	Truef(bool, string, ...interface{})
	// Return will panic if the error provided is not nil. It will return the first argument if not
	Return(interface{}, error) interface{}
	// Error is equivalent to a call to Return(nil, err)
	Error(error)
	// Output functions exactly as Return, with the only difference being that the output is included in the error message.
	// If the output is a byte array it is converted to a string.
	// This can be useful when debugging use of os/exec package for instance.
	Output(interface{}, error) interface{}
	// Fail will return a fault containing the provided error.
	// Usage: panic(check.Failure(err))
	Failure(error) Fault
}

// Checker provides a default implementation of FaultCheck
type Checker struct {
	faulter Faulter
}

// NewChecker returns a new checker that includes stack traces with errors.
//
// 	var check fault.FaultCheck = fault.NewChecker()
//
// If you don't want the overhead of accumulating stack traces then use
//
//	var check fault.FaultCheck = fault.NewChecker().SetFaulter(fault.Simple)
func NewChecker() *Checker { return &Checker{faulter: &DebugFaulter{}} }

func (c *Checker) SetFaulter(f Faulter) *Checker {
	c.faulter = f
	return c
}

// RecoverPanic implements FaultCheck.RecoverPanic
func (c *Checker) RecoverPanic(errPtr *error, panicked interface{}) {
	if panicked == nil {
		return
	} else if fault, faulty := panicked.(Fault); faulty {
		*errPtr = Chain(fault.Cause(), *errPtr)
		return
	} else {
		panic(panicked)
	}
}

// Recover implements FaultCheck.Recover
func (c *Checker) Recover(errPtr *error) {
	c.RecoverPanic(errPtr, recover())
}

var Simple Faulter = errorFaulter{}

// errorFaulter generates faults which do not contain a complete stack trace.
type errorFaulter struct{}

func (errorFaulter) New(err error) Fault { return &errorFault{err: err} }

type errorFault struct {
	err error
}

func (e *errorFault) Error() string { return e.err.Error() }
func (e *errorFault) Cause() error  { return e.err }

func (e *errorFault) String() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()

}

func (c *Checker) True(condition bool, errStr string) {
	if !condition {
		panic(c.faulter.New(errors.New(errStr)))
	}
}

// True implements FaultCheck.True
func (c *Checker) Truef(condition bool, format string, args ...interface{}) {
	if !condition {
		panic(c.faulter.New(fmt.Errorf(format, args...)))
	}
}

// Return implements FaultCheck.Return
func (c *Checker) Return(i interface{}, err error) interface{} {
	if err != nil {
		panic(c.faulter.New(err))
	}
	return i
}

// Error implements FaultCheck.Error
func (c *Checker) Error(err error) {
	if err != nil {
		panic(c.faulter.New(err))
	}
}

// Output implements FaultCheck.Output
func (c *Checker) Output(i interface{}, err error) interface{} {
	if err != nil {
		var out string
		if bytes, isByteArray := i.([]byte); isByteArray {
			out = string(bytes)
		} else {
			out = fmt.Sprintf("%v", i)
		}
		panic(c.faulter.New(&ErrorChain{chain: []error{err, fmt.Errorf("output: %s", out)}}))
	}
	return i
}

func (c *Checker) Failure(err error) Fault {
	return c.faulter.New(err)
}

// Call provides information about a function call.
type Call struct {
	File string // File provides the file of the caller
	Line int    // Line provides the line number
	Name string // Name is the name of the calling function
}

func (c *Call) String() string { return fmt.Sprintf("%s:%d", filepath.Base(c.File), c.Line) }

func (c *Call) Equal(c2 *Call) bool {
	if c == nil {
		return c2 == nil
	} else if c2 == nil {
		return false
	}
	return c.File == c2.File && c.Line == c2.Line && c.Name == c2.Name
}

type debugFault struct {
	err   error
	trace []Call
}

func GetTrace(err error) (trace []Call) {
	if chain, isChain := err.(*ErrorChain); isChain && len(chain.Errors()) == 1 {
		err = chain.Errors()[0]
	}
	if fault, isFault := err.(*debugFault); isFault {
		return fault.trace
	}
	return nil
}

func StartSite(trace []Call) (call *Call) {
	if len(trace) == 0 {
		call = &Call{"?", -1, "?"}
	} else {
		call = &trace[0]
	}
	return
}

func (d *debugFault) Error() string {
	return fmt.Sprintf("%v: %s", StartSite(d.trace), d.err.Error())
}

func (d *debugFault) Cause() error { return d }

type DebugFaulter struct {
	Prefix string
}

func (d DebugFaulter) prefix() string {
	if d.Prefix != "" {
		return d.Prefix
	}
	return checkerPrefix
}

var checkerPrefix = TypePrefix(&Checker{})

func TypePrefix(i interface{}) string {
	val := reflect.ValueOf(i)
	if val.Kind() == reflect.Ptr {
		typeVal := val.Elem().Type()
		return fmt.Sprintf("%s.(*%s)", typeVal.PkgPath(), typeVal.Name())
	} else {
		return fmt.Sprintf("%s.(%s)", val.Type().PkgPath(), val.Type().Name())
	}
}

// ReadStack reads returns the stack after ignoring all calls up to the
// function which has the first parameter as a prefix . An empty string returns
// the entire stack.
func ReadStack(prefix string) (trace []Call) {
	trace = make([]Call, 0)
	var (
		pc uintptr
		fn *runtime.Func
	)
	appendTo := false
	for skip, ok := 0, true; ok; skip++ {
		call := Call{Name: "?"}
		if pc, call.File, call.Line, ok = runtime.Caller(skip); !ok {
			break
		}
		if fn = runtime.FuncForPC(pc); fn != nil {
			call.Name = fn.Name()
		}

		if appendTo {
			trace = append(trace, call)
		}
		if strings.HasPrefix(call.Name, prefix) {
			appendTo = true
		}
	}
	return
}

func (d DebugFaulter) New(err error) Fault {
	return &debugFault{err: err, trace: ReadStack(d.prefix())}
}

// Traced returns an error with the entire stack trace
func Traced(err error) error {
	if chain, isChain := err.(*ErrorChain); isChain && len(chain.Errors()) == 1 {
		err = chain.Errors()[0]
	}
	if _, ok := err.(*debugFault); ok {
		return err
	}
	trace := ReadStack("")
	return &debugFault{err: err, trace: trace[1:]}
}

func VerboseTrace(err error) string {
	trace := GetTrace(err)
	if trace == nil {
		return err.Error()
	}

	parts := make([]string, len(trace))
	for i := range trace {
		parts[i] = trace[i].String()
	}
	parts[0] = err.Error()
	return strings.Join(parts, "\n")
}
