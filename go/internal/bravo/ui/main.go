package ui

import (
	"log"
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

var (
	SetOutput = log.SetOutput
	verbose   bool
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
}

// TODO add compile-time determined verbosity for testing / debugging
func SetVerbose(on bool) {
	printerLog.on = on
	printerDebug.on = on
	verbose = on

	if on {
		log.Print("verbose")
	}
}

func SetTesting() {
	SetVerbose(true)
}

func IsVerbose() bool {
	return verbose
}

type Printer = interfaces.Printer

// TODO make utility functions
type DevPrinter interface {
	Printer
	Caller(skip int) Printer
	FunctionName(skip int)
	Stack(skip, count int)
}

var (
	printerOut, printerErr   printer
	printerLog, printerDebug devPrinter
	printerBatsTestBody      devPrinter
	_                        Printer    = printer{}
	_                        Printer    = prefixPrinter{}
	_                        DevPrinter = devPrinter{}
)

func init() {
	printerOut = MakePrinterOn(os.Stdout, true)
	printerErr = MakePrinterOn(os.Stderr, true)

	printerLog = devPrinter{
		printer:       printerErr.withOn(false),
		includesStack: true,
		// includesTime:  true,
	}

	// TODO-P2 determine if on thru compilation
	printerDebug = devPrinter{
		printer:       printerErr,
		includesStack: true,
		// includesTime:  true,
	}

	// TODO-P2 determine thru compilation
	envVarFilter := "BATS_TEST_BODY"
	_, printerBatsTestBodyOn := os.LookupEnv(envVarFilter)

	// TODO-P2 determine thru compilation
	printerBatsTestBody = devPrinter{
		printer:       printerErr.withOn(printerBatsTestBodyOn),
		includesStack: true,
	}
}

func Out() Printer {
	return printerOut
}

func Err() Printer {
	return printerErr
}

func Log() DevPrinter {
	return printerLog
}

func Debug() DevPrinter {
	return printerDebug
}

// TODO replace with a dynamic `DebugOnEnv("BATS_TEST_BODY")`
func DebugBatsTestBody() DevPrinter {
	return printerBatsTestBody
}

func DebugAllowCommit() DevPrinter {
	return printerDebug
}
