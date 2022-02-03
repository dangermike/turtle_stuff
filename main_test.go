package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/PerformLine/go-stockutil/colorutil"
	"github.com/holizz/terrapin"
	xdraw "golang.org/x/image/draw"
)

const (
	MaxSize  = int(3072)
	MaxSizeF = float64(MaxSize)
)

var imgPool = sync.Pool{
	New: func() interface{} {
		return image.NewRGBA(image.Rect(0, 0, 30000, 30000))
	},
}

func DoTest(tst *testing.T, move func(t *terrapin.Terrapin) bool) {
	img := imgPool.Get().(*image.RGBA)
	defer imgPool.Put(img)
	// clear the image
	draw.Draw(img, img.Rect, image.NewUniform(color.Transparent), image.Pt(0, 0), draw.Src)

	t := terrapin.NewTerrapin(
		img,
		terrapin.Position{
			X: float64(img.Bounds().Dx()) / 2,
			Y: float64(img.Bounds().Dy()) / 2,
		},
	)

	t.PenDown()

	minX, minY, maxX, maxY := t.Pos.X, t.Pos.Y, t.Pos.X, t.Pos.Y
	for move(t) {
		if minX > t.Pos.X {
			minX = t.Pos.X
		} else if maxX < t.Pos.X {
			maxX = t.Pos.X
		}
		if minY > t.Pos.Y {
			minY = t.Pos.Y
		} else if maxY < t.Pos.Y {
			maxY = t.Pos.Y
		}
	}

	sourceRect := image.Rect(
		int(math.Floor(minX-1)),
		int(math.Floor(minY-1)),
		int(math.Ceil(maxX+1)),
		int(math.Floor(maxY+1)),
	)

	_ = os.Mkdir("results", 0777) // ignore error if exists
	writer, err := os.OpenFile(path.Join("results", strings.ReplaceAll(tst.Name(), "/", "_")+".png"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		tst.Fatal(err)
	}
	if err := WriteImage(writer, img, sourceRect); err != nil {
		tst.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		tst.Fatal(err)
	}
}

func TestEuclidianSpiral(tst *testing.T) {
	i := 0
	angle := 0.0
	twopi := math.Pi * 2
	theta := 4.321 * math.Pi / 180

	DoTest(tst, func(t *terrapin.Terrapin) bool {
		if i >= 100000 {
			return false
		}
		r, g, b := colorutil.HslToRgb(float64(i), .5, .5)
		t.Color = color.RGBA{r, g, b, 0xff}
		t.Forward(10)
		angle += theta
		if angle > twopi {
			angle -= twopi
		}
		t.Right(angle)
		i++
		return true
	})
}

type instruction byte

const (
	iX instruction = iota
	iY
	iL
	iR
)

func TestSierpinski(tst *testing.T) {
	subs := map[instruction][]instruction{
		iX: {iY, iR, iX, iR, iY},
		iY: {iX, iL, iY, iL, iX},
	}

	iterate := func(in []instruction) []instruction {
		out := make([]instruction, 0, len(in)*9)
		for _, i := range in {
			if sval, ok := subs[i]; ok {
				out = append(out, sval...)
				continue
			}
			out = append(out, i)
		}
		return out
	}

	instructions := []instruction{iX}

	for i := 0; i < 9; i++ {
		tst.Run(strconv.Itoa(i), func(tst *testing.T) {
			instructions = iterate(instructions)
			lcl := instructions // so we don't lose our place
			step := float64(MaxSize >> i)
			DoTest(tst, func(t *terrapin.Terrapin) bool {
				t.Color = color.White
				if len(lcl) == 0 {
					return false
				}
				switch lcl[0] {
				case iL:
					t.Left(60 * math.Pi / 180)
				case iR:
					t.Right(60 * math.Pi / 180)
				default:
					t.Forward(step)
				}
				lcl = lcl[1:]
				return true
			})
		})
	}
}

func WriteImage(writer io.Writer, i *image.RGBA, sourceRect image.Rectangle) error {
	bigDim := sourceRect.Bounds().Dy()
	if bigDim < sourceRect.Bounds().Dx() {
		bigDim = sourceRect.Bounds().Dx()
	}

	if bigDim <= MaxSize {
		return png.Encode(writer, i.SubImage(sourceRect))
	}

	scaleFactor := MaxSizeF / float64(bigDim)
	outrect := image.Rect(
		0,
		0,
		int(float64(sourceRect.Dx())*scaleFactor),
		int(float64(sourceRect.Dy())*scaleFactor),
	)
	di := image.NewRGBA(outrect)
	xdraw.CatmullRom.Scale(
		di,
		outrect,
		i,
		sourceRect,
		draw.Over,
		nil,
	)
	return png.Encode(writer, di)
}
