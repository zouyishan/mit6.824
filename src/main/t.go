package main

import (
	"fmt"
	"os"
)

func main() {
	os.Mkdir("GGGGG", os.ModePerm)
	//os.Create("ttttt/fdf.txt")
	create, err := os.Create("GGGGG/fdsfa.txt")
	if err != nil {
		fmt.Printf("g了啊 %v\n", err)
	}
	file, err := os.Create("GGGGG/jjj.txt")
	if err != nil {
		fmt.Printf("g了啊 %v\n", err)
	}
	file.Close()
	create.Close()
	//file, err := os.OpenFile("ttttt/fdf.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	//if err != nil {
	//	fmt.Errorf("open error")
	//}
	//defer file.Close()
	//err := ioutil.WriteFile("ttttt/fdf.txt", []byte("uuiubgfdff\nfdxhgcj"), 0666)
	//if err != nil {
	//	fmt.Println("G")
	//}
	//write, err := file.Write([]byte("\nGGGGGGGG\nCCCCCCC\n"))
	//if err != nil {
	//	fmt.Printf("出问题G %v", err)
	//	return
	//}
	//fmt.Println(write)
}
