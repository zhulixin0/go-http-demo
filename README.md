## 一个真正意义上的http并发

> 当我们得知http协议包后，便可以在tcp层面做手脚。
> 
> 在服务器接收到我们发送完毕的指令前，把数据截断置最后的一个字符，然后发送出去。
> 
> 这时我们只需要往tcp通道中放入最后一个字符，服务器即开始处理（其他协议同理）
>
* server client
```
// nginx config by openrestry
location /test {
	content_by_lua_block {
		ngx.header.content_type = "text/html"
		ngx.say(ngx.now())
	}
}
```
* test
* ![](/test.gif)