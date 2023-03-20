package main

type BPB struct {
	BytePerSector, SectorPerClus, ResSector, NumFAT, SectorPerFAT int
}

// ShortC 短文件名目录项
type ShortC struct {
	FileName  string //文件名
	ExtName   string //扩展名
	Attr      byte   //属性
	StartClus int    //起始簇
	FileLen   int    //文件长度
}

// LongC 长文件名目录项
type LongC struct {
	Flag   byte     //标志位
	Fname1 [10]byte //长文件名，unicode16
	Fname2 [12]byte
	Fname3 [4]byte
}
