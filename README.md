# gomm
GoMM: a MatrixMarket parser in Go. 

Parses matrices from the [MatrixMarket](https://math.nist.gov/MatrixMarket/). 

> The Matrix Market provides convenient access to a repository of test data for use in comparative studies of algorithms for numerical linear algebra. Matrices as well as matrix generation software and services, from linear systems, least squares, and eigenvalue computations in a wide variety of scientific and engineering disciplines are provided. Tools for browsing through the collection or for searching for matrices with special properties are included. -- [MatrixMarket](https://math.nist.gov/MatrixMarket/). 

## Usage 
```
matrix := gomm.Matrix{
  collection: "Harwell-Boeing",
  set: "nnceng",
  name: "hor__131",
}

// downloads to hor__131.mtx.gz 
err := matrix.Download()

// open and decompress
f, _ := os.Open(file)
defer f.Close()

rd, _ := gzip.NewReader(f)

// parse matrix, returs mat.Matrix interface 
mm, _ := matrix.Parse(rd) 

// sparse matrices can be retrieved
csr, _ := mm.(*sparse.CSR) 
```
## References
- https://github.com/james-bowman/sparse
- https://godoc.org/gonum.org/v1/gonum/mat
- https://math.nist.gov/MatrixMarket/
