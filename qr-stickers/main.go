package main

import "bufio"
import "bytes"
import "image"
import _ "image/png"
import "log"
import "os"
import "github.com/signintech/gopdf"
import qrcode "github.com/skip2/go-qrcode"

const qrsize int = 128 + 32

func main() {
	var texts []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		texts = append(texts, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4 })
	pdf.AddPage()

	for i := 0; i < len(texts); i++ {
		if len(texts[i]) == 0 { continue }
		pdf.ImageFrom(text2image(texts[i]), coord(i,"x"), coord(i,"y"), nil)
	}

	if len(os.Args) == 2 {
		pdf.WritePdf(os.Args[1])
	} else {
		pdf.WritePdf("stickers.pdf")
	}
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
