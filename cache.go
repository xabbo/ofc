package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
)

func loadOrFetch(name, url string) (b []byte, err error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return
	}

	if fi.Size() == 0 {
		var res *http.Response
		res, err = http.Get(url)
		if err != nil {
			return
		}
		if res.StatusCode != 200 {
			err = errors.New(res.Status)
			return
		}
		buf := bytes.NewBuffer(make([]byte, 0, res.ContentLength))
		_, err = io.Copy(io.MultiWriter(buf, f), res.Body)
		if err == nil {
			b = buf.Bytes()
		}
	} else {
		buf := bytes.NewBuffer(make([]byte, 0, fi.Size()))
		_, err = io.Copy(buf, f)
		b = buf.Bytes()
	}

	return
}
