![Go](https://github.com/MaxvdKolk/gomm/workflows/Go/badge.svg)
# gomm
`gomm`: a MatrixMarket parser in Go. 

Parses matrices from the [MatrixMarket](https://math.nist.gov/MatrixMarket/). 

> The Matrix Market provides convenient access to a repository of test data for use in comparative studies of algorithms for numerical linear algebra. Matrices as well as matrix generation software and services, from linear systems, least squares, and eigenvalue computations in a wide variety of scientific and engineering disciplines are provided. Tools for browsing through the collection or for searching for matrices with special properties are included. -- [MatrixMarket](https://math.nist.gov/MatrixMarket/). 

## Usage 
Matrices can be downloaded directly as a `mat.Matrix` interface 
using the `GetMatrix` routine. For example, to download a sparse
matrix and obtain its `*sparse.CSR` representation:
``` 
mm, err := GetMatrix("Harwell-Boeing", "bcsstruct1", "bcsstk01")
csr, ok := mm.(*sparse.CSR)

n, m := csr.Dims()
nnz := csr.NNZ()
fmt.Printf("type: %T, (rows,cols): (%d,%d), nzz: %d", csr, n, m, nnz)

// Output: type: *sparse.CSR, (rows,cols): (48,48), nzz: 400
```
Alternatively, the matrices can be dowloaded to disk first in 
as `.mtx.gz` and parsed from there: 
```
matrix := gomm.Matrix{
  collection: "Harwell-Boeing",
  set: "bcsstruct1",
  name: "bcsstk01",
}

// downloads to bcsstk01.mtx.gz 
err := matrix.Download()

// open and decompress
f, _ := os.Open(file)
defer f.Close()

rd, _ := gzip.NewReader(f)

// parse matrix, returs mat.Matrix interface 
mm, _ := matrix.Parse(rd) 

// sparse matrices can be retrieved
csr, _ := mm.(*sparse.CSR) 

n, m := csr.Dims()
nnz := csr.NNZ()
fmt.Printf("type: %T, (rows,cols): (%d,%d), nzz: %d", csr, n, m, nnz)

// Output: type: *sparse.CSR, (rows,cols): (48,48), nzz: 400
```

## Install
```
go get github.com/maxvdkolk/gomm
```

## References
- The `MatrixMarket` format: https://math.nist.gov/MatrixMarket/
- Dense matrices and the `mat.Matrix` interface: https://godoc.org/gonum.org/v1/gonum/mat
- Sparse matrices `*sparse.CSR`: https://github.com/james-bowman/sparse
