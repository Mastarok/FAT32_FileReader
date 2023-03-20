# FAT32_FileReader
读取fat32磁盘中的文件目录项信息以及文件簇链  
输入：任一文件（A）的路径（英文）  
输出：  
（1）该文件的短文件名目录项信息  
（2）该文件的簇链  
（3）根据上述的文件簇链，从磁盘上提取数据并拼接而得的新文件（B）  
（4）文件A与文件B内容的比较结果  
![image](https://user-images.githubusercontent.com/96651300/226340240-319d59c6-1e1a-412a-ad75-a50db879a5e0.png)

