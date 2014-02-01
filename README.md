Fault
======
[![GoDoc](https://godoc.org/github.com/surullabs/fault?status.png)](https://godoc.org/github.com/surullabs/fault) [![Build Status](https://drone.io/github.com/surullabs/fault/status.png)](https://drone.io/github.com/surullabs/fault/latest) [![Coverage Status](https://coveralls.io/repos/surullabs/fault/badge.png?branch=master)](https://coveralls.io/r/surullabs/fault?branch=master)

Fault is a library that helps with package internal error handling in Go.

It supports a simple idiom for reducing the amount of package internal error
handling. It allows you to panic with a Fault instance in package internal code.
This is then recovered using a defer for a all exported methods, which then return
an error extracted from the fault. As an example, if you were to be reading data
from a file and then writing it you could use

	import (
		"github.com/surullabs/fault"
	)

	var check = fault.Checker{}

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

## Documentation and Examples

Please consult the package [GoDoc](https://godoc.org/github.com/surullabs/fault)
 for detailed documentation.

## Benchmarks

On an 2.000 GHz Intel i7-2630QM CPU there was ~70 ns overhead per CheckReturn call
with 164 ns overhead for the call to Recover(...). The results of the naive benchmarks
in the package are below (formatted for clarity)

	BenchmarkNormalSuccess			200000000		8.50 ns/op
	BenchmarkNormalFailure			200000000		8.98 ns/op
	BenchmarkCheckReturnFailure		2000000			941 ns/op
	BenchmarkCheckReturnOnly		20000000	        77.8 ns/op
	BenchmarkCheckRecoverOnlyNoError	10000000		164 ns/op

If you wish to test the overhead on your machine please run

	go test github.com/surullabs/fault -bench .

## Licensing and Usage

Fault is licensed under a 3-Clause BSD license. Please consult the
LICENSE file for details.

We also ask that you please file bugs and enhancement requests if you run
into any problems. In additon, we're always happy to accept pull requests!
If you do find this useful please share it with others who might also find
it useful. The more users we have the better the software becomes.

