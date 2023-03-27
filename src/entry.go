package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

const Grcnts = 256 //4096*2÷32,协程数量

var nxt int //记录下个目录的簇偏移

func main() {
	if len(os.Args) > 2 {
		fmt.Println("请输入单个目录")
		return
	}
	fullPath := os.Args[1]
	opath := strings.Split(fullPath, "/")
	fpath := strings.Split(strings.ToUpper(fullPath), "/") //路径分割转大写
	if len(fpath) < 2 {
		fmt.Println("路径无效")
		return
	}
	parName := "\\\\.\\" + fpath[0] //分区号
	odir := opath[1:]
	dir := fpath[1:]
	name := make([]string, len(dir)) //每部分路径的名称
	ext := make([]string, len(dir))  //每部分路径的扩展
	for i, v := range dir {          //根据fat32存储长短文件名的规则处理名字和扩展
		if i != len(dir)-1 {
			name[i] = v
			ext[i] = ""
			if len(name[i]) <= 8 {
				name[i] += strings.Repeat(" ", 8-len(name[i]))
				ext[i] += strings.Repeat(" ", 3-len(ext[i]))
			} else {
				name[i] = name[i][:6] + "~ "
				ext[i] += strings.Repeat(" ", 3-len(ext[i]))
			}
		} else {
			if v == "" {
				fmt.Println("路径无效")
				return
			}
			name[i] = strings.Split(v, ".")[0]
			if find := strings.Contains(v, "."); find {
				ext[i] = strings.Split(v, ".")[1]
			} else {
				ext[i] = ""
			}
			if len(name[i]) <= 8 && len(ext[i]) <= 3 {
				name[i] += strings.Repeat(" ", 8-len(name[i]))
				ext[i] += strings.Repeat(" ", 3-len(ext[i]))
			}
			if len(name[i]) > 8 && len(ext[i]) <= 3 {
				name[i] = name[i][:6] + "~ "
			}
			if len(name[i]) > 8 && len(ext[i]) > 3 {
				name[i] = name[i][:6] + "~ "
				ext[i] = ext[i][0:3]
			}
			if len(name[i]) <= 8 && len(ext[i]) > 3 {
				if len(name[i]) <= 6 {
					name[i] = name[i] + "~ "
				} else {
					name[i] = name[i][:6] + "~ "
				}
				ext[i] = ext[i][0:3]
			}
		}
	}
	//fmt.Println(name, ext)

	fi, err := os.Open(parName)
	if err != nil {
		fmt.Println(err)
		return
	}
	if fi != nil {
		defer fi.Close()
	}
	//r := bufio.NewReader(fi)
	buf := make([]byte, 4096*2) //从引导扇区开始读
	_, err = fi.Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	fbpb := new(BPB) //填充BPB
	fbpb.BytePerSector = Byte2I(buf[0x0b:0x0d])
	fbpb.SectorPerClus = int(buf[0x0d])
	fbpb.ResSector = Byte2I(buf[0x0e:0x10])
	fbpb.NumFAT = int(buf[0x10])
	fbpb.SectorPerFAT = Byte2I(buf[0x24:0x28])
	//fmt.Printf("% X", fbpb)
	RootOffset := (fbpb.ResSector + fbpb.SectorPerFAT*fbpb.NumFAT) * 512 //计算根目录偏移
	FatOffset := fbpb.ResSector * 512                                    //计算FAT偏移
	nxt = RootOffset
	fmt.Printf("根目录偏移：%X\nFAT偏移%X\n", RootOffset, FatOffset)
	var fok bool
	for idx := range dir {
		fok, err = FindFile(idx, fi, buf, nxt, name, ext, dir, odir, fullPath, FatOffset, RootOffset)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	if !fok {
		fmt.Println("文件不存在")
	}
}

/*
Findfile完成的工作：每次处理路径的一个部分（由/分割）,在偏移=nxtoffset的位置读取一个簇大小的数据，由128个协程并行查找每一个目录项（每个32字节，默认为短目录项）
如果该目录项的名称和name[]中对应的名称相同（短文件名的情况下直接比较，长文件名情况下要进行处理），那么就求出簇链，且进一步判断目录项属性是目录还是文件，
如果是目录，就计算和更新下一次执行时的目录偏移nxoffset并返回，如果是文件就根据文件长度和簇链到相应的簇偏移处读取数据并写入新的文件，再和源文件逐字节比对&&MD5比对。循环执行Findfile就能处理好路径的每个部分。
*/
func FindFile(idx int, fi *os.File, buf []byte, nxtoffset int, name, ext, dir, odir []string, fullPath string, FatOffset, RootOffset int) (bool, error) {
	ok := false                                //是否找到文件的标志
	_, err := fi.ReadAt(buf, int64(nxtoffset)) //读取当前目录的首簇数据
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	//fmt.Printf("% X\n", buf)
	var wg sync.WaitGroup
	wg.Add(Grcnts)
	chlock := make(chan int, 1)
	defer close(chlock)
	for i := 0; i < Grcnts; i++ {
		go func(i int, c chan int) {
			defer wg.Done()
			pos := i * 32
			buf1 := buf[pos : pos+32]
			if buf1[0] == 0x00 || buf1[0] == 0xe5 {
				return
			}
			sc := new(ShortC) //填充短文件目录项
			sc.FileName = string(buf1[0x0:0x8])
			sc.ExtName = string(buf1[0x8:0x0B])
			sc.Attr = buf1[0x0B]
			sc.StartClus = Byte2I(buf1[0x14:0x16])<<8 + Byte2I(buf1[0x1A:0x1C])
			sc.FileLen = Byte2I(buf1[0x1C:])
			fvalid := false
			//fmt.Printf("% v\n", sc)
			if name[idx] == strings.Split(sc.FileName, "~")[0]+"~ " && sc.ExtName == ext[idx] {
				checkName := GetLongName(buf, pos, len(dir[idx]))
				//fmt.Println(checkName)
				if checkName == odir[idx] {
					fvalid = true
				}
			}
			if name[idx] == sc.FileName && sc.ExtName == ext[idx] {
				fvalid = true
			}
			if !fvalid {
				return
			}
			clusChain := FindClus(sc.StartClus, FatOffset, fi)
			if sc.Attr == 0x10 {
				for _, v := range clusChain {
					nxt = RootOffset + (v-2)*512*8
					return
				}
			}

			if sc.Attr == 0x20 {
				ok = true
				fcheckName := strings.Split(strings.ToUpper(fullPath), "/")[0] + "/" + "+" + odir[idx]
				ft, err := os.Create(fcheckName)
				if err != nil {
					fmt.Println(err)
					return
				}
				if ft != nil {
					defer ft.Close()
				}
				leftLen := sc.FileLen
				bufW := make([]byte, 4096)
				for _, v := range clusChain {
					clusOffset := RootOffset + (v-2)*512*8
					_, err = fi.ReadAt(bufW, int64(clusOffset))
					if err != nil {
						return
					}
					if leftLen > 4096 {
						_, err := ft.Write(bufW)
						if err != nil {
							return
						}
						leftLen -= 4096
					} else {
						_, err := ft.Write(bufW[:leftLen])
						if err != nil {
							return
						}
					}
				}
				fmt.Println("拼接得到文件保存在：", fcheckName)
				fs, err := os.Open(fullPath)
				if err != nil {
					fmt.Println(err)
					return
				}
				if fs != nil {
					defer fs.Close()
				}
				_, _ = fs.Seek(0, 0)
				_, _ = ft.Seek(0, 0)
				sum1, _ := FileMD5(fs)
				sum2, _ := FileMD5(ft)
				//chlock <- 1
				fmt.Printf("短文件名目录项信息：\n文件名：%s\n扩展名：%s\n文件长度：%d字节\n文件首簇号：%d\n", sc.FileName, sc.ExtName, sc.FileLen, sc.StartClus)
				fmt.Printf("文件簇链：%v\n", clusChain)
				if sum1 == sum2 {
					fmt.Printf("源文件MD5：%s\n新文件MD5：%s\nmd5校验文件相同\n", sum1, sum2)
				}
				if FileCompare(fs, ft) {
					fmt.Println("经过逐字节比对，两文件相同")
				}
				//<-chlock
			}
		}(i, chlock)
	}
	wg.Wait()
	return ok, nil
}
