package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//
import (
	"bytes"
	"encoding/gob"
	"io"
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

type MapRequest struct {
	// 由于是单机多进程worker进程的标识是pid
	Id int
	// worker最大的文件请求数，默认为1。
	// 论文默认是1，这里留作扩展
	MaxFileNum int
	// 处理完的文件
	Handled []string
}

type MapResponse struct {
	// master上的map任务是否已经处理完了。
	IsEnd bool
	// 可能和MaxFileNum请求的数量不一样。
	// 因为没有被mapreduce的文件没有那么多了。
	// 也可能为null，但是IsEnd没有被标记为true之前，还是要请求。
	ContentMap map[string]string
	// 一共要分为多少个区域
	NReduce int
}

type ReduceRequest struct {
	// worker进程标记
	Id int
	// worker最大的文件处理数, 默认为1。
	// 论文默认是1，这里留作扩展
	MaxFileNum int
	// 处理完的文件Id
	HandleID []int
}

type ReduceResponse struct {
	// master上的reduce任务是否已经处理完了
	IsEnd bool
	// 要处理的磁盘文件，实际就是1个
	// 论文默认是1，这里留作扩展
	FileNames []string
	// 由于是单机多进程，所以这里存的是所有worker成功的pid的目录
	Ids []int
	// 这次ok的ID
	HandId []int
}

// ToJSONResponse 为什么会有这个呢？当然是为了兼容go那奇怪的不能传递数组啊
type ToJSONResponse struct {
	JsonString string
}

func Encode(data interface{}) *bytes.Buffer {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(data)
	return &buf
}

func MapResponseDecode(data interface{}) *MapResponse {
	d := data.(io.Reader)
	dec := gob.NewDecoder(d)
	var res MapResponse
	dec.Decode(&res)
	return &res
}

func MapRequestDecode(data interface{}) *MapRequest {
	d := data.(io.Reader)
	dec := gob.NewDecoder(d)
	var res MapRequest
	dec.Decode(&res)
	return &res
}

func ReduceResponseDecode(data interface{}) *ReduceResponse {
	d := data.(io.Reader)
	dec := gob.NewDecoder(d)
	var res ReduceResponse
	dec.Decode(&res)
	return &res
}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the master.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
