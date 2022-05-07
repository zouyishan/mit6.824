# mit6.824
6.824实验

记录菜的坑
# Lab1
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