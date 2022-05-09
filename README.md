# Lab1

实验指导手册：https://pdos.csail.mit.edu/6.824/labs/lab-mr.html

主目录上放置了mapreduce的论文
## 单机mapreduce
```bash
cd src/main
go run mrsequential.go wc.so pg*.txt
```
这样会发现这样的报错信息。
```txt
unexpected directory layout:
        import path: _/Users/bytedance/go/src/github.com/mit6.824/src/mr
        root: /Users/bytedance/go/src
        dir: /Users/bytedance/go/src/github.com/mit6.824/src/mr
        expand root: /Users/bytedance/go
        expand dir: /Users/bytedance/go/src/github.com/mit6.824/src/mr
        separator: /
```
其实是源文件的`../mr`找不到包。只用将`../mr` -> `github.com/mit6.824/src/mr` 然后就可以。

最后
```bash
go run mrsequential.go wc.so pg*.txt
more mr-out-0
```
最后就有可以出来了
```txt
A 509
ABOUT 2
ACT 8
ACTRESS 1
ACTUAL 8
ADLER 1
ADVENTURE 12
ADVENTURES 7
AFTER 2
AGREE 16
```
全部输出就到了`mr-out-0`文档中了

`go build -buildmode=plugin ../mrapps/wc.go` 该命令的作用是构建`wc.go`的动态链接库，`wc.go`实现了Map函数和Reduce函数，但是Map和Reduce随时都会改变，所以采用这种动态链接的方式，将其编译为`wc.so`文件。

所以我们用命令行运行，将`wc.so`文件带上的时候就会加载最新的打包好代码了。
```go
mapf, reducef := loadPlugin(os.Args[1])
```
同时在`mrapps`目录中都实现了`map, reduce`的函数，所以goLand中会报错。。。。如果我们想用其他文件的map和reduce函数就可以直接打包成'*.so'文件。

这样就做到了模块的解耦。

`mrsequential.go`实现的非分布式的，具体实现很简单：文章提取单词，调用wc.so的Map函数返回一个数组，数组元素为<word1, 1>对，然后排序，然后同样的单词会挨在一起，这样就能某个单词收集到一个数组传到list即可，然后将list传到Reduce函数，最后返回的结果写入文件即可。

## 分布式mapreduce
### 实验流程
首先看下实验手册的描述：
> The workers will talk to the coordinator via RPC. Each worker process will ask the coordinator for a task, read the task's input from one or more files, execute the task, and write the task's output to one or more files.

实验描述的很简单，建议还是好好看看论文的步骤流程来实现。

我实现的大致思路就是首先 worker通过rpc请求拉map任务，然后将输入的内容传递给用户自定义的map函数，通过hash分区，并将输出的数据存到本地磁盘(多进程用进程pid区别)。成功就告知master文件的位置，当所有的map操作都完成了以后，worker给master发reduce数据的请求。master将分区的所有的文件地址告诉worker，worker通过把数据弄到内存(外部排序不太会，也有复杂性)进行排序，然后执行用户定义的reduce操作，并写文件，写完后告诉master。

和论文有一些小出入的地方：
* 这里的map任务没有像论文描述的拆成16 ～ 64MB，直接是对一个文件进行解析
* 本来是map操作完了以后master给worker发任务，但是考虑到要在worker上加个rpc的复杂性，所以知道map执行完worker就直接向master请求reduce操作了
* reduce操作是worker直接将所有数据弄到内存里面，外部排序不太会
* worker执行完map或者reduce以后，告诉master前面的任务执行完了还是通过请求master任务的req结构体里面，因为再给master加个接口具有复杂性和不必要性。

**结构体设计**：

master的结构体设计，和论文描述差不太多：map和reduce任务的状态，由于没有正确的get到go这个rpc的设计，所以对应的worker的地址没有存。

master如论文描述，就像一个数据管道，中间文件存储区域的位置信息通过这个管道从Map传递到Reduce。

但是由于没有存worker的地址信息，所以也不存在论文描述的周期性ping，这里设计的就是通过map和reduce的状态来容错。
```go
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
```

rpc结构体的设计：

主要需要讲解的就是map和reduce的请求参数分别带有`Handled []string` 和 `HandleID []int`，设计的目的是为了下次请求master时，把这次已经处理完的任务发送给master。这样设计的原因是想偷懒，不想又建一个结构体，然后master多加个接口来更新任务的状态。

另外就是`ReduceRequest`里面的进程id没啥用，其他感觉注释都写得挺清楚的。
```go
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
```

结构体给出来了，大体框架也就出来了，剩下的具体实现可以去`src/mr/`目录下查看

### 说明
不是专门做计算的，所以此实验做出了让步，worker挂掉的情况没有考虑，其实考虑下来也很简单，我们的`Master`结构有记录每个任务的信息:
```go
// 按照论文中的描述，map时的状态。
// 0：表示文件没有被读取
// 1：表示文件发送给worker但是没有返回结果的
// 2：成功发送给worker，并且被成功执行
mapState []int
```
挂掉只需要将mapState的值重新设置为0，其他任务就会来争抢这个任务。并且没有真正成功的worker我们是不会将其内容写进磁盘，让master感知到的。

为什么不做的原因主要实在不想学go的定时线程，社区太烂了.....同时如果真的做了对自己提升也不大，毕竟不是做计算层的，论文给的看明白了即可。同时我认为最重要的还是其处理数据的流程图：

![image](https://user-images.githubusercontent.com/57765968/167312391-04d75966-d365-433e-aa2d-c5b916d7f49d.png)


对于单机多进程的环境，代码实现和论文给出的流程图几乎实现一样，我觉得这是我想要的。
