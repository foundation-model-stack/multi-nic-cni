// This is a relevant code fragment from multi-nicd (daemon) in the repo
// for controller testing
package controllers_test

import (
	"fmt"
	"strconv"
	"strings"
)

var DaemonStub = Daemon{}

var SHIFT_BYTE_VAL int64 = 256

type Daemon struct{}

type IPValue struct {
	Address string
	Value   int64
}

func (d Daemon) valueToAddr(value int64) [4]byte {
	var output [4]byte
	for index := 3; index >= 0; index-- {
		output[index] = byte(value % SHIFT_BYTE_VAL)
		value = value / SHIFT_BYTE_VAL
	}
	return output
}

func (d Daemon) valueToAddrStr(value int64) string {
	ip := d.valueToAddr(value)
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}

func (d Daemon) addrToValue(address string) int64 {
	splits := strings.Split(address, ".")
	var sumValue int64
	sumValue = 0
	for _, split := range splits {
		val, _ := strconv.ParseInt(split, 10, 64)
		sumValue = sumValue*SHIFT_BYTE_VAL + val
	}
	return sumValue
}

func (d Daemon) getIPValue(address string) IPValue {
	ip := strings.Split(address, "/")[0]
	return IPValue{Address: address, Value: d.addrToValue(ip)}
}

func (d Daemon) getAddressByIndex(cidr string, index int) string {
	startIPInIpValue := d.getIPValue(cidr)
	addressByIndex := startIPInIpValue.Value + int64(index)
	return d.valueToAddrStr(addressByIndex)
}
