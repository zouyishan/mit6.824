package mr

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"
)

type Master struct {
	// Your definitions here.
	// 一共有多少个map任务
	nMap int
	// 有n个reduce任务
	nReduce int
	// 已经执行完成的map的任务
	mapExecuted int
	// 已经执行完成的reduce的任务
	reduceExecuted int

	// 文件内容
	fileNames []string
	// 按照论文中的描述，map时的状态。
	// 0：表示文件没有被读取
	// 1：表示文件发送给worker但是没有返回结果的
	// 2：成功发送给worker，并且被成功执行
	mapState []int
	// 按照论文中的描述，map时的状态
	// 0：表示文件没有被读取
	// 1：表示文件发送给worker但是没有返回结果的
	// 2：成功发送给worker，并且被成功执行
	reduceState []int
	// 多线程加锁操作
	mutex sync.Mutex

	// 对于已经完成map操作的worker的id标识
	// 由于是单机多进程，所以这里的标识为进程的pid
	ids []int

	// master存的map函数生成的文件名字
	// 一共有nReduce个文件名字
	mapFileName []string
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

// ReduceTask
// 计算map后的结果
func (m *Master) ReduceTask(req *ReduceRequest, resp *ReduceResponse) error {
	fmt.Printf("m.nReduce:%v m.reduceExecuted:%v req:%v\n", m.nReduce, m.reduceExecuted, req)

	if len(req.HandleID) > 0 {
		for i := range req.HandleID {
			fmt.Printf("reduce executed: %v", i)
			m.reduceState[req.HandleID[i]] = 2
			m.reduceExecuted++
		}
	}

	if m.reduceExecuted == m.nReduce {
		resp.IsEnd = true
		return nil
	}

	resp.fileNames = make([]string, 0)
	resp.ids = m.ids
	m.mutex.Lock()
	fmt.Printf("===开始执行函数reduceExecute: %v\n", resp)
	m.reduceExecute(req, resp)
	fmt.Printf("然后的resp是: %v\n", resp)
	m.mutex.Unlock()
	return nil
}

func isRepeat(arr []int, x int) bool {
	if len(arr) == 0 {
		return false
	}
	for i := 0; i < len(arr); i++ {
		if arr[i] == x {
			return true
		}
	}
	return false
}

// MapTask
// 分发给worker任务，worker执行任务可能会中途挂掉。
func (m *Master) MapTask(req *MapRequest, resp *MapResponse) error {
	fmt.Printf("m.nFile:%v m.mapExecuted:%v req:%v\n", m.nMap, m.mapExecuted, req)

	if len(req.Handled) > 0 {
		for i := range req.Handled {
			for j := range m.fileNames {
				if m.fileNames[j] == req.Handled[i] {
					fmt.Printf("file executed: %v\n", m.fileNames[j])
					m.mapState[j] = 2
					if !isRepeat(m.ids, req.Id) {
						m.ids = append(m.ids, req.Id)
					}
					m.mapExecuted++
				}
			}
		}
	}

	if m.mapExecuted == m.nMap {
		resp.IsEnd = true
		return nil
	}
	resp.NReduce = m.nReduce
	resp.ContentMap = make(map[string]string)
	fmt.Printf("===开始执行 %v, %v\n", req.Handled, resp)
	m.mutex.Lock()
	m.mapExecute(req, resp)
	m.mutex.Unlock()
	return nil
}

func (m *Master) mapExecute(req *MapRequest, resp *MapResponse) {
	sendCount := 0
	resp.IsEnd = false

	if m.mapExecuted == m.nMap {
		resp.IsEnd = true
		return
	}

	for i := 0; i < m.nMap; i++ {
		// 分发
		if m.mapState[i] == 0 && sendCount < req.MaxFileNum {
			sendCount++
			m.mapState[i] = 1
			file, err := os.Open(m.fileNames[i])
			if err != nil {
				log.Fatalf("cannot open %v", m.fileNames[i])
			}
			c, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", m.fileNames[i])
			}
			file.Close()
			fmt.Printf("name:%v  content:%v\n", m.fileNames[i], len(c))
			//TODO 定时器，如果10秒后，mapState的状态还是1，那么就将状态改为0
			resp.ContentMap[m.fileNames[i]] = string(c)
		}
	}
}

func (m *Master) reduceExecute(req *ReduceRequest, resp *ReduceResponse) error {
	if m.nReduce == m.reduceExecuted {
		resp.IsEnd = true
		return nil
	}
	countTask := 0
	fmt.Printf("开始发送文件\n")
	for i := 0; i < len(m.reduceState); i++ {
		if m.reduceState[i] == 0 && countTask < req.MaxFileNum {
			countTask++
			resp.handId = append(resp.handId, i)
			resp.fileNames = append(resp.fileNames, "reduceFile"+strconv.Itoa(i))
			//TODO 定时器，如果10秒后，reduceState的状态还是1，那么就将状态改为0
			m.reduceState[i] = 1
		}
	}
	fmt.Printf("返回的结果为：%v\n", resp)
	return nil
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
	if m.mapExecuted == m.nMap && m.reduceExecuted == m.nReduce {
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
	m.nMap = len(files)
	m.nReduce = nReduce
	m.mapExecuted = 0
	m.reduceExecuted = 0
	m.fileNames = files

	m.mapState = make([]int, m.nMap+1)
	m.reduceState = make([]int, m.nReduce+1)
	m.mapFileName = make([]string, m.nReduce+1)

	m.server()
	return &m
}
