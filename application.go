// This is Free Software covered by the terms of the Apache 2 license.
// See LICENSE file for details.
package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/NYTimes/gziphandler"
)

const (
	maxSize = 2 * 1024 * 1024 // 2MB
	minDim  = 11
	maxDim  = 501
)

const htmlHeader = `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01//EN">
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<title>Maze</title>
<script type="text/javascript">
function toggleSize() {
    var disabled = !document.getElementById("custom-size").checked;
    document.getElementById("width").disabled = disabled;
    document.getElementById("height").disabled = disabled;
}
</script>
</head>
<body>
  <h1>Image based maze generator</h1>
`

const htmlForm = `
<div>
<form method="POST" action="/" enctype="multipart/form-data">
<fieldset>
<legend><a name="params">Parameters</a></legend>
<table summary="Parameters">
  <tr>
    <td style="text-align:right"><label for="upimage">Image<sup>*</sup>:</label></td>
    <td><input type="file" name="upimage" id="upimage"></td>
  </tr>
  <tr>
    <td style="text-align:right"><label for="custom-size">Custom Size:</label></td>
    <td><input type="checkbox" name="custom-size" id="custom-size" onchange="toggleSize();" value="custom-size" %s></td>
  </tr>
  <tr>
    <td style="text-align:right"><label for="width">Width<sup>**</sup>:</label></td>
    <td><input type="text" name="width" id="width" value="%d" %s></td>
  </tr>
  <tr>
    <td style="text-align:right"><label for="height">Height<sup>**</sup>:</label></td>
    <td><input type="text" name="height" id="height" value="%d" %s></td>
  </tr>
</table>
<br>
<table>
  <tr>
    <td style="text-align:right;vertical-align:top"><sup>*</sup>:</td>
    <td><small>Max. size 2MB.<br>Accepts PNG, JPEGs and GIFs.</small></td>
  </tr>
  <tr>
    <td style="text-align:right;vertical-align:top"><sup>**</sup>:</td>
    <td><small>Max. 501 pixels.
      <br>Leaving a field empty scales maze proportional.
      <br>Will be adjusted to odd values to have a nicely closed border.</small>
    </td>
  </tr>
</table>
<input type="submit" value="Generate maze!">
</fieldset>
</form>
</div>
`

const htmlFooter = `
<div>&copy; 2013, 2018 by Sascha L. Teichmann
(<a href="https://github.com/s-l-teichmann/aws-image-maze">Sources</a>)</div>
</body>
</html>`

func getInt(req *http.Request, name string) (int, bool) {
	v, err := strconv.Atoi(req.FormValue(name))
	return v, err == nil
}

func getBool(req *http.Request, name string) bool {
	return req.FormValue(name) == name
}

func getImage(req *http.Request, name string) (img image.Image, err error) {
	var f multipart.File
	if f, _, err = req.FormFile(name); err != nil {
		return nil, err
	}
	defer f.Close() // XXX: Really needed?
	img, _, err = image.Decode(io.LimitReader(f, maxSize))
	return
}

func getImageOrDefault(req *http.Request, name string) (image.Image, error) {
	img, err := getImage(req, name)
	if err == nil {
		return img, nil
	}

	img, _, err = image.Decode(bytes.NewReader(defaultImage[:]))
	return img, err
}

func clamp(a, b int) func(int) int {
	return func(x int) int {
		switch {
		case x < a:
			return a
		case x > b:
			return b
		default:
			return x
		}
	}
}

func index(rw http.ResponseWriter, req *http.Request) {

	img, err := getImageOrDefault(req, "upimage")
	if err != nil {
		http.Error(rw,
			fmt.Sprintf("image broken: %v.", err),
			http.StatusInternalServerError)
		return
	}

	dim := img.Bounds()

	var width, height int

	cl := clamp(minDim, maxDim)

	customSize := getBool(req, "custom-size")

	if customSize {
		w, wOk := getInt(req, "width")
		h, hOk := getInt(req, "height")

		switch {
		case wOk && hOk:
			width = cl(w)
			height = cl(h)
		case wOk:
			ratio := float64(dim.Dy()) / float64(dim.Dx())
			width = cl(w)
			height = cl(int(float64(width) * ratio))
		case hOk:
			ratio := float64(dim.Dx()) / float64(dim.Dy())
			height = cl(h)
			width = cl(int(float64(height) * ratio))
		}
	} else {
		ratio := float64(dim.Dy()) / float64(dim.Dx())
		width = cl(dim.Dx())
		height = cl(int(float64(width) * ratio))
	}

	if width&1 == 0 {
		width++
	}
	if height&1 == 0 {
		height++
	}

	rw.Header().Set("Content-type", "text/html; charset=UTF-8")

	fmt.Fprint(rw, htmlHeader)

	if err != nil {
		fmt.Fprintf(rw, "<strong>Error: %v.</strong>\n", err)
	} else {
		m := newMaze(img, width, height)
		m.generate()
		m.writeBase64Image(rw)
	}

	var checked, disabled string

	if customSize {
		checked = `checked="checked"`
	} else {
		disabled = `disabled="disabled"`
	}

	fmt.Fprintf(rw, htmlForm,
		checked,
		width, disabled,
		height, disabled)

	fmt.Fprintln(rw, htmlFooter)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	gz := gziphandler.GzipHandler(http.HandlerFunc(index))

	mux := http.NewServeMux()
	mux.Handle("/", gz)

	log.Fatalln(http.ListenAndServe(":"+port, mux))
}
