// golang.org/x/sync/errgroup 实现了多任务只要有一个任务出错，
// 就取消其他任务，并返回第一个错误信息。当然也实现了pipeline :
// https://pkg.go.dev/golang.org/x/sync/errgroup
// 也要注意使用, 拿justError 例子：

package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
)

func main() {
	//g := new(errgroup.Group)
	g, ctx := errgroup.WithContext(context.Background()) //应该生成新的ctx
	var urls = []string{
		"http://www.golang.org/",
		"http://www.google.com/",
		"http://www.somestupidname.com/",
	}
	for _, url := range urls {
		// Launch a goroutine to fetch the URL.
		url := url // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			// Fetch the URL.
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			client := http.Client{}
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
			return err
		})
	}
	// Wait for all HTTP fetches to complete.
	if err := g.Wait(); err == nil {
		fmt.Println("Successfully fetched all URLs.")
	}
}
