# Lab1

实验指导手册：https://pdos.csail.mit.edu/6.824/labs/lab-mr.html
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
首先看下实验手册的描述：
> The workers will talk to the coordinator via RPC. Each worker process will ask the coordinator for a task, read the task's input from one or more files, execute the task, and write the task's output to one or more files.

worker通过RPC拉取任务，然后执行这些任务，输出一个或多个文件。

