# gomm
GoMM: a MatrixMarket parser in Go. 

Parses matrices from the [MatrixMarket](https://math.nist.gov/MatrixMarket/). 

```
matrix := gomm.Matrix{
  collection: "Harwell-Boeing",
  set: "smtape",
  name: "ash608",
}
matrix.Download()
```
