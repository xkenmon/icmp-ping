package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//ICMP protocol
type ICMP struct {
	Type        uint8    //8
	Code        uint8    //8
	Checksum    uint16   //16
	ID          uint16   //16
	SequenceNum uint16   //16
	Content     [32]byte //32*8		//total :320bit 40byte
}

//CountInfo ICMP ping count info
type CountInfo struct {
	CountPkg  uint8
	CountTime float32
	LossPkg   uint8
	MinTime   float32
	MaxTime   float32
}

var ipPing string
var t int
var timeout int
var interval int

func main() {
	for _, str := range os.Args {
		indexIP := strings.Index(str, "ip=")
		if indexIP != -1 {
			ipPing = str[indexIP+3:]
		}
		indext := strings.Index(str, "t=")
		if indext != -1 {
			t, _ = strconv.Atoi(str[indext+2:])
		}
		indexTT := strings.Index(str, "timeout=")
		if indexTT != -1 {
			timeout, _ = strconv.Atoi(str[indexTT+8:])
		}
		indexIt := strings.Index(str, "interval=")
		if indexIt != -1 {
			interval, _ = strconv.Atoi(str[indexIt+9:])
		}
	}
	if ipPing == "" {
		// fmt.Println("please input ping Addr, like \"ping=XXX.XXX.XXX.XXX\"")
		fmt.Printf("No \"ip\" Args,Exiting\nUsage:\n\tip=\"remote ip\"\n\ttimeout=\"timeout(ms)default 3ms\"\n\tt=\"ping time\"\n")
		// time.Sleep(3e9)
		os.Exit(1)
	}
	if t == 0 {
		t = 5
	}
	if timeout == 0 {
		timeout = 3000
	}
	if interval == 0 {
		interval = 1
	}

	info := PingIP(ipPing)
	printInfo(info)

}

//PingIP ping this ip
func PingIP(ipPing string) CountInfo {
	addr, err := net.ResolveIPAddr("ip", ipPing)
	if err != nil {
		fmt.Println("Resolution error", err.Error())
		os.Exit(1)
	}

	var info CountInfo

	conn, err := net.Dial("ip:icmp", ipPing)
	fmt.Printf("远程地址:%s\n", addr)
	defer conn.Close()
	if err != nil {
		fmt.Printf("网络不可达:%s", err)
		os.Exit(1)
	}

	recv := make([]byte, 40)
	for i := t; i > 0; i-- {
		fmt.Printf("正在 Ping %s 具有 32 字节的数据:\n", ipPing)
		info.CountPkg++
		icmpreq := getIcmpReq(uint16(i), uint16(i))
		icmpbyte := getICMPByte(icmpreq)
		_, err := conn.Write(icmpbyte)
		checkErr(err)
		//设置五秒超时时间
		start := time.Now()
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
		_, err = conn.Read(recv)
		// fmt.Printf("recv pkg: %v\n", recv)
		if err != nil {
			info.LossPkg++
			info.CountTime += float32(timeout)
			fmt.Printf("请求超时:%s\n", err)
			continue
		}
		end := time.Now()
		dur := float32(end.Sub(start).Nanoseconds()) / 1e6 //将纳秒转换为毫秒
		info.CountTime += dur
		if info.MaxTime < dur {
			info.MaxTime = dur
		}
		if info.MinTime == 0 {
			info.MinTime = dur
		}
		if info.MinTime > dur {
			info.MinTime = dur
		}
		fmt.Printf("来自 %s 的回复: 时间 = %.2fms\n", ipPing, dur)
		time.Sleep(time.Duration(interval) * time.Second)
	}
	return info
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func getICMPByte(icmp ICMP) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, icmp)
	return buf.Bytes()
}

func getIcmpReq(id uint16, seq uint16) (icmp ICMP) {
	icmp.Type = 8
	icmp.Code = 0
	icmp.ID = id
	icmp.SequenceNum = seq
	var buf bytes.Buffer
	buf.WriteString("Go Go Guy!!")
	var b [32]byte
	bb := buf.Bytes()
	for i, v := range bb {
		b[i] = v
	}
	icmp.Content = b
	icmp.Checksum = ChecksumICMP(icmp)
	return
}

//ChecksumICMP  checksum for ICMP
func ChecksumICMP(icmp ICMP) uint16 {
	sum := 0
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, icmp)
	msg := buf.Bytes()
	for n := 0; n < len(msg)-1; n += 2 {
		sum += int(msg[n])*256 + int(msg[n+1])
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += (sum >> 16)
	return uint16(^sum)
}

func printInfo(info CountInfo) {
	rcvPkg := info.CountPkg - info.LossPkg
	lossRate := float32(info.LossPkg) / float32(info.CountPkg)
	fmt.Print("\n-------------统计信息--------------\n")
	fmt.Printf("共发送了 %d 个包, %d 个包已接收, 丢包率：%.2f%%, 总时间：%.2fms\n", info.CountPkg, rcvPkg, lossRate*100, info.CountTime)
	fmt.Printf("最大延时：%.2fms 最小延时：%.2fms 平均延时：%.2fms\n", info.MaxTime, info.MinTime, (float32(info.CountTime)-float32(int(info.LossPkg)*timeout))/float32(rcvPkg))
}
