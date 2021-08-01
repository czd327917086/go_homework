package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

/*
问题描述：
基于 errgroup 实现一个 http server 的启动和关闭 ，以及 linux signal 信号的注册和处理，要保证能够一个退出，全部注销退出。

根据描述信息，可以简单汇总成3块内容：
1.实现HTTP server的启动和关闭
2.监听linux signal信号，使用chan实现对linux signal中断的注册和处理 按ctrl+c之类退出程序
3.errgroup实现多个goroutine的级联退出，通过errgroup+context的形式，对1、2中的goroutine进行级联注销
*/

func main() {
	// WithContext返回一个新的Group和一个从参数ctx派生的关联上下文。
	g, ctx := errgroup.WithContext(context.Background())

	// 增加一个http handler
	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})

	// 利用无缓冲chan 模拟单个服务错误退出
	serverOut := make(chan struct{})
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		serverOut <- struct{}{} // 阻塞
	})

	server := http.Server{
		Addr: ":8080",
	}

	// -- 测试http server的启动和退出

	// g1 退出后, context 将不再阻塞，g2, g3 都会随之退出
	// 然后 main 函数中的 g.Wait() 退出，所有协程都会退出
	g.Go(func() error {
		return server.ListenAndServe() // 启动http server服务
	})

	// g2 退出时，调用了 shutdown，g1 会退出
	// g2 退出后, context 将不再阻塞，g3 会随之退出
	// 然后 main 函数中的 g.Wait() 退出，所有协程都会退出
	g.Go(func() error {
		select {
		case <-ctx.Done():
			log.Println("errgroup exit...")
		case <-serverOut:
			log.Println("server will out...")
		}

		timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		log.Println("shutting down server...")
		return server.Shutdown(timeoutCtx) // 关闭http server
	})

	// -- 测试linux signal信号的注册和处理

	// g3 捕获到 os 退出信号将会退出
	// g3 退出后, context 将不再阻塞，g2 会随之退出
	// g2 退出时，调用了 shutdown，g1 会退出
	// 然后 main 函数中的 g.Wait() 退出，所有协程都会退出
	g.Go(func() error {
		c := make(chan os.Signal, 1) // 一般设置大小为1的缓冲区
		// 设置信号通知 syscall.SIGINT表示：用户发送INTR字符(Ctrl+C)触发; syscall.SIGTERM表示：结束程序
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-c:
			return errors.Errorf("get os signal: %v", sig)
		}
	})

	fmt.Printf("errgroup exiting: %+v\n", g.Wait())
}
