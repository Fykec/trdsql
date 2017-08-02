package main

import (
	"io"
	"os"
)

// TRDSQL is output stream define
type TRDSQL struct {
	outStream io.Writer
	errStream io.Writer
	driver    string
	dsn       string
	iguess    bool
	iltsv     bool
	inSep     string
	ihead     bool
	iskip     int
	ifrow     bool
	outHeader bool
	omd       bool
	outSep    string
}

func main() {
	trdsql := &TRDSQL{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(trdsql.Run(os.Args))
}
