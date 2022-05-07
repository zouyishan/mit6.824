package mr

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
)

type Master struct {
	// Your definitions here.
	// 一共有多少个文件
	nFile int
	// 这里节点可能会挂
	executed int
	// 文件内容
	fileNames []string
	// 可以理解为记录状态的数组
	// 0：表示文件没有被读取
	// 1：表示文件发送给worker但是没有返回结果的
	// 2：成功发送给worker，并且被成功执行
	state []int
	// 多线程加锁操作
	mutex sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.

// Example
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// Dispense
// 分发给worker任务，worker执行任务可能会中途挂掉。
func (m *Master) Dispense(req *MapRequest, resp *MapResponse) error {
	fmt.Printf("m.nFile:%v m.executed:%v req:%v\n", m.nFile, m.executed, req)

	if len(req.Handled) > 0 {
		for i := range req.Handled {
			for j := range m.fileNames {
				if m.fileNames[j] == req.Handled[i] {
					fmt.Printf("file executed: %v", m.fileNames[j])
					m.state[j] = 2
					m.executed++
				}
			}
		}
	}

	if m.executed == m.nFile {
		resp.IsEnd = true
		return nil
	}
	resp.ContentMap = make(map[string]string)
	fmt.Printf("===开始执行 多线程 %v, %v\n", req.Handled, resp)
	m.execute(req, resp)
	return nil
}

func (m *Master) execute(req *MapRequest, resp *MapResponse) {
	count := 0
	sendCount := 0
	resp.IsEnd = false

	//m.mutex.Lock()

	if count == m.nFile {
		resp.IsEnd = true
		return
	}

	for i := 0; i < m.nFile; i++ {
		// 分发
		if m.state[i] == 0 && sendCount < req.MaxFileNum {
			sendCount++
			m.state[i] = 1
			file, err := os.Open(m.fileNames[i])
			if err != nil {
				log.Fatalf("cannot open %v", m.fileNames[i])
			}
			c, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", m.fileNames[i])
			}
			file.Close()
			fmt.Printf("name:%v  content:%v", m.fileNames[i], len(c))
			resp.ContentMap[m.fileNames[i]] = string(c)
			//fmt.Printf("Content is: %v \n", resp.ContentMap)
		} else if m.state[i] == 2 {
			count++
		}
	}

	if count == m.nFile {
		resp.IsEnd = true
	}
	//m.mutex.Unlock()
}

//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// Done
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	ret := false
	if m.executed == m.nFile {
		ret = true
	}
	return ret
}

// MakeMaster
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}

	// Your code here.
	m.nFile = len(files)
	m.executed = 0
	m.fileNames = files
	m.state = make([]int, m.nFile+1)

	m.server()
	return &m
}
