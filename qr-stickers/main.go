package main

import "bytes"
import "image"
import "github.com/signintech/gopdf"
import _ "image/png"
import qrcode "github.com/skip2/go-qrcode"

const qrsize int = 128 + 32

func main() {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4 })
	pdf.AddPage()

	// First row at 15 is ok
//	pdf.ImageFrom(text2image("Test 1"),  58, 15, nil)
//	pdf.ImageFrom(text2image("Test 2"), 256, 15, nil)
//	pdf.ImageFrom(text2image("Test 3"), 454, 15, nil)
	pdf.ImageFrom(text2image("Test 1"), coord(0,"x"), coord(0,"y"), nil)
	pdf.ImageFrom(text2image("Test 2"), coord(1,"x"), coord(1,"y"), nil)
	pdf.ImageFrom(text2image("Test 3"), coord(2,"x"), coord(2,"y"), nil)

	// Second row at 115 is ok
	pdf.ImageFrom(text2image("Test 4"),  58, 115, nil)
	pdf.ImageFrom(text2image("Test 5"), 256, 115, nil)
	pdf.ImageFrom(text2image("Test 6"), 454, 115, nil)

	// Third row at 220 is ok
	pdf.ImageFrom(text2image("Test 7"),  58, 220, nil)
	pdf.ImageFrom(text2image("Test 8"), 256, 220, nil)
	pdf.ImageFrom(text2image("Test 9"), 454, 220, nil)

	pdf.ImageFrom(text2image("Test 19"), coord(18,"x"), coord(18,"y"), nil)
	pdf.ImageFrom(text2image("Test 20"), coord(19,"x"), coord(19,"y"), nil)
	pdf.ImageFrom(text2image("Test 21"), coord(20,"x"), coord(20,"y"), nil)

	pdf.ImageFrom(text2image("Test 22"), coord(21,"x"), coord(21,"y"), nil)
	pdf.ImageFrom(text2image("Test 23"), coord(22,"x"), coord(22,"y"), nil)
	pdf.ImageFrom(text2image("Test 24"), coord(23,"x"), coord(23,"y"), nil)

	pdf.WritePdf("test2.pdf")
}

func text2image (s string) (image.Image) {
	png, _ := qrcode.Encode(s, qrcode.Medium, qrsize)

	imgobj, _, _ := image.Decode(bytes.NewReader(png))
	return imgobj
}

func coord (pos int, xy string) (float64) {
	x := make(map[int][]float64)
	x[0] = []float64{ 58, 15}
	x[1] = []float64{256, 15}
	x[2] = []float64{454, 15}

	x[3] = []float64{ 58, 115}
	x[4] = []float64{256, 115}
	x[5] = []float64{454, 115}

	x[6] = []float64{ 58, 220}
	x[7] = []float64{256, 220}
	x[8] = []float64{454, 220}

	 x[9] = []float64{ 58, 325}
	x[10] = []float64{256, 325}
	x[11] = []float64{454, 325}

	x[12] = []float64{ 58, 430}
	x[13] = []float64{256, 430}
	x[14] = []float64{454, 430}

	x[15] = []float64{ 58, 535}
	x[16] = []float64{256, 535}
	x[17] = []float64{454, 535}

	x[18] = []float64{ 58, 640}
	x[19] = []float64{256, 640}
	x[20] = []float64{454, 640}

	x[21] = []float64{ 58, 745}
	x[22] = []float64{256, 745}
	x[23] = []float64{454, 745}

	if xy == "x" { return x[pos][0] }
	if xy == "y" { return x[pos][1] }
	
	return 0
}
