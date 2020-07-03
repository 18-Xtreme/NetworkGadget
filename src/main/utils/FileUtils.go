package utils

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
)

// 一次读取所有内容（存入内存），适合小文件读取
func ReadAll(filePth string) ([]byte, error) {
	f, err := os.Open(filePth)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}

// 分块读取 根据传入的字节大小读取文件
func ReadBlock(filePth string, bufSize int, callback func([]byte)) error {
	f, err := os.Open(filePth)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, bufSize) //一次读取多少个字节
	reader := bufio.NewReader(f)
	for {
		n, err := reader.Read(buf)
		callback(buf[:n]) // n 是成功读取字节数

		if err != nil { //遇到任何错误立即返回，并忽略 EOF 错误信息
			if err == io.EOF {
				return nil
			}
			return err
		}
	}

	return nil
}

// 逐行读取 性能可能慢一些，但是仅占用极少的内存空间。
func ReadLine(filePth string, callback func([]byte)) error {
	f, err := os.Open(filePth)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadBytes('\n')
		//放在错误处理前面，即使发生错误，也会处理已经读取到的数据。
		callback(line)

		//遇到任何错误立即返回，并忽略 EOF 错误信息
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}
