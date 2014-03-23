// Copyright 2014, Surul Software Labs GmbH
// All rights reserved.

package fault

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// for testing
var _ = fmt.Sprintf

var check FaultCheck = NewChecker()

func TestErrorChain(t *testing.T) {
	for _, test := range []struct {
		name string
		test func() error
		err  string
	}{
		{
			"Test one error",
			func() error { return &ErrorChain{chain: []error{errors.New("error1")}} },
			"error1",
		},
		{
			"Test two errors",
			func() error { return &ErrorChain{chain: []error{errors.New("error1"), errors.New("error2")}} },
			"error1; error2",
		},
		{
			"Test three errors",
			func() error {
				return &ErrorChain{chain: []error{errors.New("error1"), errors.New("error2"), errors.New("error3")}}
			},
			"error1; error2; error3",
		},
		{
			"Test error chain nil",
			func() error { return &ErrorChain{} },
			"",
		},
		{
			"Test chain nil",
			func() error {
				chain := &ErrorChain{chain: nil}
				chain.Chain(errors.New("error1"))
				return chain
			},
			"error1",
		},
		{
			"Test chain nil call",
			func() error {
				chain := &ErrorChain{chain: []error{errors.New("error1")}}
				chain.Chain(nil)
				return chain
			},
			"error1",
		},
		{
			"Test chain one error",
			func() error {
				chain := &ErrorChain{chain: []error{errors.New("error1")}}
				chain.Chain(errors.New("error2"))
				return chain
			},
			"error1; error2",
		},
		{
			"Test chain multi nil",
			func() error {
				chain := &ErrorChain{chain: []error{errors.New("error1")}}
				chain.Chain(&ErrorChain{})
				return chain
			},
			"error1",
		},
		{
			"Test chain multi",
			func() error {
				chain := &ErrorChain{chain: []error{errors.New("error1")}}
				chain.Chain(&ErrorChain{chain: []error{errors.New("error2"), errors.New("error3")}})
				return chain
			},
			"error1; error2; error3",
		},
		{
			"Test chainer empty",
			func() error { return Chain() },
			"",
		},
		{
			"Test chainer one",
			func() error { return Chain(errors.New("error1")) },
			"error1",
		},
		{
			"Test chainer two",
			func() error { return Chain(errors.New("error1"), errors.New("error2")) },
			"error1; error2",
		},
		{
			"Test chainer error chain",
			func() error { return Chain(errors.New("error1"), Chain(errors.New("error2"))) },
			"error1; error2",
		},
		{
			"Test chainer error chain first",
			func() error { return Chain(Chain(errors.New("error1")), errors.New("error2")) },
			"error1; error2",
		},
		{
			"Test chainer error chain multi",
			func() error { return Chain(Chain(errors.New("error1")), Chain(errors.New("error2"))) },
			"error1; error2",
		},
	} {
		t.Log(test.name)
		err := test.test()
		if err == nil && test.err != "" {
			t.Error("Expected error", test.err, "not found")
		} else if err != nil && err.Error() != test.err {
			t.Error("Expected", test.err, "found", err.Error())
		}
	}

	// Test error listing
	chain := Chain(errors.New("error1"), errors.New("error2")).(*ErrorChain)
	if len(chain.Errors()) != 2 {
		t.Error("Invalid chain found", chain.Errors())
	}

	// Test empty errors
	if Chain(nil, nil, nil) != nil {
		t.Error("Failed nil only chain")
	}
}

func runRecover(fn func()) (err error) {
	defer check.Recover(&err)
	fn()
	return
}

func TestRecover(t *testing.T) {
	for _, test := range []struct {
		name string
		test func()
		err  string
	}{
		{
			"Check false",
			func() { check.True(false, "error1") },
			"error1",
		},
		{
			"Check true",
			func() { check.True(true, "error1") },
			"",
		},
		{
			"Checkf false",
			func() { check.Truef(false, "error1 %s", "error") },
			"error1 error",
		},
		{
			"Checkf true",
			func() { check.Truef(true, "error1 %s", "error") },
			"",
		},
		{
			"Check return",
			func() { check.Return("str", errors.New("error1")) },
			"error1",
		},
		{
			"Check return success",
			func() {
				if "str" != check.Return("str", nil).(string) {
					t.Error("Check return failed")
				}
			},
			"",
		},
		{
			"Check output",
			func() { check.Output("str", errors.New("error1")) },
			"error1; output: str",
		},
		{
			"Check output bytes",
			func() { check.Output([]byte("str bytes"), errors.New("error1")) },
			"error1; output: str bytes",
		},
		{
			"Check output success",
			func() {
				if "str" != check.Output("str", nil).(string) {
					t.Error("Check output failed")
				}
			},
			"",
		},
		{
			"Check error",
			func() { check.Error(errors.New("error1")) },
			"error1",
		},
		{
			"Check error no error",
			func() { check.Error(nil) },
			"",
		},
	} {
		t.Log(test.name)
		err := runRecover(test.test)
		if err == nil && test.err != "" {
			t.Error("Expected error", test.err, "not found")
		} else if err != nil && err.Error() != test.err {
			t.Error("Expected", test.err, "found", err.Error())
		}
	}
}

func TestRecoverPanic(t *testing.T) {
	defer func() {
		e := recover()
		if e == nil || e.(string) != "different panic" {
			t.Error("Not recovered")
		}
	}()

	runRecover(func() {
		panic("different panic")
	})
}

func TestContains(t *testing.T) {
	error1 := errors.New("error1")
	error2 := errors.New("error2")

	for _, test := range []struct {
		name   string
		err1   error
		err2   error
		result bool
	}{
		{"nil", nil, nil, false},
		{"nil one", nil, error1, false},
		{"nil other", error1, nil, false},
		{"equal", error1, error1, true},
		{"equal", error1, errors.New("error1"), true},
		{"unequal", error1, errors.New("error2"), false},
		{"unequal", error1, Chain(error2), false},
		{"unequal", Chain(error1), error2, false},
		{"equalchain", error1, Chain(error2, error1), true},
		{"equalchain", Chain(error2, error1), error1, true},
	} {
		t.Log(test.name)
		if Contains(test.err1, test.err2) != test.result {
			t.Error("Failed")
		}
	}
}

func TestString(t *testing.T) {
	err := Chain(errors.New("err1")).(*ErrorChain)
	if err.Error() != err.String() || err.Error() != "err1" {
		t.Error("Error string does not match")
	}

	fault := &errorFault{err: nil}
	if fault.String() != "" {
		t.Error("Fault string not empty")
	}
	fault.err = err
	if fault.String() != "err1" {
		t.Error("Fault string mismatch")
	}
}

func getIntForDebug(failErr error) (retVal int, err error) {
	return 5, failErr
}

func DebugFaultFunc(failErr error) (err error) {
	debug := NewChecker()
	debug.SetFaulter(DebugFaulter{})

	defer debug.Recover(&err)
	debug.True(3 == debug.Return(getIntForDebug(failErr)).(int), "Number mismatch")
	return
}

func TestDebugging(t *testing.T) {
	ptr := reflect.ValueOf(DebugFaultFunc).Pointer()
	fn := runtime.FuncForPC(ptr)
	name := fn.Name()
	_, line := fn.FileLine(ptr)
	prefix := fmt.Sprintf("fault_test.go:%d:%s", line+5, name) // The fail is 5 lines into the function
	for _, test := range []struct {
		fail     error
		expected string
	}{
		{nil, prefix + ": Number mismatch"},
		{errors.New("return error"), prefix + ": return error"},
	} {
		err := DebugFaultFunc(test.fail)
		trace := GetTrace(err)
		if len(trace) == 0 {
			t.Error("No debug trace found when testing err", test.fail)
			continue
		}

		if trace[0].Name != name {
			t.Error("Unexpected trace beginning", trace[0].Name, "expected", name)
		}
		if test.expected != err.Error() {
			t.Error("Expected", test.expected, "found", err.Error())
		}
	}

	// Now test a missing trace
	if GetTrace(errors.New("err")) != nil {
		t.Error("Found trace when non expected")
	}

	errStr := (&debugFault{err: errors.New("err")}).Error()
	if errStr != "?:-1:?: err" {
		t.Error("Found invalid error string")
	}
}

func TestTypePrefix(t *testing.T) {
	if !strings.HasSuffix(TypePrefix(&Checker{}), "(*Checker)") {
		t.Error("Invalid suffix for pointer")
	}
	if !strings.HasSuffix(TypePrefix(Checker{}), "(Checker)") {
		t.Error("Invalid suffix for non pointer")
	}
}

func TestCallEquals(t *testing.T) {
	for _, test := range []struct {
		c1     *Call
		c2     *Call
		result bool
	}{
		{nil, nil, true},
		{nil, &Call{}, false},
		{&Call{}, &Call{}, true},
		{&Call{"f", 1, "n"}, nil, false},
		{&Call{"f", 1, "n"}, &Call{"f", 2, "n"}, false},
		{&Call{"f", 1, "n"}, &Call{"f", 1, "m"}, false},
		{&Call{"f", 1, "n"}, &Call{"g", 1, "n"}, false},
		{&Call{"f", 1, "n"}, &Call{"f", 1, "n"}, true},
	} {
		if test.c1.Equal(test.c2) != test.result {
			t.Error("Equal error")
		} else if test.c2.Equal(test.c1) != test.result {
			t.Error("Equal not transitive")
		}
	}
}

func testFunc(fail bool) (string, error) {
	if fail {
		return "", errors.New("error")
	} else {
		return "not failed", nil
	}
}

func runNormal(fail bool) (result string, err error) {
	result, err = testFunc(false)
	if err != nil {
		return
	}
	return
}

func runCheck(fail bool) (result string, err error) {
	defer check.Recover(&err)
	result = check.Return(testFunc(fail)).(string)
	return
}

var debugCheck = func() *Checker {
	checker := NewChecker()
	checker.SetFaulter(DebugFaulter{})
	return checker
}()

func runDebug(fail bool) (result string, err error) {
	defer debugCheck.Recover(&err)
	result = debugCheck.Return(testFunc(fail)).(string)
	return
}

func runNoRecoverDebug() {
	debugCheck.Return(testFunc(false))
}

func runNoRecover() {
	check.Return(testFunc(false))
}

func recoverOnly() (err error) {
	defer check.Recover(&err)
	return
}

func recoverOnlyDebug() (err error) {
	defer debugCheck.Recover(&err)
	return
}

func BenchmarkNormalSuccess(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runNormal(false)
	}
}

func BenchmarkNormalFailure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runNormal(true)
	}
}

func BenchmarkCheckReturnFailure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runCheck(true)
	}
}

func BenchmarkCheckReturnOnly(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runNoRecover()
	}
}

func BenchmarkCheckRecoverOnlyNoError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		recoverOnly()
	}
}

func BenchmarkCheckReturnFailureDebug(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runCheck(true)
	}
}

func BenchmarkCheckReturnOnlyDebug(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runNoRecoverDebug()
	}
}

func BenchmarkCheckRecoverOnlyNoErrorDebug(b *testing.B) {
	for i := 0; i < b.N; i++ {
		recoverOnlyDebug()
	}
}
