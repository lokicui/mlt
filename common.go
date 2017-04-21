package main

import (
	"container/list"
	"strings"
)

//全角->半角
func SBC2DBC(s string) string {
	r := []string{}
	for _, i := range s {
		inside_code := i
		if inside_code == 0x3000 {
			inside_code = 0x0020
		} else if inside_code >= 0xff01 && inside_code <= 0xff5e {
			inside_code -= 0xfee0
		}
		r = append(r, string(inside_code))
	}
	return strings.Join(r, "")
}

//半角->全角
func DBC2SBC(s string) string {
	r := []string{}
	for _, i := range s {
		inside_code := i
		if inside_code == 0x20 {
			inside_code = 0x3000
		} else if inside_code >= 0x20 && inside_code <= 0x7e {
			inside_code += 0xfee0
		}
		r = append(r, string(inside_code))
	}
	return strings.Join(r, "")
}

//最长公共字串
func GetLongestSubString(lhs []string, rhs []string) (r []string) {
	maxlen := 0
	maxindex := 0
	dp := make([][]int, len(lhs), len(lhs))
	for i, _ := range dp {
		dp[i] = make([]int, len(rhs), len(rhs))
	}
	for i := 0; i < len(lhs); i++ {
		for j := 0; j < len(rhs); j++ {
			if lhs[i] == rhs[j] {
				if i > 0 && j > 0 {
					dp[i][j] = dp[i-1][j-1] + 1
				} else { // (i == 0 || j == 0) {
					dp[i][j] = 1
				}
				if dp[i][j] > maxlen {
					maxlen = dp[i][j]
					maxindex = i + 1 - maxlen
				}
			}
		}
	}
	return lhs[maxindex : maxindex+maxlen]
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

//最长公共子序列
func GetLongestSubSequence(lhs []string, rhs []string) (r []string) {
	dp := make([][]int, len(lhs)+1, len(lhs)+1)
	for i, _ := range dp {
		dp[i] = make([]int, len(rhs)+1, len(rhs)+1)
	}
	for i := 1; i < len(lhs)+1; i++ {
		for j := 1; j < len(rhs)+1; j++ {
			if lhs[i-1] == rhs[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = Max(dp[i-1][j], dp[i][j-1])
			}
		}
	}
	//回溯打印
	i := len(lhs)
	j := len(rhs)
	L := list.New()
	for i > 0 && j > 0 {
		stub := dp[i][j]
		if stub == 0 {
			break
		}
		if stub == dp[i][j-1] {
			j--
			continue
		}
		if stub == dp[i-1][j] {
			i--
			continue
		}
		if stub == dp[i-1][j-1]+1 {
			L.PushFront(lhs[i-1])
		}
		i--
		j--
	}
	for e := L.Front(); e != nil; e = e.Next() {
		r = append(r, e.Value.(string))
	}
	return r
}
