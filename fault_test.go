// Copyright 2014, Surul Software Labs GmbH
// All rights reserved.

package fault

import (
	"errors"
	"testing"
)

var check FaultCheck = Checker{}

func TestErrorChain(t *testing.T) {
	for _, test := range []struct {
		name string
		test func() *ErrorChain
		err  string
	}{
		{
			"Test one error",
			func() *ErrorChain { return &ErrorChain{chain: []error{errors.New("error1")}} },
			"error1",
		},
		{
			"Test two errors",
			func() *ErrorChain { return &ErrorChain{chain: []error{errors.New("error1"), errors.New("error2")}} },
			"error1; error2",
		},
		{
			"Test three errors",
			func() *ErrorChain {
				return &ErrorChain{chain: []error{errors.New("error1"), errors.New("error2"), errors.New("error3")}}
			},
			"error1; error2; error3",
		},
		{
			"Test error chain nil",
			func() *ErrorChain { return &ErrorChain{} },
			"",
		},
		{
			"Test chain nil",
			func() *ErrorChain {
				chain := &ErrorChain{chain: nil}
				chain.Chain(errors.New("error1"))
				return chain
			},
			"error1",
		},
		{
			"Test chain nil call",
			func() *ErrorChain {
				chain := &ErrorChain{chain: []error{errors.New("error1")}}
				chain.Chain(nil)
				return chain
			},
			"error1",
		},
		{
			"Test chain one error",
			func() *ErrorChain {
				chain := &ErrorChain{chain: []error{errors.New("error1")}}
				chain.Chain(errors.New("error2"))
				return chain
			},
			"error1; error2",
		},
		{
			"Test chain multi nil",
			func() *ErrorChain {
				chain := &ErrorChain{chain: []error{errors.New("error1")}}
				chain.Chain(&ErrorChain{})
				return chain
			},
			"error1",
		},
		{
			"Test chain multi",
			func() *ErrorChain {
				chain := &ErrorChain{chain: []error{errors.New("error1")}}
				chain.Chain(&ErrorChain{chain: []error{errors.New("error2"), errors.New("error3")}})
				return chain
			},
			"error1; error2; error3",
		},
		{
			"Test chainer empty",
			func() *ErrorChain { return Chain() },
			"",
		},
		{
			"Test chainer one",
			func() *ErrorChain { return Chain(errors.New("error1")) },
			"error1",
		},
		{
			"Test chainer two",
			func() *ErrorChain { return Chain(errors.New("error1"), errors.New("error2")) },
			"error1; error2",
		},
		{
			"Test chainer error chain",
			func() *ErrorChain { return Chain(errors.New("error1"), Chain(errors.New("error2"))) },
			"error1; error2",
		},
		{
			"Test chainer error chain first",
			func() *ErrorChain { return Chain(Chain(errors.New("error1")), errors.New("error2")) },
			"error1; error2",
		},
		{
			"Test chainer error chain multi",
			func() *ErrorChain { return Chain(Chain(errors.New("error1")), Chain(errors.New("error2"))) },
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
	chain := Chain(errors.New("error1"), errors.New("error2"))
	if len(chain.Errors()) != 2 {
		t.Error("Invalid chain found", chain.Errors())
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
	err := Chain(errors.New("err1"))
	if err.Error() != err.String() || err.Error() != "err1" {
		t.Error("Error string does not match")
	}

	fault := errorFault{err: nil}
	if fault.String() != "" {
		t.Error("Fault string not empty")
	}
	fault.err = err
	if fault.String() != "err1" {
		t.Error("Fault string mismatch")
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

func runNoRecover() {
	check.Return(testFunc(false))
}

func recoverOnly() (err error) {
	defer check.Recover(&err)
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
