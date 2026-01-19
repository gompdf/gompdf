module github.com/gompdf/gompdf

go 1.24.0

toolchain go1.24.6

require golang.org/x/net v0.49.0

require (
	codeberg.org/go-pdf/fpdf v0.11.1
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef
	golang.org/x/image v0.15.0
)

require golang.org/x/text v0.33.0 // indirect

replace github.com/gompdf/gompdf => /home/henrrius/code/gompdf
