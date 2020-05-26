package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"reflect"
)

// Encode 将interface{}转成字符流，不支持可变长度类型
func Encode(in interface{}) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, in)
	if err != nil { 
		log.Println("fail to encode interface:", reflect.TypeOf(in).String(), in)
		panic(err) 
		// return nil
	}
	return buf.Bytes()
}

// Decode 将字符流填充到指定结构体
func Decode(in []byte, out interface{}) int {
	buf := bytes.NewReader(in)
	err := binary.Read(buf, binary.BigEndian, out)
	if err != nil {
		log.Println("fail to decode interface:", in[:20], len(in))
		log.Printf("type:%T\n", out)
		panic(err)
		//return 0
	}
	return len(in) - buf.Len()
}