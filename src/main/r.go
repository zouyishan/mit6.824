package main

import (
	"bufio"
	"fmt"
	"github.com/mit6.824/src/mr"
	"os"
	"strings"
)

func main() {
	file, err := os.Open("GGGGG/jjj.txt")
	if err != nil {
		fmt.Printf("g了 打开文件失败了, %v", err)
	}
	br := bufio.NewReader(file)
	var res []mr.KeyValue
	for {
		line, b, err := br.ReadLine()
		if err != nil {
			fmt.Printf("g了 读取行失败了, %v", err)
			break
		}
		if b {
			fmt.Printf("g了 一行的字节太多了, %v", err)
		}
		var temp mr.KeyValue
		str := strings.Fields(string(line))
		temp.Key = str[0]
		temp.Value = str[1]
		res = append(res, temp)
	}
	fmt.Printf("文件内容是：%v\n", res)
	//
	//if err != nil {
	//	fmt.Printf("g了 读取文件失败了, %v", err)
	//}
	//e := json.Unmarshal(fileContent, &res)
	//if e != nil {
	//	fmt.Printf("G了啊，转化出错了: %v", e)
	//}
	//fmt.Printf("%v", res)
}
