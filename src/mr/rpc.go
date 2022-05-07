package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//
import (
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

type MapResponse struct {
	// 是否已经处理完了。
	IsEnd bool
	// 可能和MaxFileNum请求的数量不一样。
	// 因为没有被mapreduce的文件没有那么多了。
	// 也可能为null，但是IsEnd没有被标记为true之前，还是要请求。
	ContentMap map[string]string
}

type MapRequest struct {
	// 最大的文件请求数，默认为2。
	MaxFileNum int
	// 处理完的文件
	Handled []string
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
