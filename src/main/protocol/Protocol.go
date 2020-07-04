package protocol

import (
	"net"
	"strconv"
)

/*
	Socks协议分为Socks4和Socks5两个版本，他们最明显的区别是Socks5同时支持TCP和UDP两个协议，而Socks4只支持TCP。

	第一次socks5请求信息：
	+---------+----------+-----------+
	|   VER	  | NMETHODS |  METHODS  |
	+---------+----------+-----------+
	|   1	  |     1    |  1 to 255 |
	+---------+----------+-----------+

	VER: Socks的版本，Socks5默认为0x05，其固定长度为1个字节
	NMETHODS: 第三个字段METHODS的长度，它的长度也是1个字节
	METHODS: 客户端支持的验证方式，可以有多种，他的尝试是1-255个字节。



	METHODS目前支持的验证方式:
		X'00' NO AUTHENTICATION REQUIRED（不需要验证）
		X'01' GSSAPI
		X'02' USERNAME/PASSWORD（用户名密码）
		X'03' to X'7F' IANA ASSIGNED
		X'80' to X'FE' RESERVED FOR PRIVATE METHODS
		X'FF' NO ACCEPTABLE METHODS（都不支持，没法连接了）

	服务器返回信息:
	+---------+----------+
	|   VER	  |	 METHOD  |
	+---------+----------+
	|    1	  |	    1    |
	+---------+----------+

	VER: Socks的版本，Socks5默认为0x05，其固定长度为1个字节
	METHOD: 需要服务端需要客户端按照此验证方式提供验证信息，其值长度为1个字节，选择为上面的六种验证方式

	例如：[]byte{0x05, 0x00} 即无需验证

	+---------+---------+---------+---------+------------+------------+
	|   VER	  |	  CMD   |   RSV   |   ATYP  |  DST.ADDR  |  DST.PORT  |
	+---------+---------+---------+---------+------------+------------+
	|    1	  |	   1    |  X'00'  |    1    |  Variable  |      2     |
	+---------+---------+---------+---------+------------+------------+

	VER: Socket的版本，Soket5默认为0x05，其值长度为1个字节
	CMD: 客户端请求的类型，值长度也是1个字节，有三种类型:
		CONNECT X'01'
		BIND X'02'
		UDP ASSOCIATE X'03'

	RSV: 保留字，值长度为1个字节
	ATYP: 请求的远程服务器地址类型，值长度1个字节，有三种类型:
		IP V4 address: X'01'
		DOMAINNAME: X'03'
		IP V6 address: X'04'

	DST.ADDR: 远程服务器的地址，根据ATYP进行解析，值长度不定。
	DST.PORT: 远程服务器的端口，要访问哪个端口的意思，值长度2个字节



	+---------+---------+---------+---------+------------+------------+
	|   VER	  |	  REP   |   RSV   |   ATYP  |  BND.ADDR  |  BND.PORT  |
	+---------+---------+---------+---------+------------+------------+
	|    1	  |	   1    |  X'00'  |    1    |  Variable  |      2     |
	+---------+---------+---------+---------+------------+------------+

	VER: Socket的版本，Soket5默认为0x05，其值长度为1个字节
	REP: 响应状态码，值长度也是1个字节:
		X'00' succeeded
		X'01' general SOCKS server failure
		X'02' connection not allowed by ruleset
		X'03' Network unreachable
		X'04' Host unreachable
		X'05' Connection refused
		X'06' TTL expired
		X'07' Command not supported
		X'08' Address type not supported
		X'09' to X’FF’ unassigned

	RSV: 保留字，值长度为1个字节
	ATYP: 请求的远程服务器地址类型，值长度1个字节，有三种类型:
		IP V4 address: X'01'
		DOMAINNAME: X'03'
		IP V6 address: X'04'

	BND.ADDR: 绑定地址，值长度不定。
	BND.PORT: 绑定端口，值长度2个字节
*/

// 解析socks5协议获取ip和port
func ParsingSocks5GetIpAndPort(buf []byte, len int) (host, port string) {
	switch buf[3] {
	// Ipv4
	case 0x01:
		host = net.IPv4(buf[4], buf[5], buf[6], buf[7]).String()
	// 域名信息
	case 0x03:
		//b[4]表示域名的长度
		host = string(buf[5 : len-2])
	// Ipv6
	case 0x04:
		host = net.IP{buf[4], buf[5], buf[6], buf[7], buf[8], buf[9], buf[10], buf[11], buf[12], buf[13], buf[14], buf[15], buf[16], buf[17], buf[18], buf[19]}.String()
	}

	port = strconv.Itoa(int(buf[len-2])<<8 | int(buf[len-1]))

	return host, port
}
