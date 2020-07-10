# NetworkGadget

类似lcx + proxy的小工具
<br/>


<img src="https://i.niupic.com/images/2020/07/06/8mYM.png" alt="" width="833" height="399" data-load="full" style="">


# 实现功能

1.本地端口转发  
2.内网穿刺  
3.tcp代理(socks5)

> ps：可以执行build_tls.sh文件，生成证书加密流量.

<br/>
使用代理时，需要创建'proxy_node'文件.<br/>
格式如下:<br/>
{ip port}<br/>
101.23.16.77 1234



# 编译

```
cd <文件目录>
go build -i -o target/ng NetworkGadget/src/main
```

**ps：不想执行命令的，target目录下有build_linux.bat文件(根据情况修改就行了)**



# 使用说明

参数搭配(一)：

```
ng -<forward>

ng -forward 1234 3389

ng -forward 1235 x.x.x.x:1234

然后连接本地1235端口相当于连接x.x.x.x的3389
```

参数搭配(二)：

```
ng -<listen|connect>

ng -listen 51006 51007

ng -connect x.x.x.x:51007 3389

此时访问x.x.x.x:51006即可连接3389端口

另外也可以: ng -forward 1234 x.x.x.x:51006，转发到本地来连接1234端口
```

参数搭配(三)：

```
ng -<proxy|-local>

ng -proxy 51006

需要在proxy_node中加入代理服务器的地址和端口：x.x.x.x 51006

ng --proxy-local 1234
```

特别说明 ng -foward --tls <1|2|3>

```
ng -forward --tls 1 1234 x.x.x.x:3389	即1234端口收到的数据必须是加密过后的数据且

ng -forward --tls 2 1234 x.x.x.x:3389	即连接到3389端口的数据都进行加密

ng -forward --tls 3 1234 x.x.x.x:3389 	两个端口的连接都进行加密处理
```

# 代理功能的骚操作

```
比如有内网机器A|B，服务器S

那么流量走向：A<—>S<—>B<—>目的服务器

S命令：

	ng -listen 51006 51007	

B命令：

	ng -proxy 51000

	ng -connect x.x.x.x:51007 51000

A命令：

	ng --proxy-local 7891

	配置proxy_node文件
```
