package webapp

import (
	"os"
	"fmt"
	"errors"
	"io"
)

type fileCache struct {
	cache, seeker []byte
	length, position int64
	stat os.FileInfo
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
		case io.SeekEnd:
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

func (fc *fileCache) Stat () (os.FileInfo, error) {
	return fc.stat, nil
}//--end fileCache.Stat

func newFileCache (filename string) (*fileCache, error) {
	file, err := os.Open(filename)
	if err != nil { return nil, err }
	defer file.Close()
	flen, _ := file.Seek(0, io.SeekEnd)
	content := make([]byte, int(flen))
	stat, _ := file.Stat()
	file.ReadAt(content, 0)
	return &fileCache{cache: content, seeker: content,
		length: flen, position: int64(0), stat: stat}, nil
}//-- end func New

