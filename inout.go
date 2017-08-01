package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Input is database import
type Input interface {
	firstRead(string) ([]string, error)
	firstRow([]interface{}) []interface{}
	rowRead([]interface{}) ([]interface{}, error)
}

func (trdsql TRDSQL) dbimport(db *DDB, sqlstr string) (string, error) {
	var err error
	tablenames := sqlparse(sqlstr)
	if len(tablenames) == 0 {
		// without FROM clause. ex. SELECT 1+1;
		debug.Printf("table not found\n")
	}
	for _, tablename := range tablenames {
		sqlstr, err = trdsql.makeTable(db, tablename, sqlstr)
		if err != nil {
			debug.Printf("file not found %s", tablename)
			err = nil
			continue
		}
	}
	return sqlstr, err
}

func (trdsql TRDSQL) makeTable(db *DDB, tablename string, sqlstr string) (string, error) {
	var input Input

	ltsv := false
	if trdsql.iltsv {
		ltsv = true
	} else if trdsql.iguess {
		ltsv = guessExtension(tablename)
	}
	frow := false
	file, err := tFileOpen(tablename)
	if ltsv {
		trdsql.ihead = true
		frow = true
		input, err = trdsql.ltsvInputNew(file)
	} else {
		frow = !trdsql.ihead
		input, err = trdsql.csvInputNew(file)
	}
	if err != nil {
		return sqlstr, err
	}

	var list []interface{}
	for i := 0; i < trdsql.iskip; i++ {
		r, _ := input.rowRead(list)
		debug.Printf("Skip row:%s\n", r)
	}
	rtable := db.escapetable(tablename)
	sqlstr = db.rewrite(sqlstr, tablename, rtable)
	var header []string
	header, err = input.firstRead(rtable)
	db.Create(rtable, header, trdsql.ihead)
	err = db.InsertPrepare(rtable, header, trdsql.ihead)
	if err != nil {
		return sqlstr, err
	}
	list = make([]interface{}, len(header))
	if frow {
		list = input.firstRow(list)
		rowImport(db.stmt, list)
	}
	for {
		list, err = input.rowRead(list)
		if err == io.EOF {
			err = nil
			break
		} else {
			if err != nil {
				return sqlstr, fmt.Errorf("ERROR Read: %s", err)
			}
		}
		rowImport(db.stmt, list)
	}
	return sqlstr, nil
}

// Output is database export
type Output interface {
	first([]interface{}, []string) error
	rowWrite([]interface{}, []string) error
	last()
}

func (trdsql TRDSQL) dbexport(db *DDB, sqlstr string, output Output) error {
	rows, err := db.Select(sqlstr)
	if err != nil {
		return err
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	defer rows.Close()
	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	err = output.first(scanArgs, columns)
	if err != nil {
		return err
	}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return err
		}
		output.rowWrite(values, columns)
	}
	output.last()
	return nil
}

func guessExtension(tablename string) bool {
	pos := strings.LastIndex(tablename, ".")
	if pos > 0 && strings.ToLower(tablename[pos:]) == ".ltsv" {
		debug.Printf("Guess file type as LTSV: [%s]", tablename)
		return true
	}
	debug.Printf("Guess file type as CSV: [%s]", tablename)
	return false
}

func getSeparator(sepString string) (rune, error) {
	if sepString == "" {
		return 0, nil
	}
	sepRunes, err := strconv.Unquote(`'` + sepString + `'`)
	if err != nil {
		return ',', fmt.Errorf("ERROR getSeparator: %s:%s", err, sepString)
	}
	sepRune := ([]rune(sepRunes))[0]
	return sepRune, err
}

func tFileOpen(filename string) (*os.File, error) {
	if filename == "-" {
		return os.Stdin, nil
	}
	if filename[0] == '`' {
		filename = strings.Replace(filename, "`", "", 2)
	}
	if filename[0] == '"' {
		filename = strings.Replace(filename, "\"", "", 2)
	}
	return os.Open(filename)
}

func valString(v interface{}) string {
	var str string
	b, ok := v.([]byte)
	if ok {
		str = string(b)
	} else {
		if v == nil {
			str = ""
		} else {
			str = fmt.Sprint(v)
		}
	}
	return str
}
