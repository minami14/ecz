# ecz

ecz extract corrupted zip file. Normally read the central directory entry to extract the zip, but that method cannot
extract a zip file with a corrupted central directory. ecz unzips a zip file with a corrupted central directory entry by
searching the entire file and reading the local file header. Cannot be used if the compressed size is unknown.

## Supported compression method

* stored
* deflate

## Install

```bash
go get github.com/minami14/ecz
```

## Usage

```bash
ecz corrupted.zip
```