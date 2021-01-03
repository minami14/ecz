package main

import (
	"errors"
	"io"
	"log"
	"os"

	"github.com/minami14/ecz/ecz"
)

func main() {
	for _, name := range os.Args[1:] {
		if err := extract(name); err != nil {
			log.Println(err)
		}
	}
}

func extract(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		return err
	}

	e, err := ecz.New(f, s.Size())
	if err != nil {
		return err
	}

	for {
		file, err := e.NextFile()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err
		}

		if file.IsDir() {
			if err := os.Mkdir(file.Header.FileName, 0777); err != nil {
				log.Println(err)
			}
		}

		if file.IsFile() {
			if err := write(file); err != nil {
				log.Println(err)
			}
		}
	}
}

func write(file *ecz.File) error {
	f, err := os.Create(file.Header.FileName)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := file.Write(f); err != nil {
		return err
	}

	return nil
}
