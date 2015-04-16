package istore

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
)

func resize(input io.Reader, w, h int) ([]byte, error) {
	m, format, err := image.Decode(input)
	if err != nil {
		return nil, err
	}

	m = imaging.Resize(m, w, h, imaging.Lanczos)

	buf := new(bytes.Buffer)
	switch format {
	case "gif":
		gif.Encode(buf, m, nil)
	case "jpeg":
		quality := 95
		jpeg.Encode(buf, m, &jpeg.Options{Quality: quality})
	case "png":
		png.Encode(buf, m)
	default:
		return nil, fmt.Errorf("unknown format %s", format)
	}

	return buf.Bytes(), nil
}
