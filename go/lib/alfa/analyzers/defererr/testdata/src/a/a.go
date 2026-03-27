package a

import "os"

func returnsError() error {
	return nil
}

func returnsIntError() (int, error) {
	return 0, nil
}

func returnsNothing() {}

func returnsInt() int {
	return 0
}

// --- Diagnostics expected ---

func deferSingleError() {
	defer returnsError() // want "deferred call to returnsError discards its error return value"
}

func deferMultiReturnError() {
	defer returnsIntError() // want "deferred call to returnsIntError discards its error return value"
}

func deferMethodError() {
	f, _ := os.Open("test")
	defer f.Close() // want "deferred call to Close discards its error return value"
}

// --- No diagnostics expected ---

func deferNoReturn() {
	defer returnsNothing()
}

func deferIntReturn() {
	defer returnsInt()
}

func deferWrappedInClosure() {
	defer func() {
		_ = returnsError()
	}()
}

func deferWrappedCheckingError() error {
	var err error
	defer func() {
		err = returnsError()
	}()
	return err
}

func deferSuppressed() {
	defer returnsError() //defer:err-checked
}

func deferMultiReturnSuppressed() {
	defer returnsIntError() //defer:err-checked
}
