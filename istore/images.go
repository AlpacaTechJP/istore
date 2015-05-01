package istore

/*

#cgo pkg-config: libavformat

#include "libavformat/avio.h"

*/
import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/3d0c/gmf"
	"github.com/disintegration/imaging"
	"github.com/golang/glog"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	AVSEEK_SIZE = C.AVSEEK_SIZE
)

func selfURL(p string) string {
	r := strings.NewReplacer("?", "%3F", "%", "%25")
	return "self://" + r.Replace(p)
}

func processImage(input io.Reader, mainProc func(image.Image) image.Image) ([]byte, error) {
	m, format, err := image.Decode(input)
	if err != nil {
		return nil, err
	}

	m = mainProc(m)

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

func adjustBrightness(input io.Reader, percentage float64) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.AdjustBrightness(m, percentage)
	})
}

func adjustContrast(input io.Reader, percentage float64) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.AdjustContrast(m, percentage)
	})
}

func adjustGamma(input io.Reader, gamma float64) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.AdjustGamma(m, gamma)
	})
}

func adjustSigmoid(input io.Reader, midpoint, factor float64) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.AdjustSigmoid(m, midpoint, factor)
	})
}

func blur(input io.Reader, sigma float64) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Blur(m, sigma)
	})
}

func crop(input io.Reader, x0, y0, x1, y1 int) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Crop(m, image.Rect(x0, y0, x1, y1))
	})
}

func fit(input io.Reader, width, height int) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Fit(m, width, height, imaging.Lanczos)
	})
}

func flipH(input io.Reader) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.FlipH(m)
	})
}

func flipV(input io.Reader) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.FlipV(m)
	})
}

func grayscale(input io.Reader) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Grayscale(m)
	})
}

func invert(input io.Reader) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Invert(m)
	})
}

func sharpen(input io.Reader, sigma float64) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Sharpen(m, sigma)
	})
}

func transpose(input io.Reader) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Transpose(m)
	})
}

func transverse(input io.Reader) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Transverse(m)
	})
}

func resize(input io.Reader, w, h int) ([]byte, error) {
	return processImage(input, func(m image.Image) image.Image {
		return imaging.Resize(m, w, h, imaging.Lanczos)
	})
}

type AVWrapper struct {
	inputCtx    *gmf.AVIOContext
	codec       *gmf.Codec
	videoStream *gmf.Stream
	codecCtx    *gmf.CodecCtx
}

type ExpandArgs struct {
	Video string `json:"video"`
}

func (s *Server) Expand(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Path
	dir = dir[0 : len(dir)-len("_expand")]
	if !strings.HasSuffix(dir, "/") {
		http.Error(w, "expand should finish with '/'", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	args := ExpandArgs{}
	if err := decoder.Decode(&args); err != nil {
		http.Error(w, "unrecognized args", http.StatusBadRequest)
		return
	}
	if args.Video == "" {
		http.Error(w, "\"video\" field is mandatory", http.StatusBadRequest)
		return
	}

	videopath := args.Video
	vUrl := extractTargetURL(videopath)
	if vUrl == "" {
		msg := fmt.Sprintf("target not found in path %s", videopath)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	resp, err := s.Client.Get(vUrl)
	if err != nil {
		glog.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if err := expand(s, resp.Body, dir, videopath); err != nil {
		glog.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
}

func makeInputHandlers(input io.Reader) *gmf.AVIOHandlers {
	reader, ok := input.(io.ReadSeeker)
	if !ok {
		// TODO: spill to disk if necessary
		glog.Info("Reader not seekable")
		buf := new(bytes.Buffer)
		io.Copy(buf, input)
		reader = bytes.NewReader(buf.Bytes())
	}

	return &gmf.AVIOHandlers{
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
}

func expand(s *Server, input io.Reader, dir, objkey string) error {
	handlers := makeInputHandlers(input)

	ctx := gmf.NewCtx()
	ioctx, err := gmf.NewAVIOContext(ctx, handlers)
	ctx.SetPb(ioctx)
	defer ctx.CloseInputAndRelease()

	if err = ctx.OpenInput("dummy"); err != nil {
		glog.Error(err)
		return err
	}

	batch := new(leveldb.Batch)
	seconds := float64(ctx.Duration())
	glog.Info(seconds, " seconds")
	for i := 0; i < int(seconds/1000000)+1; i++ {
		// TODO: create relpath.  filepath.Rel() removes duplicate slashes, bad for us.
		//selfpath, err := filepath.Rel(dir, objkey)
		//if err != nil {
		//	glog.Error(err)
		//	break
		//}

		// Escape only the path part to distinguish it from query string.
		selfpath := selfURL(objkey)
		// query string can be raw.
		selfpath += fmt.Sprintf("?apply=frame&fn=%d", i)

		key := dir + selfpath
		meta := map[string]interface{}{}
		d := time.Duration(i) * time.Second
		meta["timestamp"] = fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
		meta["video"] = objkey
		value, _ := json.Marshal(&meta)
		_, _, err := s.PutObject([]byte(key), string(value), batch, true)
		if err != nil {
			return err
		}
	}

	if err := s.Db.Write(batch, nil); err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

func frame(input io.Reader, fn int) ([]byte, error) {
	handlers := makeInputHandlers(input)

	ctx := gmf.NewCtx()
	ioctx, err := gmf.NewAVIOContext(ctx, handlers)
	ctx.SetPb(ioctx)
	defer ctx.CloseInputAndRelease()

	if err = ctx.OpenInput("dummy"); err != nil {
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

	// Just to surprress "deprected format" warning...
	cc.SetPixFmt(gmf.AV_PIX_FMT_YUV420P)

	swsCtx := gmf.NewSwsCtx(srcVideoStream.CodecCtx(), cc, gmf.SWS_BICUBIC)
	defer gmf.Release(swsCtx)

	dstFrame := gmf.NewFrame().
		SetWidth(srcVideoStream.CodecCtx().Width()).
		SetHeight(srcVideoStream.CodecCtx().Height()).
		SetFormat(gmf.AV_PIX_FMT_YUV420P)
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

		// TODO: Frames is not concurrency-safe.
		for frame := range packet.Frames(ist.CodecCtx()) {
			swsCtx.Scale(frame, dstFrame)

			if p, ready, _ := dstFrame.EncodeNewPacket(cc); ready {
				defer gmf.Release(p)
				return p.Data(), nil
			}
		}

		// TODO: release in early return
		// We need to stop the channel returned by GetNewPackets
		// or suck it up to release all packet objects.
		// Probably we need a synchronized version of GetNewPackets.
		// Signaling the goroutine could be another way.
		gmf.Release(packet)
	}

	return nil, fmt.Errorf("error")
}

// --- snippet
// curl localhost:8592/path/mp4/slice/ | jq -r '. | sort_by(.metadata.timestamp) | .[] | "\(.metadata.timestamp)<img src=\"http://localhost:8592\(._filepath)\"><br/>"' | sed -e 's/&/%26/' | sed -e 's/?/%3F/' > /tmp/foo.html
