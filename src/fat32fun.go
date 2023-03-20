package main

import (
	"os"
	"strings"
)

// 根据首簇号获取簇链
func FindClus(start int, fato int, fp *os.File) []int {
	clusChain := make([]int, 0)
	nxtClus := start
	bufT := make([]byte, 512)
	_, err := fp.ReadAt(bufT, int64(fato))
	//fmt.Println(bufT)
	if err != nil {
		return nil
	}
	for nxtClus >= 0x02 && nxtClus < 0x0fffffff {
		//fmt.Println(nxtClus)
		clusChain = append(clusChain, nxtClus)
		ncOffset := nxtClus * 4
		nxtClus = Byte2I(bufT[ncOffset : ncOffset+4])
	}
	return clusChain
}

// GetLongName 由短文件名目录项回头查找长文件名
func GetLongName(buf []byte, pos int, len int) string {
	var checkByte []byte
	bufLT := make([]byte, 32)
	endflag := 0
	for ; endflag == 0; pos -= 32 {
		lc := new(LongC)
		bufLT = buf[pos-32 : pos]
		lc.Flag = bufLT[0]
		lc.Fname1 = [10]byte(bufLT[0x1:0xb])
		lc.Fname2 = [12]byte(bufLT[0xe:0x1a])
		lc.Fname3 = [4]byte(bufLT[0x1c:0x20])
		checkByte = append(append(append(checkByte, lc.Fname1[:]...), lc.Fname2[:]...), lc.Fname3[:]...)
		if GetBit(lc.Flag, 6) == 1 {
			endflag = 1
		}
	}
	checkName := strings.Split(string(checkByte), string(0x00))[:len]
	checkName1 := strings.Join(checkName, "")
	return checkName1
}
