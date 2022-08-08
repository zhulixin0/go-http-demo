package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

type RequestHijack struct {
	packet     []byte
	OverPacket []byte
}

func (r *RequestHijack) Write(p []byte) (n int, err error) {
	r.packet = p[:len(p)-1]
	return 0, nil
}

func (r *RequestHijack) GetMainPacket() []byte {
	return r.packet
}

var overPacket = []byte{10}

func (r *RequestHijack) GetOverPacket() []byte {
	return overPacket
}

func readResponse(c net.Conn) *http.Response {
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		fmt.Println("ReadResponse error!")
		return nil
	}
	return resp
}

func getConn(u *url.URL) (net.Conn, error) {
	c, err := net.DialTimeout("tcp", u.Hostname()+":"+u.Scheme, 5*time.Second)
	if err != nil {
		fmt.Println("time out")
		return nil, err
	}

	if u.Scheme == "https" {
		c = tls.Client(c, &tls.Config{InsecureSkipVerify: true})
	}
	return c, nil
}

func main() {

	// init
	count := 500
	over := &sync.WaitGroup{}
	u, _ := url.Parse("https://hotagr.xyz/test")
	req, _ := http.NewRequest("GET", u.String(), nil)
	result := make([]string, count)

	fmt.Println("start")
	// run
	letsGo(over, count, u, req, result)

	over.Wait()
	// destroy
	table := make(map[string]int)
	sort.Slice(result, func(i, j int) bool {
		if val, ok := table[result[i]]; !ok {
			table[result[i]] = 1
		} else {
			table[result[i]] = val + 1
		}
		return result[i] < result[j]
	})
	fmt.Println(result)
	fmt.Println(table)
	fmt.Println("done")
}

func letsGo(over *sync.WaitGroup, count int, u *url.URL, req *http.Request, result []string) {

	// [scheme:][//[userinfo@]host][/]path[?query][#fragment]
	// =======
	buf := &RequestHijack{}
	_ = req.Write(buf)
	// =======

	// atomic lock
	w := &sync.WaitGroup{}

	for i := 0; i < count; i++ {
		// tcp不能等太久了，容易关闭
		// TODO 在并发请求前，可以先测试一下tcp空连接多久断开
		// TODO 如果节省资源情况下 可考虑一个goroutine 管理多个tcp
		w.Add(1)
		over.Add(1)
		idx := i
		go func() {
			_ = doHttp(w, u, buf, result, idx)
			over.Done()
		}()

	}
}

func doHttp(w *sync.WaitGroup, u *url.URL, buf *RequestHijack, result []string, idx int) error {
	c, err := func() (net.Conn, error) {
		defer w.Done()
		c, err := getConn(u)
		if err != nil {
			return nil, err
		}

		// 写入数据
		if _, err = c.Write(buf.GetMainPacket()); err != nil {
			return nil, err
		}
		return c, nil
	}()
	defer func(c net.Conn) { _ = c.Close() }(c)
	if err != nil {
		return err
	}

	w.Wait()
	// 并发写
	_, err = c.Write(buf.GetOverPacket())
	if err != nil {
		return err
	}

	resp := readResponse(c)
	defer func() { _ = resp.Body.Close() }()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	result[idx] = string(data[:len(data)-1])
	return nil
}
