package istore

/*

#cgo pkg-config: libavformat

#include "libavformat/avio.h"

*/
import "C"

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/3d0c/gmf"
	"github.com/disintegration/imaging"
	"github.com/golang/glog"
)

var (
	AVSEEK_SIZE = C.AVSEEK_SIZE
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

func frame(input io.Reader, fn int) ([]byte, error) {
	ctx := gmf.NewCtx()

	reader, ok := input.(io.ReadSeeker)
	if !ok {
		// TODO: spill to disk if necessary
		glog.Info("Reader not seekable")
		buf := new(bytes.Buffer)
		io.Copy(buf, input)
		reader = bytes.NewReader(buf.Bytes())
	}

	handlers := &gmf.AVIOHandlers{
		ReadPacket: func() ([]byte, int) {
			b := make([]byte, 512)
			n, err := reader.Read(b)
			if err != nil {
				glog.Error(err)
			}
			return b, n
		},
		WritePacket: func(b []byte) {
			glog.Error("unexpected Write call")
		},
		Seek: func(offset int64, whence int) int64 {
			n, err := reader.Seek(offset, whence)
			if whence != AVSEEK_SIZE && err != nil {
				glog.Error(err, fmt.Sprintf(" (offset = %d, whence = %d)", offset, whence))
			}
			return n
		},
	}
	ioctx, err := gmf.NewAVIOContext(ctx, handlers)
	ctx.SetPb(ioctx)
	defer ctx.CloseInputAndRelease()

	if err = ctx.OpenInput("dummy.mp4"); err != nil {
		glog.Error(err)
		return nil, err
	}

	srcVideoStream, err := ctx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	if err = ctx.SeekFrameAt(fn, srcVideoStream.Index()); err != nil {
		glog.Error(err)
		return nil, err
	}

	codec, err := gmf.FindEncoder(gmf.AV_CODEC_ID_MJPEG)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	cc := gmf.NewCodecCtx(codec)
	defer gmf.Release(cc)

	cc.SetPixFmt(gmf.AV_PIX_FMT_YUVJ420P).SetWidth(srcVideoStream.CodecCtx().Width()).SetHeight(srcVideoStream.CodecCtx().Height()).SetTimeBase(gmf.AVR{Num: 1, Den: 50})

	if codec.IsExperimental() {
		cc.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err = cc.Open(nil); err != nil {
		glog.Error(err)
		return nil, err
	}

	swsCtx := gmf.NewSwsCtx(srcVideoStream.CodecCtx(), cc, gmf.SWS_BICUBIC)
	defer gmf.Release(swsCtx)

	dstFrame := gmf.NewFrame().
		SetWidth(srcVideoStream.CodecCtx().Width()).
		SetHeight(srcVideoStream.CodecCtx().Height()).
		SetFormat(gmf.AV_PIX_FMT_YUVJ420P)
	defer gmf.Release(dstFrame)

	if err := dstFrame.ImgAlloc(); err != nil {
		glog.Error(err)
		return nil, err
	}

	for packet := range ctx.GetNewPackets() {
		if packet.StreamIndex() != srcVideoStream.Index() {
			continue
		}
		ist, err := ctx.GetStream(packet.StreamIndex())
		if err != nil {
			return nil, err
		}

		for frame := range packet.Frames(ist.CodecCtx()) {
			swsCtx.Scale(frame, dstFrame)

			if p, ready, _ := dstFrame.EncodeNewPacket(cc); ready {
				defer gmf.Release(p)
				return p.Data(), nil
			}
		}
		// TODO: release in early return
		gmf.Release(packet)
	}

	gmf.Release(dstFrame)

	return nil, fmt.Errorf("error")
}
