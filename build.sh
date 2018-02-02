#!/bin/sh
go get golang.org/x/image/draw
go get github.com/NYTimes/gziphandler

go build -o bin/application

zip -9 aws-image-maze.zip Procfile bin/application
