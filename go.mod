module github.com/maxvdkolk/gomm

go 1.14

require (
	github.com/gonum/floats v0.0.0-20181209220543-c233463c7e82 // indirect
	github.com/gonum/internal v0.0.0-20181124074243-f884aa714029 // indirect
	github.com/james-bowman/sparse v0.0.0-20200514124614-ae250424e52d
	github.com/jlaffaye/ftp v0.0.0-20200602180915-5563613968bf
	gonum.org/v1/gonum v0.7.0
)

// replace with bugfix for COO with leading zeros.
replace github.com/james-bowman/sparse => github.com/maxvdkolk/sparse v0.0.0-20200610142949-d6ed124132d5
