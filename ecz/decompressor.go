// archive/zip/register.go

package ecz

import (
	"compress/flate"
	"errors"
	"io"
	"io/ioutil"
	"sync"
)

type Decompressor func(reader io.Reader) io.ReadCloser

var (
	decompressors map[int]Decompressor
)

const (
	Store   = 0
	Deflate = 8
)

func init() {
	decompressors = map[int]Decompressor{}
	decompressors[Store] = ioutil.NopCloser
	decompressors[Deflate] = newFlateReader
}

var flateReaderPool sync.Pool

func newFlateReader(r io.Reader) io.ReadCloser {
	fr, ok := flateReaderPool.Get().(io.ReadCloser)
	if ok {
		fr.(flate.Resetter).Reset(r, nil)
	} else {
		fr = flate.NewReader(r)
	}
	return &pooledFlateReader{fr: fr}
}

type pooledFlateReader struct {
	mu sync.Mutex // guards Close and Read
	fr io.ReadCloser
}

func (r *pooledFlateReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fr == nil {
		return 0, errors.New("Read after Close")
	}
	return r.fr.Read(p)
}

func (r *pooledFlateReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var err error
	if r.fr != nil {
		err = r.fr.Close()
		flateReaderPool.Put(r.fr)
		r.fr = nil
	}
	return err
}
