// This is Free Software covered by the terms of the Apache 2 license.
// See LICENSE file for details.
package main

import (
	"container/heap"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"

	"golang.org/x/image/draw"
)

type maze struct {
	image.Gray
	maze []byte
}

func newMaze(src image.Image, width, height int) *maze {

	r := image.Rect(0, 0, width, height)
	w, h := r.Dx(), r.Dy()
	pix := make([]uint8, w*h)

	m := &maze{
		Gray: image.Gray{
			Pix:    pix,
			Stride: w,
			Rect:   r,
		},
	}

	draw.CatmullRom.Scale(&m.Gray, r, src, src.Bounds(), draw.Src, nil)

	return m
}

func (m *maze) index(x, y int) int {
	return y*m.Stride + x
}

func (m *maze) position(p int) (int, int) {
	return p % m.Stride, p / m.Stride
}

func (m *maze) weight(n, p int) int {
	return 4*int(m.Pix[p]) + int(m.Pix[n])
}

type edge struct {
	n, p, a, v int
}

type edgeHeap []edge

func (e edgeHeap) Len() int {
	return len(e)
}

func (e edgeHeap) Less(i, j int) bool {
	vi, vj := e[i].v, e[j].v
	if vi > vj {
		return true
	}
	if vj == vi {
		return e[i].a > e[j].a
	}
	return false
}

func (e edgeHeap) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e *edgeHeap) Push(x interface{}) {
	*e = append(*e, x.(edge))
}

func (e *edgeHeap) Pop() interface{} {
	old := *e
	n := len(old)
	item := old[n-1]
	*e = old[:n-1]
	return item
}

func (m *maze) generate() {

	width, height := m.Rect.Dx(), m.Rect.Dy()

	size := width * height

	m.maze = make([]byte, size, size)

	// Fill with walls.
	for i := range m.maze {
		m.maze[i] = 1
	}

	startx := width / 2
	starty := height / 2

	if startx%2 == 0 {
		startx--
	}

	if starty%2 == 0 {
		starty--
	}

	p := m.index(startx, starty)

	// Clear start position.
	m.maze[p] = 0

	h := &edgeHeap{}

	heap.Init(h)

	age := 0

loop:

	x, y := m.position(p)

	if yp := y - 2; yp > 0 {
		if next := m.index(x, yp); m.maze[next] == 1 {
			heap.Push(h, edge{next, next + width, age, m.weight(next, next+width)})
		}
	}

	if yn := y + 2; yn < height {
		if next := m.index(x, yn); m.maze[next] == 1 {
			heap.Push(h, edge{next, next - width, age, m.weight(next, next-width)})
		}
	}

	if xp := x - 2; xp > 0 {
		if next := m.index(xp, y); m.maze[next] == 1 {
			heap.Push(h, edge{next, next + 1, age, m.weight(next, next+1)})
		}
	}

	if xn := x + 2; xn < width {
		if next := m.index(xn, y); m.maze[next] == 1 {
			heap.Push(h, edge{next, next - 1, age, m.weight(next, next-1)})
		}
	}

	for h.Len() > 0 {
		item := heap.Pop(h).(edge)

		if m.maze[item.n] == 0 {
			continue
		}

		m.maze[item.n], m.maze[item.p] = 0, 0
		p = item.n
		age = item.a + 1
		goto loop
	}
}

var palette = []color.Color{
	color.NRGBA{0xff, 0xff, 0xff, 0xff},
	color.NRGBA{0, 0, 0, 0xff}}

func (m *maze) writeBase64(w io.Writer) error {
	if m.maze == nil {
		return nil
	}

	p := image.NewPaletted(m.Rect, palette)

	width, height := m.Rect.Dx(), m.Rect.Dy()

	pos := 0
	for y := 0; y < height; y++ {
		for x, v := range m.maze[pos : pos+width] {
			p.SetColorIndex(x, y, uint8(v))
		}
		pos += width
	}

	b64enc := base64.NewEncoder(base64.StdEncoding, w)
	png.Encode(b64enc, p)
	return b64enc.Close()
}

func (m *maze) writeBase64Image(w io.Writer) error {
	fmt.Fprintf(w,
		`<div><img alt="maze" width="%d" height="%d" src="data:image/png;base64,`,
		m.Rect.Dx(), m.Rect.Dy())
	m.writeBase64(w)
	_, err := fmt.Fprintln(w, `"></div>`)
	return err
}
