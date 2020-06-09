package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

type entry struct {
	str    []byte
	matrix Matrix
	err    bool
}

func (e entry) String() string {
	return fmt.Sprintf("{input: '%q', expected matrix: %v", e.str, e.matrix)
}

func TestParseMatrixMarketArray(t *testing.T) {
	t.Errorf("Cannot parse array format yet")
}

func TestParseMatrixMarketCoordinate(t *testing.T) {
	t.Errorf("Cannot parse coordinate format yet")
}

func TestParseMatrixMarketDimensions(t *testing.T) {
	entries := []entry{
		entry{ // valid
			str: []byte("5 6 7\n"),
			matrix: Matrix{
				n:   5,
				m:   6,
				nnz: 7,
			},
		},
		entry{ // invalid
			str: []byte("5 6\n"),
			err: true,
		},
	}

	for _, entry := range entries {
		matrix := &Matrix{}
		err := matrix.ParseDimensions(bufio.NewReader(bytes.NewBuffer(entry.str)))

		if entry.err {
			if err == nil {
				t.Errorf("Expected to fail: %+v", entry)
			}
		} else {
			if err != nil {
				t.Error(err)
			}
		}
		if matrix.n != entry.matrix.n {
			t.Errorf("Wrong first dimension: got %d, exp %d", matrix.n, entry.matrix.n)
		}
		if matrix.m != entry.matrix.m {
			t.Errorf("Wrong second dimension: got %d, exp %d", matrix.m, entry.matrix.m)
		}
		if matrix.nnz != entry.matrix.nnz {
			t.Errorf("Wrong NNZ: got %d, exp %d", matrix.nnz, entry.matrix.nnz)
		}
	}
}

func TestParseMatrixMarketComment(t *testing.T) {
	entries := []entry{
		entry{ // valid
			str: []byte("%Hello\n%World!\n10 10 10"),
			matrix: Matrix{
				comment: "%Hello\n%World!\n",
			},
		},
		entry{ // no following lines
			str: []byte("%Hello\n%World!"),
			matrix: Matrix{
				comment: "%Hello\n%World!",
			},
		},
		entry{ // some consequetive newlines
			str: []byte("%Hello\n\n\n\n%World!"),
			matrix: Matrix{
				comment: "%Hello\n\n\n\n%World!",
			},
		},
		entry{ // emtpy line
			str:    []byte(""),
			matrix: Matrix{},
		},
	}

	for _, entry := range entries {
		matrix := &Matrix{}
		err := matrix.ParseComment(bufio.NewReader(bytes.NewBuffer(entry.str)))
		if err != nil {
			t.Error(err)
		}
		if matrix.comment != entry.matrix.comment {
			t.Errorf("Expected comment: %#v, got: %#v", entry.matrix.comment, matrix.comment)
		}
	}
}

func TestParseMatrixMarketHeader(t *testing.T) {

	entries := []entry{
		// valid headers...
		entry{
			str: []byte("%%MatrixMarket matrix coordinate real general"),
			matrix: Matrix{
				Format:   FormatCoordinate,
				Type:     TypeReal,
				Symmetry: General,
			},
		},
		entry{
			str: []byte("%%MatrixMarket matrix array pattern general"),
			matrix: Matrix{
				Format:   FormatArray,
				Type:     TypePattern,
				Symmetry: General,
			},
		},
		entry{ // lower case %%MatrixMarket should also pass
			str: []byte("%%matrixmarket matrix array pattern general"),
			matrix: Matrix{
				Format:   FormatArray,
				Type:     TypePattern,
				Symmetry: General,
			},
		},
		// faulty headers...
		entry{ // empty header
			str: []byte(""),
			err: true,
		},
		entry{ // no obj
			str: []byte("%%MatrixMarket"),
			err: true,
		},
		entry{ // missing '%%MatrixMarket' at beginning
			str: []byte("%MatrixMarket matrix coordinate real general"),
			err: true,
		},
		entry{ // EOF directly
			str: []byte(`%MatrixMarket matrix coordinate real general`),
			err: true,
		},
		entry{ // no 'matrix'
			str: []byte("%%MatrixMarket m coordinate real general"),
			err: true,
		},
		entry{ // no 'format'
			str: []byte("%%MatrixMarket m c real general"),
			err: true,
		},
		entry{ // no 'type'
			str: []byte("%%MatrixMarket matrix coordinate r general"),
			err: true,
		},
		entry{ // no 'symmetry'
			str: []byte("%%MatrixMarket matrix coordinate real g"),
			err: true,
		},
	}

	for _, entry := range entries {
		rd := bufio.NewReader(bytes.NewBuffer(entry.str))
		matrix := &Matrix{}
		err := matrix.ParseHeader(rd)

		// ensure faulty return error
		if entry.err {
			if err == nil {
				t.Errorf("Expected error, but got none")
			}
			continue
		}

		// ensure non-faulty lines proceed without error
		if err != nil {
			t.Errorf("%v, %v", err, entry.err)
		}
		if matrix.Format != entry.matrix.Format {
			t.Errorf("Wrong format: exp %#v, got %#v", entry.matrix.Format, matrix.Format)
		}
		if matrix.Type != entry.matrix.Type {
			t.Errorf("Wrong type: exp %#v, got %#v", entry.matrix.Type, matrix.Type)
		}
		if matrix.Symmetry != entry.matrix.Symmetry {
			t.Errorf("Wrong symmetry: exp %#v, got %#v", entry.matrix.Symmetry, matrix.Symmetry)
		}
	}
}

/* TODO: add as soon as the components are working
func TestParseMatrixMarketFormat(t *testing.T) {
	matrices := []Matrix{
		Matrix{ // real unsymmetric
			collection: "Harwell-Boeing",
			set:        "nnceng",
			name:       "hor__131",
		},
		// pattern style -- do later
		//matrix := Matrix{
		//	collection: "Harwell-Boeing",
		//	set:        "smtape",
		//	name:       "ibm32",
		//}
	}

	for _, matrix := range matrices {
		file := fmt.Sprintf("%s.mtx.gz", matrix.name)
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if err := matrix.Download(); err != nil {
				t.Fatal(err)
			}
		}

		if err := matrix.Parse(); err != nil {
			t.Error(err)
		}
	}
}
*/

func TestDownloadMatrix(t *testing.T) {
	matrix := Matrix{
		collection: "Harwell-Boeing",
		set:        "smtape",
		name:       "ash608",
	}
	t.Log("Downloading...")
	if err := matrix.Download(); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(fmt.Sprintf("%s.mtx.gz", matrix.name)); os.IsNotExist(err) {
		t.Error(err)
	}
	if err := os.Remove(fmt.Sprintf("%s.mtx.gz", matrix.name)); err != nil {
		t.Error(err)
	}
}

func TestParseList(t *testing.T) {
	market, err := NewMatrixMarket()
	if err != nil {
		t.Error(err)
	}
	if len(market.Matrices) != 498 {
		msg := "Wrong number of matrices encountered: got %d, exp %d"
		t.Errorf(msg, len(market.Matrices), 498)
	}
}

func TestParseHREF(t *testing.T) {
	type entry struct {
		str    string
		matrix Matrix
	}

	entries := []entry{
		entry{
			str: `<A HREF="/MatrixMarket/data/Harwell-Boeing/smtape/ash608.html">ASH608</A><BR>`,
			matrix: Matrix{
				collection: "Harwell-Boeing",
				set:        "smtape",
				name:       "ash608",
			},
		},
		entry{
			str: `<A HREF="/MatrixMarket/data/Harwell-Boeing/smtape/shl____0.html">SHL    0</A><BR>`,
			matrix: Matrix{
				collection: "Harwell-Boeing",
				set:        "smtape",
				name:       "shl____0",
			},
		},
	}

	for _, e := range entries {
		m, err := ParseEntry(e.str)
		if err != nil {
			t.Error(err)
		}

		if !strings.EqualFold(m.collection, e.matrix.collection) {
			t.Errorf("Wrong collection: exp %#v, got %#v", e.matrix.collection, m.collection)
		}
		if !strings.EqualFold(m.set, e.matrix.set) {
			t.Errorf("Wrong set: exp %#v, got %#v", e.matrix.set, m.set)
		}
		if !strings.EqualFold(m.name, e.matrix.name) {
			t.Errorf("Wrong name: exp %#v, got %#v", e.matrix.name, m.name)
		}

	}
}
