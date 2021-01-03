package ecz

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

const (
	sigLocal = "PK\003\004"
)

var (
	ErrInvalidSig                    = errors.New("invalid local signature")
	ErrNotImplementCompressionMethod = errors.New("not implement compression method")
)

type Ecz struct {
	reader io.ReaderAt
	size   int64
	offset int64
}

type LocalHeader struct {
	Version           int
	Flag              int
	CompressionMethod int
	LastModTime       int
	LastModDate       int
	Crc32             int
	CompressedSize    int
	UncompressedSize  int
	FileNameLength    int
	ExtraFieldLength  int
	FileName          string
	ExtraField        []byte
}

type compressedDataReader struct {
	reader io.ReaderAt
	size   int
	offset int64
}

func (c *compressedDataReader) Read(buf []byte) (int, error) {
	if len(buf) > c.size {
		buf = buf[:c.size]
	}
	return c.reader.ReadAt(buf, c.offset)
}

type File struct {
	Header           LocalHeader
	decompressor     Decompressor
	CompressedReader *compressedDataReader
}

func (f *File) IsDir() bool {
	return f.Header.CompressedSize == 0
}

func (f *File) IsFile() bool {
	return f.Header.CompressedSize > 0
}

func (f *File) Write(w io.Writer) (int64, error) {
	decompress := f.decompressor(f.CompressedReader)
	return io.Copy(w, decompress)
}

func New(r io.ReaderAt, size int64) (*Ecz, error) {
	return &Ecz{
		reader: r,
		size:   size,
		offset: 0,
	}, nil
}

func (e *Ecz) NextFile() (*File, error) {
	if err := e.next(); err != nil {
		return nil, err
	}

	buf := make([]byte, (1<<16)-1)
	if _, err := e.reader.ReadAt(buf[:4], e.offset); err != nil {
		return nil, err
	}
	e.offset += 4
	if string(buf[:4]) != sigLocal {
		return nil, ErrInvalidSig
	}

	if _, err := e.reader.ReadAt(buf[:2], e.offset); err != nil {
		return nil, err
	}
	e.offset += 2
	version := int(binary.LittleEndian.Uint16(buf[:2]))

	if _, err := e.reader.ReadAt(buf[:2], e.offset); err != nil {
		return nil, err
	}
	e.offset += 2
	flag := int(binary.LittleEndian.Uint16(buf[:2]))

	if _, err := e.reader.ReadAt(buf[:2], e.offset); err != nil {
		return nil, err
	}
	e.offset += 2
	compressionMethod := int(binary.LittleEndian.Uint16(buf[:2]))

	if _, err := e.reader.ReadAt(buf[:2], e.offset); err != nil {
		return nil, err
	}
	e.offset += 2
	lastModTime := int(binary.LittleEndian.Uint16(buf[:2]))

	if _, err := e.reader.ReadAt(buf[:2], e.offset); err != nil {
		return nil, err
	}
	e.offset += 2
	lastModDate := int(binary.LittleEndian.Uint16(buf[:2]))

	if _, err := e.reader.ReadAt(buf[:4], e.offset); err != nil {
		return nil, err
	}
	e.offset += 4
	crc32 := int(binary.LittleEndian.Uint16(buf[:4]))

	if _, err := e.reader.ReadAt(buf[:4], e.offset); err != nil {
		return nil, err
	}
	e.offset += 4
	compressedSize := int(binary.LittleEndian.Uint16(buf[:4]))

	if _, err := e.reader.ReadAt(buf[:4], e.offset); err != nil {
		return nil, err
	}
	e.offset += 4
	uncompressedSize := int(binary.LittleEndian.Uint16(buf[:4]))

	if _, err := e.reader.ReadAt(buf[:2], e.offset); err != nil {
		return nil, err
	}
	e.offset += 2
	filenameLength := int(binary.LittleEndian.Uint16(buf[:2]))

	if _, err := e.reader.ReadAt(buf[:2], e.offset); err != nil {
		return nil, err
	}
	e.offset += 2
	extraFieldLength := int(binary.LittleEndian.Uint16(buf[:2]))

	if _, err := e.reader.ReadAt(buf[:filenameLength], e.offset); err != nil {
		return nil, err
	}
	e.offset += int64(filenameLength)
	name := string(buf[:filenameLength])

	extraField := make([]byte, extraFieldLength)
	if _, err := e.reader.ReadAt(extraField, e.offset); err != nil {
		return nil, err
	}
	e.offset += int64(extraFieldLength)

	compressedReader := &compressedDataReader{
		reader: e.reader,
		size:   compressedSize,
		offset: e.offset,
	}
	e.offset += int64(compressedSize)

	header := LocalHeader{
		Version:           version,
		Flag:              flag,
		CompressionMethod: compressionMethod,
		LastModTime:       lastModTime,
		LastModDate:       lastModDate,
		Crc32:             crc32,
		CompressedSize:    compressedSize,
		UncompressedSize:  uncompressedSize,
		FileNameLength:    filenameLength,
		ExtraFieldLength:  extraFieldLength,
		FileName:          name,
		ExtraField:        extraField,
	}

	decompressor, ok := decompressors[compressionMethod]
	if !ok {
		return nil, ErrNotImplementCompressionMethod
	}

	f := &File{
		Header:           header,
		CompressedReader: compressedReader,
		decompressor:     decompressor,
	}

	return f, nil
}

func (e *Ecz) next() error {
	buf := make([]byte, 1024)
	offset := e.offset
	for offset < e.size {
		n, err := e.reader.ReadAt(buf, offset)
		if err != nil {
			return err
		}

		i := bytes.Index(buf, []byte(sigLocal))
		if i == -1 {
			offset += int64(n)
			continue
		}

		e.offset = offset + int64(i)
		return nil
	}

	return io.EOF
}
