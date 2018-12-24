package webapp

import (
	"fmt"
	"errors"
	"io"
	"ioutil"
)

type fileCache struct {
	cache, seeker []byte
	length, position int64
}//-- end fileCache struct

func (fc *fileCache) Read (p []byte) (int, error) {
	return copy(p, fc.seeker), nil
}//-- end func fileCache.Read

func (fc *fileCache) Seek (offset int64, whence int) (int64, error) {
	newPos := int64(0)
	switch (whence) {
		case io.SeekStart:
			newPos = offset
			break
		case io.SeekCurrent:
			newPos = fc.position + offset
			break
		case io.SeekSet:
			newPos = fc.length + offset - 1
			break
		default:
			return fc.position, fmt.Errorf("invalid whence (%d)", whence)
	}//-- end switch whence
	if newPos < 0 {
		return fc.position, errors.New("cannot seek before 0")
	}
	if newPos >= fc.length { newPos = fc.length - 1 }
	fc.position, fc.seeker = newPos, fc.cache[newPos:]
	return fc.position, nil
}//-- end func fileCache.Seek

func newFileCache (filename string) (*fileCache, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &fileCache{cache: content, seeker: content,
		length: len(content), position: 0}, nil
}//-- end func New

