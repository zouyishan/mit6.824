package mr

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strconv"
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

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// Worker
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the master.
	// CallExample()

	resp := MapResponse{}
	resp.ContentMap = make(map[string]string)

	resp.IsEnd = false
	count := 0
	temp := make([]string, 0)
	for resp.IsEnd == false {
		req := MapRequest{
			MaxFileNum: 2,
		}
		req.Handled = temp
		temp = temp[:0]
		var intermediate []KeyValue
		call("Master.Dispense", &req, &resp)
		fmt.Printf("返回的resp: %v, 长度: %v\n", resp.IsEnd, len(resp.ContentMap))
		req.Handled = make([]string, 1)
		for k, v := range resp.ContentMap {
			fmt.Printf("what is k: %v", k)
			temp = append(temp, k)
			req.Handled = append(req.Handled, k)
			kva := mapf(k, v)
			intermediate = append(intermediate, kva...)
		}
		fmt.Printf("新增的req: %v \n", req)

		if len(resp.ContentMap) == 0 || len(req.Handled) == 0 {
			break
		}
		sort.Sort(ByKey(intermediate))
		oName := "mr-out-"
		oName += strconv.Itoa(os.Getpid())
		oName += "-"
		oName += strconv.Itoa(count)
		count++
		oFile, _ := os.Create(oName)

		i := 0
		for i < len(intermediate) {
			j := i + 1
			for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
				j++
			}
			var values []string
			for k := i; k < j; k++ {
				values = append(values, intermediate[k].Value)
			}
			output := reducef(intermediate[i].Key, values)
			// fmt.Printf("开始写数据")
			// this is the correct format for each line of Reduce output.
			fmt.Fprintf(oFile, "%v %v\n", intermediate[i].Key, output)

			i = j
		}

		oFile.Close()
		resp.ContentMap = nil
	}

	fmt.Printf("end pid: %v", os.Getpid())
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
