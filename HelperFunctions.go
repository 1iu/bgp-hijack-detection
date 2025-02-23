package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	Black   = Color("\033[1;30m%s\033[0m")
	Red     = Color("\033[1;31m%s\033[0m")
	Green   = Color("\033[1;32m%s\033[0m")
	Yellow  = Color("\033[1;33m%s\033[0m")
	Purple  = Color("\033[1;34m%s\033[0m")
	Magenta = Color("\033[1;35m%s\033[0m")
	Teal    = Color("\033[1;36m%s\033[0m")
	White   = Color("\033[1;37m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

func arrayToString(a []uint8, delim string) string {
	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
}

func IsContainedInUint32(slice []uint32, val uint32) bool { //a generic method to find out if a specific uint32 is contained in a slice of uint32
	result := false
	for i := 0; i < len(slice); i++ {
		if slice[i] == val {
			result = true
			break
		}
	}
	return result
}

func aspathtoIntSlice(aspath []uint32) []int {
	aspathAsString := make([]int, len(aspath))
	for i := 0; i < len(aspath); i++ {
		aspathAsString[i] = int(aspath[i])
	}
	return aspathAsString
}

func min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

//seen on StackOverflow: https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
func Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
