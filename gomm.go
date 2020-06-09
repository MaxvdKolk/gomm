package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jlaffaye/ftp"
)

const marketUrl string = `http://math.nist.gov/MatrixMarket/matrices.html`
const ftpDialUrl string = `math.nist.gov`
const ftpPath string = `pub/MatrixMarket2/%s/%s/%s.%s`

const (
	FormatArray      string = "array"
	FormatCoordinate string = "coordinate"
)

const (
	TypeReal    = "real"
	TypeInteger = "complex"
	TypeComplex = "integer"
	TypePattern = "pattern"
)

const (
	General       = "general"
	Symmetric     = "symmetric"
	SkewSymmetric = "skew-symmetric"
	Hermitian     = "hermitian"
)

type MatrixMarket struct {
	Matrices []Matrix
}

type Matrix struct {
	comment    string
	collection string
	set        string
	name       string
	Format     string
	Type       string
	Symmetry   string
	n, m       int
	nnz        int
	// mat *Matrix?
}

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

func (market *MatrixMarket) Download(m Matrix) error {
	return m.Download()
}

func GetMatrixMarket() (io.ReadCloser, error) {

	resp, err := http.Get(marketUrl)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

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

func (matrix *Matrix) Path() string {
	// TODO: consider other formats
	return fmt.Sprintf("%s.mtx.gz", matrix.name)
}

func (matrix *Matrix) String() string {
	format := "Matrix `%s`: format: `%s`, type: `%s`\n"
	return fmt.Sprintf(format, matrix.name, matrix.Format, matrix.Type)
}

// TODO maybe just pass the line and not the full buffer?
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

// ParseComment consumes all comment and/or emtpy lines from the reader. The
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
		case '%', '\n':
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

// ParseDimensions parses the dimensions and expected number of non-zero
// entries (nnz).
func (matrix *Matrix) ParseDimensions(buf *bufio.Reader) error {
	line, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	dims := strings.Split(strings.TrimSpace(line), " ")
	if len(dims) != 3 {
		return fmt.Errorf("Expect three values: (n, m, nnz), got: %v", dims)
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

	nnz, err := strconv.Atoi(dims[2])
	if err != nil {
		return err
	}
	matrix.nnz = nnz

	return nil
}

func (matrix *Matrix) Parse() error {
	path := matrix.Path()

	// preprocessing to get a *io.Reader
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	//var buf *bufio.Reader

	//	if filepath.Ext(path) == ".gz" {
	//		unzip, err := gzip.NewReader(file)
	//		if err != nil {
	//			defer file.Close()
	//		}
	//		buf = bufio.NewReader(unzip)
	//	} else {
	//		buf = bufio.NewReader(file)
	//	}
	//
	// consume all empty or comment lines
	// TODO: maybe store as general description
	/*
		line := string(b)
		for {
			if len(strings.TrimSpace(line)) != 0 {
				if line[0] != '%' {
					break
				}
			}
			line, err = buf.ReadString('\n')
			if err != nil {
				return err
			}
		}

		// matrix dimensions and number of entries
		log.Println(line)

	*/
	//b, err := buf.ReadBytes('\n')
	//if err != nil {
	//	return err
	//}

	// actual processing of the file
	//for scanner.Scan() {
	//	log.Println(scanner.Text())
	//}
	//if err := scanner.Err(); err != nil {
	//	return err
	//}

	return fmt.Errorf("%s", matrix)

	//gunzip, err := gzip.NewReader(resp.Body)
	//if err != nil {
	//	return err
	//}

	//// ftp://math.nist.gov/pub/MatrixMarket2/Harwell-Boeing/bcsstruc2/bcsstk14.mtx.gz

	//scanner := bufio.NewScanner(gunzip)
	//log.Printf("Contents of %+v\n", m)
	//for scanner.Scan() {
	//	log.Println(scanner.Text())
	//}
	//if err := scanner.Err(); err != nil {
	//	return err
	//}
}
