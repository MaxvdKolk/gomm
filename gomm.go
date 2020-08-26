package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/james-bowman/sparse"
	"github.com/jlaffaye/ftp"
	"gonum.org/v1/gonum/mat"
)

// MatrixMarket remote URLs and FTP path.
const (
	marketUrl  string = `http://math.nist.gov/MatrixMarket/matrices.html`
	ftpDialUrl string = `math.nist.gov`
	ftpPath    string = `pub/MatrixMarket2/%s/%s/%s.%s`
)

// Supported formats for the MatrixMarket matrices.
const (
	FormatArray      string = "array"
	FormatCoordinate string = "coordinate"
)

// Possible value types for the `MatrixMarket` matrices.
const (
	TypeReal    = "real"
	TypeInteger = "complex"
	TypeComplex = "integer"
	TypePattern = "pattern"
)

// Symmetry properties for the `MatrixMarket` matrices. For general matrices all
// the non-zeroes are provided. For symmetric and skew-symmetric only the
// lower-triangular (including the diagonal) is given.
//
// Note hermitian matrices are not yet supported.
const (
	General       = "general"
	Symmetric     = "symmetric"
	SkewSymmetric = "skew-symmetric"
	Hermitian     = "hermitian"
)

// MatrixMarket represents the MatrixMarket in the sense that it can hold on
// to various instances of `MatrixMarket` matrices.
type MatrixMarket struct {
	Matrices []Matrix
}

// Matrix represents a single matrix from the MatrixMarket. The struct contains
// generic properties obtained by parsing the formats and stores the actual
// matrix using the `mat.Matrix` interface. This can capture both dense (for
// `FormatArray`) and sparse (for `FormatCoordinate`) systems.
type Matrix struct {
	comment    string
	collection string
	set        string
	name       string
	Format     string
	Type       string
	Symmetry   string
	n, m       int

	// The number of non-zeroes in the matrix. This differs from `lines` in
	// the sense that `lines` only provides details on the number of lines
	// parsed in the formats. For some cases, such as `General` type
	// matrices without duplicates or `FormatArray` type matrices, both
	// values might coincide.
	nnz   int
	lines int

	mat mat.Matrix
}

// GetMatrix gets a single matrix from the `MatrixMarket`. The routine requires
// the collection, set, and name of the matrix and attempts to download and
// parse the obtained document. On success a `mat.Matrix` interface is returned
// that either contains a sparse or dense matrix depending on the matrix's type.
func GetMatrix(collection, set, name string) (mat.Matrix, error) {
	matrix := NewMatrix(collection, set, name)
	if err := matrix.Download(); err != nil {
		return nil, err
	}

	f, err := os.Open(matrix.Filename())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rd, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	mat, err := matrix.Parse(rd)
	if err != nil {
		return nil, err
	}

	return mat, nil
}

// NewMatrixMarket creates a local representation of the `MatrixMarket`. It
// forms a list of all available matrices from the `/MatrixMarket/data/` page.
func NewMatrixMarket() (*MatrixMarket, error) {
	list, err := GetMatrixMarket()
	if err != nil {
		return nil, err
	}
	defer list.Close()

	market := new(MatrixMarket)

	scanner := bufio.NewScanner(list)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `<A HREF="/MatrixMarket/data/`) {
			m, err := ParseEntry(line)
			if err != nil {
				log.Printf("Failed to parse: %#v\n", line)
			}
			market.Matrices = append(market.Matrices, m)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return market, nil
}

// NewMatrix provides a `Matrix` struct initialised with a collection, set, and
// name.
func NewMatrix(collection, set, name string) Matrix {
	return Matrix{collection: collection, set: set, name: name}
}

// Dims returns the dimensions of the matrix `(rows, cols)`.
func (matrix *Matrix) Dims() (int, int) {
	return matrix.n, matrix.m
}

// At returns the value of the matrix at `(i,j)` using the matrix interface.
func (matrix *Matrix) At(i, j int) float64 {
	return matrix.mat.At(i, j)
}

// NNZ returns the number of non-zeroes of the matrix.
func (matrix *Matrix) NNZ() int {
	return matrix.nnz
}

// Filename forms the filename of the matrix. Currently, the code only processes
// the `MatrixMarket` format and the extensions are hardcoded to `.mtx.gz`.
func (matrix *Matrix) Filename() string {
	return fmt.Sprintf("%s.mtx.gz", matrix.name)
}

// Download downloads the matrix to disk.
func (market *MatrixMarket) Download(m Matrix) error {
	return m.Download()
}

// GetMatrixMarket reads the body of the response for a matrix request.
func GetMatrixMarket() (io.ReadCloser, error) {
	resp, err := http.Get(marketUrl)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// ParseEntry parses a single entry in the list of `MatrixMarket` matrices and
// forms a new Matrix given the obtained collection, set, and name.
func ParseEntry(line string) (Matrix, error) {
	res := strings.Split(strings.Split(line, `"`)[1], "/")
	if len(res) != 6 {
		return Matrix{}, nil
	}

	// split .html
	name := strings.Split(res[5], ".")[0]

	return Matrix{
		collection: res[3],
		set:        res[4],
		name:       name,
	}, nil
}

// Download a single matrix to disk. This stores the matrix as a `gz` compressed
// file.
func (m *Matrix) Download() error {
	c, err := ftp.Dial(ftpDialUrl + `:21`)
	if err != nil {
		return err
	}

	err = c.Login("anonymous", "anonymous")
	if err != nil {
		return err
	}

	// TODO can be harwell-boeing or matrixmarket format...
	f, err := c.Retr(fmt.Sprintf(ftpPath, m.collection, m.set, m.name, "mtx.gz"))
	if err != nil {
		return err
	}
	defer f.Close()

	file, err := os.Create(fmt.Sprintf("%s.mtx.gz", m.name))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, f)
	return err
}

// Path returns the formatted path of the matrix.
func (matrix *Matrix) Path() string {
	// TODO: consider other formats
	return fmt.Sprintf("%s.mtx.gz", matrix.name)
}

// String forms a string representation of the matrix.
func (matrix *Matrix) String() string {
	format := "Matrix `%s`: format: `%s`, type: `%s`\n"
	return fmt.Sprintf(format, matrix.name, matrix.Format, matrix.Type)
}

// ParseHeader attempts to parse a header of the `MatrixMarket` format. This
// extracts the first line from the provided `Reader`.
func (matrix *Matrix) ParseHeader(buf *bufio.Reader) error {
	// read first line
	b, err := buf.ReadBytes('\n')
	if err != nil {
		if err != io.EOF {
			return err
		}
	}
	tokens := strings.Split(strings.TrimSpace(string(b)), " ")

	// for 'matrix' objects we expect four tokens in the header
	if len(tokens) != 5 {
		return fmt.Errorf("Wrong number of header tokens: %#v (%d), exp: 5", tokens, len(tokens))
	}

	// start header
	if !strings.EqualFold(tokens[0], "%%MatrixMarket") {
		return fmt.Errorf("Expected header '%%MatrixMarket', got %s", tokens[0])
	}

	// object
	if !strings.EqualFold(tokens[1], "matrix") {
		return fmt.Errorf("Unsupported object: %v, expected 'matrix'", tokens[1])
	}

	// format
	switch strings.ToLower(tokens[2]) {
	case FormatArray:
		matrix.Format = FormatArray
	case FormatCoordinate:
		matrix.Format = FormatCoordinate
	default:
		return fmt.Errorf("Unsupported format: %v", tokens[2])
	}

	// element type
	switch strings.ToLower(tokens[3]) {
	case TypeReal:
		matrix.Type = TypeReal // float64
	case TypeComplex:
		matrix.Type = TypeComplex // complex?!
	case TypeInteger:
		matrix.Type = TypeInteger // int
	case TypePattern:
		matrix.Type = TypePattern // bool
	default:
		return fmt.Errorf("Unsupported format: %v", tokens[3])
	}

	// matrix type
	switch strings.ToLower(tokens[4]) {
	case General:
		matrix.Symmetry = General
	case Symmetric:
		matrix.Symmetry = Symmetric
	case SkewSymmetric:
		matrix.Symmetry = SkewSymmetric
	case Hermitian:
		matrix.Symmetry = Hermitian
	default:
		return fmt.Errorf("Unsupported matrix symmetry: %v", tokens[4])
	}

	return nil
}

// ParseComment consumes all comment and/or empty lines from the reader. The
// comments are stored, in case they contain valuable information. EOF is not
// treated as error, it simply terminates the processing of comments.
func (matrix *Matrix) ParseComment(buf *bufio.Reader) error {

	var comment bytes.Buffer

loop:
	for {
		// EOF is not an error; should just terminate
		b, err := buf.Peek(1)
		if err != nil {
			if err == io.EOF {
				break loop
			}
			return err
		}

		switch b[0] {
		case '%', '\n', ' ', '\t':
			// consume and store comment and empty lines
			b, err := buf.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					return err
				}
			}
			comment.Write(b)
		default:
			break loop
		}
	}

	if comment.Len() > 0 {
		matrix.comment = comment.String()
	}
	return nil
}

// ParseDimensions parses the dimensions and expected number of lines.
func (matrix *Matrix) ParseDimensions(buf *bufio.Reader) error {
	line, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	dims := strings.Split(strings.TrimSpace(line), " ")
	if len(dims) < 2 {
		return fmt.Errorf("Expect at least two values: (n, m, _), got: %v", dims)
	}

	n, err := strconv.Atoi(dims[0])
	if err != nil {
		return err
	}
	matrix.n = n

	m, err := strconv.Atoi(dims[1])
	if err != nil {
		return err
	}
	matrix.m = m

	// `FormatArray` is dense, thus the number of lines is already known
	if matrix.Format == FormatArray {
		matrix.lines = matrix.n * matrix.m
		return nil
	}

	// For other formats the matrix entries might overlap, e.g. the COO
	// triplets are to be summed, or only a subset of symmetric matrices
	// are provided. Thus the number of expected lines is parsed.
	if len(dims) < 3 {
		return fmt.Errorf("Expect at least three values: (n, m, v), got: %v", dims)
	}
	lines, err := strconv.Atoi(dims[2])
	if err != nil {
		return err
	}
	matrix.lines = lines
	return nil
}

// ParseMatrix performs the parsing of the body of the matrix. This routine
// invokes a specialised routine, depending on the matrix format, to perform
// the actual parsing.
func (matrix *Matrix) ParseMatrix(buf *bufio.Reader) error {
	switch matrix.Format {
	case FormatCoordinate:
		return matrix.ParseCoordinate(buf)
	case FormatArray:
		return matrix.ParseArrayFormat(buf)
	default:
		return fmt.Errorf("not supported format %#v", matrix.Format)
	}
}

// splitTriplet splits a COO-triplet of (i, j, v) form from strings to two
// integer indices (i, j) and the matching floating point value (v).
func splitTriplet(s string) (i int, j int, v float64, err error) {
	splits := strings.Fields(strings.TrimSpace(s))
	if len(splits) != 3 {
		return i, j, v, fmt.Errorf("Too little entries to unpack triplet %d, %s", len(splits), splits)
	}

	i, err = strconv.Atoi(splits[0])
	if err != nil {
		return i, j, v, err
	}

	j, err = strconv.Atoi(splits[1])
	if err != nil {
		return i, j, v, err
	}

	v, err = strconv.ParseFloat(splits[2], 64)
	if err != nil {
		return i, j, v, err
	}

	return i, j, v, nil
}

// ParseCoordinate parses a `MatrixMarket` of the `Coordinate` format.
func (matrix *Matrix) ParseCoordinate(buf *bufio.Reader) error {
	// fill COO
	n, m := matrix.Dims()
	if n == 0 || m == 0 {
		return fmt.Errorf("Matrix dimensions are empty (%d, %d)", n, m)
	}

	// estimate number of non-zeros by number of lines in file
	nnz := matrix.lines
	I, J, V := make([]int, 0, nnz), make([]int, 0, nnz), make([]float64, 0, nnz)
	coo := sparse.NewCOO(n, m, I, J, V)

	// exhaust all lines with scanner
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		i, j, v, err := splitTriplet(scanner.Text())
		if err != nil {
			return err
		}

		// prevent inserting explicit zeros
		// FIXME: not sure if `SmallestNonzeroFloat64` makes sense
		if math.Abs(v) < math.SmallestNonzeroFloat64 {
			continue
		}

		// correct for one-base
		coo.Set(i-1, j-1, v)

		// for symmetric types also insert its symmetric counterpart
		if i != j {
			switch matrix.Symmetry {
			case Symmetric:
				coo.Set(j-1, i-1, v)
			case SkewSymmetric:
				coo.Set(j-1, i-1, -v)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// return CSR
	matrix.mat = coo.ToCSR()
	return nil
}

// ParseArrayFormat parses a `MatrixMarket` of format `Array`.
func (matrix *Matrix) ParseArrayFormat(buf *bufio.Reader) error {
	// prepare dense matrix
	n, m := matrix.Dims()
	if n == 0 || m == 0 {
		return fmt.Errorf("Matrix dimensions are empty (%d, %d)", n, m)
	}

	//mat := mat.NewDense(n, m, nil)
	values := make([]float64, n*m)

	// exhaust all lines with scanner
	scanner := bufio.NewScanner(buf)
	cnt := 0
	for scanner.Scan() {
		v, err := strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64)
		if err != nil {
			return err
		}

		values[cnt] = v
		cnt++
	}

	// Construct a dense matrix where the extracted values are put in the
	// right order, as the ordering of `MatrixMarket` is column-major,
	// whereas `mat.NewDense` would assume row-major.
	mm := mat.NewDense(n, m, nil)
	for c := 0; c < m; c++ {
		for r := 0; r < n; r++ {
			mm.Set(r, c, values[c*n+r])
		}
	}
	matrix.mat = mm
	return nil
}

// Parse parses the matrix form a `Reader` by parsing the header, comment,
// dimensions, and finally the body of the matrix. The parsed information is
// stored in the matrix. If all steps complete without error the matrix
// interface is returned.
func (matrix *Matrix) Parse(rd io.Reader) (mat.Matrix, error) {
	buf := bufio.NewReader(rd)

	if err := matrix.ParseHeader(buf); err != nil {
		return nil, err
	}

	if err := matrix.ParseComment(buf); err != nil {
		return nil, err
	}

	if err := matrix.ParseDimensions(buf); err != nil {
		return nil, err
	}

	// it is expected to exhaust the reader till EOF
	err := matrix.ParseMatrix(buf)
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	}
	return matrix.mat, nil
}

// SaveToMatrixMarket writes a `mat.Matrix` interface towards the `MatrixMarket`
// format. Currently, all matrices are written as `coordinate real general`
// types.
//
// TODO: support (skew)symmetric outputs
// TODO: support dense matrix outputs
func SaveToMatrixMarket(matrix mat.Matrix, wr io.Writer) error {
	// bufferend output
	buf := bufio.NewWriter(wr)

	// sparse variant
	csr, ok := matrix.(*sparse.CSR)
	if ok {
		// MatrixMarket header
		header := fmt.Sprintf("%%%%MatrixMarket matrix %s %s %s\n", FormatCoordinate, TypeReal, General)
		buf.WriteString(header)

		// Matrix dimensions and number of lines of output
		n, m := csr.Dims()
		nnz := csr.NNZ()
		buf.WriteString(fmt.Sprintf("%d %d %d\n", n, m, nnz))

		// Apply write function to each non-zero
		writeNonZero := func(i, j int, v float64) {
			// Correct for one-base
			buf.WriteString(fmt.Sprintf("%d %d %v\n", i+1, j+1, v))
		}
		csr.DoNonZero(writeNonZero)

		return buf.Flush()
	}

	// dense variant
	dense, ok := matrix.(*mat.Dense)
	if ok {
		header := fmt.Sprintf("%%%%MatrixMarket matrix %s %s %s\n", FormatArray, TypeReal, General)
		buf.WriteString(header)

		// Matrix dimensions and number of lines of output
		n, m := dense.Dims()
		buf.WriteString(fmt.Sprintf("%d %d\n", n, m))

		for c := 0; c < m; c++ {
			for r := 0; r < n; r++ {
				buf.WriteString(fmt.Sprintf("%v\n", dense.At(r, c)))
			}
		}
		return buf.Flush()
	}

	// support dense variant later
	return fmt.Errorf("No output support yet for dense matrices.")
}
