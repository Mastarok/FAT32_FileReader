package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// Byte2I 字节数组转整形
func Byte2I(b []byte) int {
	var result int
	for i, v := range b {
		result += int(v) << (8 * i)
	}
	return result
}

// FileCompare 逐字节比对文件
func FileCompare(f1 *os.File, f2 *os.File) bool {
	buf1 := make([]byte, 512)
	buf2 := make([]byte, 512)
	for {
		_, err1 := f1.Read(buf1)
		_, err2 := f2.Read(buf2)
		if err1 != nil || err2 != nil {
			if err1 != err2 {
				return false
			}
			if err1 == io.EOF {
				break
			}
		}
		if bytes.Equal(buf1, buf2) {
			continue
		}
		return false
	}
	return true
}

// FileMD5 计算文件MD5
func FileMD5(f *os.File) (string, error) {
	hash := md5.New()
	_, _ = io.Copy(hash, f)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetBit 单字节位截取
func GetBit(b byte, id int) int {
	bit := (int)((b >> id) & 0x1)
	return bit
}
