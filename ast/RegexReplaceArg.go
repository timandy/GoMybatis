package ast

import (
	"bytes"
)

//find like #{*} value *
//
//历史 API：返回 str 中所有 #{...} 占位符的 name（出现一次返回一项，不去重）。
//出现 ',' 时取逗号前部分（兼容 #{name, jdbcType=...} 写法）。
//
//当前 NodeString 渲染已迁移到 Token 流（参见 Token.go::tokenize），
//此函数保留作为对外暴露的字符串工具供调用方和测试使用。
func FindExpress(str string) []string {
	var finds = []string{}
	var item []byte
	var lastIndex = -1
	var startIndex = -1
	var strBytes = []byte(str)
	for index, v := range strBytes {
		if v == 35 {
			lastIndex = index
		}
		if v == 123 && lastIndex == (index-1) {
			startIndex = index + 1
		}
		if v == 125 && startIndex != -1 {
			item = strBytes[startIndex:index]

			//去掉逗号之后的部分
			if bytes.Contains(item, []byte(",")) {
				item = bytes.Split(item, []byte(","))[0]
			}
			finds = append(finds, string(item))
			item = nil
			startIndex = -1
			lastIndex = -1
		}
	}
	item = nil
	strBytes = nil

	var strs = []string{}
	for _, k := range finds {
		strs = append(strs, k)
	}
	return strs
}

//find like ${*} value *
//
//历史 API：返回 str 中所有 ${...} 占位符的 name（出现一次返回一项，不去重）。
//出现 ',' 时取逗号前部分。
//
//当前 NodeString 渲染已迁移到 Token 流（参见 Token.go::tokenize），
//此函数保留作为对外暴露的字符串工具供调用方和测试使用。
func FindRawExpressString(str string) []string {
	var finds = []string{}
	var item []byte
	var lastIndex = -1
	var startIndex = -1
	var strBytes = []byte(str)
	for index, v := range str {
		if v == 36 {
			lastIndex = index
		}
		if v == 123 && lastIndex == (index-1) {
			startIndex = index + 1
		}
		if v == 125 && startIndex != -1 {
			item = strBytes[startIndex:index]
			//去掉逗号之后的部分
			if bytes.Contains(item, []byte(",")) {
				item = bytes.Split(item, []byte(","))[0]
			}
			finds = append(finds, string(item))
			item = nil
			startIndex = -1
			lastIndex = -1
		}
	}
	item = nil
	strBytes = nil

	var strs = []string{}
	for _, k := range finds {
		strs = append(strs, k)
	}
	return strs
}
