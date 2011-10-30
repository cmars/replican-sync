
package fs

import (
	"bytes"
	"gob"
	"os"
	"reflect"
)

func decodeReflect(i interface{}, buf []byte) os.Error {
	typ := reflect.TypeOf(i)
	val := reflect.ValueOf(i)
	if val.IsNil() {
		return nil
	}
	
	elem := val.Elem()
	buffer := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(buffer)
	
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		fieldType := typ.Elem().Field(i)
		if field.CanAddr() && fieldType.Name != "parent" {
			if err := decoder.DecodeValue(field.Addr()); err != nil {
				return err
			}
		}
	}
	
	return nil
}

func (block *Block) GobDecode(buf []byte) os.Error {
	return decodeReflect(block, buf)
}

func (file *File) GobDecode(buf []byte) os.Error {
	err := decodeReflect(file, buf)
	if err == nil {
		for _, block := range file.Blocks {
			block.parent = file
		}
	}
	return err	
}

func (dir *Dir) GobDecode(buf []byte) os.Error {
	err := decodeReflect(dir, buf)
	if err == nil {
		for _, file := range dir.Files {
			file.parent = dir
		}
		for _, subdir := range dir.SubDirs {
			subdir.parent = dir
		}
	}
	return err	
}

func encodeReflect(i interface{}) ([]byte, os.Error) {
	typ := reflect.TypeOf(i)
	val := reflect.ValueOf(i)
	if val.IsNil() {
		return make([]byte, 0), nil
	}
	
	elem := val.Elem()
	buffer := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buffer)
	
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		fieldType := typ.Elem().Field(i)
		if fieldType.Name != "parent" {
			if err := encoder.EncodeValue(field); err != nil {
				return nil, err
			}
		}
	}
	
	return buffer.Bytes(), nil
}

func (block *Block) GobEncode() ([]byte, os.Error) {
	return encodeReflect(block)
}

func (file *File) GobEncode() ([]byte, os.Error) {
	return encodeReflect(file)
}

func (dir *Dir) GobEncode() ([]byte, os.Error) {
	return encodeReflect(dir)
}



