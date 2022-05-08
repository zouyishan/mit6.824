package mr

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"strings"
)

// KeyValue
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

// ByKey 通过key来排序
type ByKey []KeyValue

// Len for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

var flag bool

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func createFile() error {
	if flag == true {
		return nil
	}
	dirName := strconv.Itoa(os.Getpid())
	os.Mkdir(dirName, os.ModePerm)
	for i := 0; i < 10; i++ {
		fileName := dirName + "/reduceFile" + strconv.Itoa(i)
		file, err := os.Create(fileName)
		if err != nil {
			fmt.Printf("进程创建文件失败 %v\n", err)
		}
		file.Close()
	}
	return nil
}

// Worker
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the master.
	// CallExample()

	// 本地创建对应进程的文件
	createFile()
	doMap(mapf)
	doReduce(reducef)
	fmt.Printf("end pid: %v\n", os.Getpid())
}

func doMap(mapf func(string, string) []KeyValue) error {
	resp := MapResponse{
		IsEnd: false,
	}
	temp := make([]string, 0)
	for resp.IsEnd == false {
		req := MapRequest{
			MaxFileNum: 1,
			Id:         os.Getpid(),
		}
		req.Handled = temp
		temp = temp[:0]
		call("Master.MapTask", &req, &resp)
		fmt.Printf("返回的resp: %v, 长度: %v\n", resp.IsEnd, len(resp.ContentMap))
		req.Handled = make([]string, 1)

		var partitionData [][]KeyValue
		partitionData = make([][]KeyValue, 10)
		for k, v := range resp.ContentMap {
			fmt.Printf("what is k: %v\n", k)
			temp = append(temp, k)
			req.Handled = append(req.Handled, k)
			// 传递给用户自己定义的map函数
			kva := mapf(k, v)
			for i := range kva {
				// 根据k分区
				x := ihash(kva[i].Key) % resp.NReduce
				partitionData[x] = append(partitionData[x], kva[i])
			}
		}

		if flag == false {
			err := createFile()
			if err != nil {
				log.Fatalf("cannot createFile %v", os.Getpid())
			}
			flag = true
		}

		// 写分区内容
		for i := 0; i < resp.NReduce; i++ {
			name := strconv.Itoa(os.Getpid()) + "/reduceFile" + strconv.Itoa(i)
			fmt.Printf("打开文件: %v\n", name)
			file, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModeAppend) // ignore_security_alert
			if err != nil {
				log.Fatalf("cannot open %v", name)
			}
			for j := 0; j < len(partitionData[i]); j++ {
				str := partitionData[i][j].Key + " " + partitionData[i][j].Value + "\n"
				file.Write([]byte(str))
			}
			file.Close()
		}

		fmt.Printf("新增的req: %v\n", req)
		if len(resp.ContentMap) == 0 || len(req.Handled) == 0 {
			break
		}
		resp.ContentMap = nil
	}
	return nil
}

func doReduce(reducef func(string, []string) string) error {
	resp := ReduceResponse{
		IsEnd: false,
	}
	fmt.Printf("开始执行Reduce了\n")
	for resp.IsEnd == false {
		req := ReduceRequest{
			Id:         os.Getpid(),
			MaxFileNum: 1,
		}
		call("Master.ReduceTask", &req, &resp)
		fmt.Printf("返回的参数是：%v\n", resp)
		if resp.IsEnd == true || len(resp.fileNames) == 0 {
			return nil
		}
		req.HandleID = resp.handId

		// 实际上只有一个, 遍历是为了留作扩展用。
		var res []KeyValue
		for i := range resp.fileNames {
			for j := range resp.ids {
				fileName := strconv.Itoa(resp.ids[j])
				fileName += "/"
				fileName += resp.fileNames[i]
				fmt.Printf("文件名称是：%v", fileName)
				file, err := os.Open(fileName)
				if err != nil {
					fmt.Printf("g了 打开文件失败了, %v", err)
				}
				br := bufio.NewReader(file)
				for {
					line, b, err := br.ReadLine()
					if err != nil {
						fmt.Printf("g了 读取行失败了, %v", err)
						break
					}
					if b {
						fmt.Printf("g了 一行的字节太多了, %v", err)
					}
					var temp KeyValue
					str := strings.Fields(string(line))
					temp.Key = str[0]
					temp.Value = str[1]
					res = append(res, temp)
				}
				file.Close()
			}
		}
		sort.Sort(ByKey(res))

		oName := "mr-out-"
		oName += strconv.Itoa(os.Getpid())
		oName += resp.fileNames[0]
		oFile, _ := os.Create(oName)
		i := 0
		for i < len(res) {
			j := i + 1
			for j < len(res) && res[j].Key == res[i].Key {
				j++
			}
			var values []string
			for k := i; k < j; k++ {
				values = append(values, res[k].Value)
			}
			output := reducef(res[i].Key, values)
			// fmt.Printf("开始写数据")
			// this is the correct format for each line of Reduce output.
			fmt.Fprintf(oFile, "%v %v\n", res[i].Key, output)

			i = j
			oFile.Close()
		}
	}
	return nil
}

// CallExample
// example function to show how to make an RPC call to the master.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	call("Master.Example", &args, &reply)

	// reply.Y should be 100.
	fmt.Printf("reply.Y %v\n", reply.Y)
}

//
// send an RPC request to the master, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := masterSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
