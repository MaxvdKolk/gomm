package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/james-bowman/sparse"
	"gonum.org/v1/gonum/mat"
)

type entry struct {
	str    []byte
	matrix Matrix
	err    bool
}

func (e entry) String() string {
	return fmt.Sprintf("{input: '%q', expected matrix: %v", e.str, e.matrix)
}

func TestParsMatrixMarketArrayFormat(t *testing.T) {
	mm := []byte(`%%MatrixMarket matrix array real general
4 3
1.0
2.0
3.0
4.0
5.0
6.0
7.0
8.0
9.0
10.0
11.0
12.0`)

	ref := mat.NewDense(4, 3, nil)
	cnt := 1.0
	for c := 0; c < 3; c++ {
		for r := 0; r < 4; r++ {
			ref.Set(r, c, cnt)
			cnt += 1.0
		}
	}

	matrix := &Matrix{}
	smat, err := matrix.Parse(bufio.NewReader(bytes.NewBuffer(mm)))
	if err != nil {
		t.Errorf("Error in parsing matrix: %v", err)
	}

	n, m := matrix.Dims()
	if n != 4 || m != 3 {
		t.Errorf("Wrong matrix dimensions: (%d, %d), exp: (%d, %d)", n, m, 5, 5)
	}
	if matrix.lines != 12 {
		t.Errorf("Wrong number of lines parsed: %d, exp: %d", matrix.lines, 12)
	}

	if matrix.mat == nil {
		t.Fatal("Matrix interface is nil after parsing")
	}

	if !mat.Equal(ref, matrix.mat) {
		t.Logf("Expected:\n%v\n but created:\n%v\n", mat.Formatted(ref), mat.Formatted(matrix.mat))
		t.Errorf("Wrong content")
	}

	dense, ok := smat.(*mat.Dense)
	if !ok {
		t.Errorf("Failed conversion matrix interface to expected type %T, from %T", dense, smat)
	}

}

func TestParseMatrixMarketCoordinate(t *testing.T) {

	/*
		This requires the following extensions:
		- integer, complex, pattern style matrices;
		  export all simply as float
		- support symmetric / skew-symmetric, perform post operations
		- export array format to dense matrices
		- ensure the output is always a Matrix
	*/

	mm := []byte(`%%MatrixMarket matrix coordinate real general
% A 5x5 sparse matrix with 8 nonzeros
5 5 8
1 1     1.0
2 2     10.5
4 2     250.5
3 3     0.015
1 4     6.0
4 4     -280.0
4 5     33.32
5 5     12.0`)

	ref := sparse.NewCOO(5, 5, make([]int, 8), make([]int, 8), make([]float64, 8))
	ref.Set(0, 0, 1.0)
	ref.Set(1, 1, 10.5)
	ref.Set(3, 1, 250.5)
	ref.Set(2, 2, 0.015)
	ref.Set(0, 3, 6.0)
	ref.Set(3, 3, -280.0)
	ref.Set(3, 4, 33.32)
	ref.Set(4, 4, 12.0)

	matrix := &Matrix{}
	smat, err := matrix.Parse(bufio.NewReader(bytes.NewBuffer(mm)))
	if err != nil {
		t.Errorf("Error in parsing matrix: %v", err)
	}

	n, m := matrix.Dims()
	if n != 5 || m != 5 {
		t.Errorf("Wrong matrix dimensions: (%d, %d), exp: (%d, %d)", n, m, 5, 5)
	}
	if matrix.lines != 8 {
		t.Errorf("Wrong number of lines parsed: %d, exp: %d", matrix.NNZ(), 8)
	}

	if matrix.mat == nil {
		t.Fatal("Matrix interface is nil after parsing")
	}

	if !mat.Equal(ref, matrix.mat) {
		t.Logf("Expected:\n%v\n but created:\n%v\n", mat.Formatted(ref), mat.Formatted(matrix.mat))
		t.Errorf("Wrong content")
	}

	csr, ok := smat.(*sparse.CSR)
	if !ok {
		t.Errorf("Failed conversion matrix interface to expected type %T, from %T", csr, smat)
	}
}

func TestParseMatrixMarketDimensions(t *testing.T) {
	entries := []entry{
		entry{ // valid
			str: []byte("5 6 7\n"),
			matrix: Matrix{
				n:      5,
				m:      6,
				lines:  7,
				Format: FormatCoordinate,
			},
		},
		entry{ // invalid
			str: []byte("5 6\n"),
			err: true,
			matrix: Matrix{
				n:      5,
				m:      6,
				Format: FormatCoordinate,
			},
		},
		entry{ // valid for `FormatArray`
			str: []byte("5 6\n"),
			matrix: Matrix{
				n:      5,
				m:      6,
				lines:  30,
				Format: FormatArray,
			},
		},
		entry{ // invalid for `FormatArray`
			str: []byte("5\n"),
			err: true,
			matrix: Matrix{
				Format: FormatArray,
			},
		},
	}

	for _, entry := range entries {
		matrix := &Matrix{}
		matrix.Format = entry.matrix.Format
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
		if matrix.lines != entry.matrix.lines {
			t.Errorf("Wrong number of lines parsed: got %d, exp %d", matrix.nnz, entry.matrix.nnz)
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
		entry{ // some consequetive newlines with spaces
			str: []byte("%Hello\n    \n\n\n%World!"),
			matrix: Matrix{
				comment: "%Hello\n    \n\n\n%World!",
			},
		},
		entry{ // some consequetive newlines with tabs
			str: []byte("%Hello\n\t\n\n\n%World!"),
			matrix: Matrix{
				comment: "%Hello\n\t\n\n\n%World!",
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

// Complete parse: download, unzip, parse, verify.
func TestParseMatrixMarketFormat(t *testing.T) {
	type RefMatrix struct {
		Matrix
		n, m int
		nnz  int
	}

	// selection of test matrices
	matrices := []RefMatrix{
		RefMatrix{ // coordinate real unsymmetric
			Matrix{
				collection: "Harwell-Boeing",
				set:        "lns",
				name:       "lns__131",
			},
			131, 131, 536,
		},
		RefMatrix{ // coordinate real unsymmetric with explicit zeros
			Matrix{
				collection: "Harwell-Boeing",
				set:        "nnceng",
				name:       "hor__131",
			},
			434, 434, 4182,
		},
		RefMatrix{ // coordinate real symmetric positive definite
			Matrix{
				collection: "Harwell-Boeing",
				set:        "bcsstruc1",
				name:       "bcsstk01",
			},
			48, 48, 400,
		},
		RefMatrix{ // coordinate real skew-symmetric
			Matrix{
				collection: "Harwell-Boeing",
				set:        "platz",
				name:       "plsk1919",
			},
			1919, 1919, 9662,
		},
		RefMatrix{ // coordinate real unsymmetric, more dense
			Matrix{
				collection: "Harwell-Boeing",
				set:        "astroph",
				name:       "mcca",
			},
			180, 180, 2659,
		},
		RefMatrix{ // coordinate real unsymmetric, nrows > ncols
			Matrix{
				collection: "Harwell-Boeing",
				set:        "lsq",
				name:       "illc1033",
			},
			1033, 320, 4719,
		},
		RefMatrix{ // coordinate real unsymmetric, ncols > nrows
			Matrix{
				collection: "Harwell-Boeing",
				set:        "econiea",
				name:       "wm1",
			},
			207, 277, 2909,
		},
		RefMatrix{ // coordinate real unsymmetric, ncols > nrows, almost dense
			Matrix{
				collection: "Harwell-Boeing",
				set:        "econiea",
				name:       "beause",
			},
			497, 507, 44551,
		},
		// TODO: pattern style tests
	}

	for _, matrix := range matrices {
		file := matrix.Filename()
		t.Logf("Processing: %v", matrix.Filename())
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if err := matrix.Download(); err != nil {
				t.Fatal(err)
			}
		}

		mm, err := GetMatrix(matrix.collection, matrix.set, matrix.name)
		if err != nil {
			t.Fatal(err)
		}

		csr, ok := mm.(*sparse.CSR)
		if !ok {
			t.Errorf("Failed conversion %T, from %T", csr, mm)
		}

		n, m := mm.Dims()
		if n != matrix.n || m != matrix.m {
			t.Errorf("Wrong dimensions: exp: (%v, %v), got: (%v, %v)", matrix.n, matrix.m, n, m)
		}

		if csr.NNZ() != matrix.nnz {
			t.Errorf("Wrong number of non-zero entries: exp %v, got %v", matrix.nnz, csr.NNZ())
		}

		if err := os.Remove(file); err != nil {
			t.Error(err)
		}
	}
}

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
	if _, err := os.Stat(matrix.Filename()); os.IsNotExist(err) {
		t.Error(err)
	}
	if err := os.Remove(matrix.Filename()); err != nil {
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

func TestWriteMatrixMarketFormat(t *testing.T) {
	// expected outcome, after reordering and formatting of output
	mm := []byte(
		`%%MatrixMarket matrix coordinate real general
5 5 8
1 1 1
1 4 6
2 2 10.5
3 3 0.015
4 2 250.5
4 4 -280
4 5 33.32
5 5 12
`)

	// COO representation of the test output matrix
	ref := sparse.NewCOO(5, 5, make([]int, 8), make([]int, 8), make([]float64, 8))
	ref.Set(0, 0, 1.0)
	ref.Set(1, 1, 10.5)
	ref.Set(3, 1, 250.5)
	ref.Set(2, 2, 0.015)
	ref.Set(0, 3, 6.0)
	ref.Set(3, 3, -280.0)
	ref.Set(3, 4, 33.32)
	ref.Set(4, 4, 12.0)

	// create test file
	f, err := os.Create("test_write.mtx")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// populate with matrix
	csr := ref.ToCSR()
	err = SaveToMatrixMarket(csr, f)
	if err != nil {
		t.Fatal(err)
	}

	// compare both streams byte wise
	refReader := bufio.NewReader(bytes.NewBuffer(mm))
	matReader, err := os.Open("test_write.mtx")
	if err != nil {
		t.Fatal(err)
	}

	// FIXME: might be much for this simple test
	for {
		b1 := make([]byte, 64000)
		_, err1 := refReader.Read(b1)
		b2 := make([]byte, 64000)
		_, err2 := matReader.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				// end of files
				break
			} else {
				t.Logf("error: buffer 1: %v, buffer 2: %v", err1, err2)
			}
		}

		if !bytes.Equal(b1, b2) {
			t.Logf("Reference bytes: %v", b1)
			t.Logf("Custom bytes: %v", b2)
			t.Errorf("bytes are not equal")
		}
	}

	if err := os.Remove("test_write.mtx"); err != nil {
		t.Error(err)
	}
}
