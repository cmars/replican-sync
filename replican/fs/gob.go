package fs

import (
	"bytes"
	"fmt"
	"gob"
	"os"
	"reflect"
)

const gobNodeVersion int = 1

func checkVersion(decoder *gob.Decoder) os.Error {
	var version int
	decoder.DecodeValue(reflect.ValueOf(&version))
	if version != gobNodeVersion {
		return os.NewError(fmt.Sprintf("Version %d of node gobber cannot decode version %d",
			gobNodeVersion, version))
	}
	return nil
}

func (block *Block) GobDecode(buf []byte) (err os.Error) {
	buffer := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(buffer)

	err = checkVersion(decoder)
	if err != nil {
		return err
	}

	err = decoder.DecodeValue(reflect.ValueOf(&block.position))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&block.weak))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&block.strong))
	if err != nil {
		return err
	}

	return nil
}

func (file *File) GobDecode(buf []byte) (err os.Error) {
	buffer := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(buffer)

	err = checkVersion(decoder)
	if err != nil {
		return err
	}

	err = decoder.DecodeValue(reflect.ValueOf(&file.name))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&file.mode))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&file.strong))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&file.Size))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&file.Blocks))
	if err != nil {
		return err
	}

	for _, block := range file.Blocks {
		block.parent = file
	}

	return nil
}

func (dir *Dir) GobDecode(buf []byte) (err os.Error) {
	buffer := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(buffer)

	err = checkVersion(decoder)
	if err != nil {
		return err
	}

	err = decoder.DecodeValue(reflect.ValueOf(&dir.name))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&dir.mode))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&dir.strong))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&dir.SubDirs))
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(reflect.ValueOf(&dir.Files))
	if err != nil {
		return err
	}

	for _, file := range dir.Files {
		file.parent = dir
	}
	for _, subdir := range dir.SubDirs {
		subdir.parent = dir
	}

	return nil
}

func (block *Block) GobEncode() ([]byte, os.Error) {
	buffer := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buffer)

	var err os.Error
	err = encoder.EncodeValue(reflect.ValueOf(gobNodeVersion))
	if err != nil {
		return nil, err
	}

	err = encoder.EncodeValue(reflect.ValueOf(&block.position))
	if err != nil {
		return nil, err
	}
	encoder.EncodeValue(reflect.ValueOf(&block.weak))
	if err != nil {
		return nil, err
	}
	encoder.EncodeValue(reflect.ValueOf(&block.strong))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (file *File) GobEncode() ([]byte, os.Error) {
	buffer := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buffer)

	var err os.Error
	err = encoder.EncodeValue(reflect.ValueOf(gobNodeVersion))
	if err != nil {
		return nil, err
	}

	err = encoder.EncodeValue(reflect.ValueOf(&file.name))
	if err != nil {
		return nil, err
	}
	err = encoder.EncodeValue(reflect.ValueOf(&file.mode))
	if err != nil {
		return nil, err
	}
	err = encoder.EncodeValue(reflect.ValueOf(&file.strong))
	if err != nil {
		return nil, err
	}
	err = encoder.EncodeValue(reflect.ValueOf(&file.Size))
	if err != nil {
		return nil, err
	}
	err = encoder.EncodeValue(reflect.ValueOf(&file.Blocks))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (dir *Dir) GobEncode() ([]byte, os.Error) {
	buffer := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buffer)

	var err os.Error
	err = encoder.EncodeValue(reflect.ValueOf(gobNodeVersion))
	if err != nil {
		return nil, err
	}

	err = encoder.EncodeValue(reflect.ValueOf(&dir.name))
	if err != nil {
		return nil, err
	}
	err = encoder.EncodeValue(reflect.ValueOf(&dir.mode))
	if err != nil {
		return nil, err
	}
	err = encoder.EncodeValue(reflect.ValueOf(&dir.strong))
	if err != nil {
		return nil, err
	}
	err = encoder.EncodeValue(reflect.ValueOf(&dir.SubDirs))
	if err != nil {
		return nil, err
	}
	err = encoder.EncodeValue(reflect.ValueOf(&dir.Files))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func DecodeDir(data []byte) (dir *Dir, err os.Error) {
	decoder := gob.NewDecoder(bytes.NewBuffer(data))
	dir = &Dir{}
	err = decoder.Decode(dir)
	return dir, err
}
