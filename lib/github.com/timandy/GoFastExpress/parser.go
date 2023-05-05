package GoFastExpress

import (
	"errors"
	"go/scanner"
	"go/token"
	"strconv"
	"strings"
)

// Operator 操作符
type Operator = string

const (
	//一元操作符
	Size Operator = "size()"

	//计算操作符
	Add    Operator = "+"
	Reduce Operator = "-"
	Ride   Operator = "*"
	Divide Operator = "/"

	//比较操作符
	And       Operator = "&&"
	Or        Operator = "||"
	Equal     Operator = "=="
	UnEqual   Operator = "!="
	Less      Operator = "<"
	LessEqual Operator = "<="
	More      Operator = ">"
	MoreEqual Operator = ">="

	//常量
	Nil  Operator = "nil"
	Null Operator = "null"
)

var (
	ConstOperators  = []Operator{Nil, Null}
	UnaryOperators  = []Operator{Size}
	BinaryOperators = []Operator{Add, Reduce, Ride, Divide, And, Or, Equal, UnEqual, Less, LessEqual, More, MoreEqual}
)

//是否常量运算符
func isConstOperator(o Operator) bool {
	return contains(ConstOperators, o)
}

//是否一元运算符
func isUnaryOperator(o Operator) bool {
	return contains(UnaryOperators, o)
}

//是否二元运算符
func isBinaryOperator(o Operator) bool {
	return contains(BinaryOperators, o)
}

//乘除优先于加减 计算优于比较,
var priorityArray = []Operator{Size, Ride, Divide, Add, Reduce,
	LessEqual, Less, MoreEqual, More,
	UnEqual, Equal, And, Or}

var NotSupportOptMap = map[string]bool{
	"=": true,
	"!": true,
	"@": true,
	"#": true,
	"$": true,
	"^": true,
	"&": true,
	"(": true,
	")": true,
	"`": true,
}

//操作符优先级
var priorityMap = map[Operator]int{}

func init() {
	for k, v := range priorityArray {
		priorityMap[v] = k
	}
}

func Parser(express string) (Node, error) {
	var opts = ParserOperators(express)
	var nodes []Node
	for _, v := range opts {
		var node, err = parserNode(express, v)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	//check epress
	var err = checkeNodes(express, nodes)
	if err != nil {
		return nil, err
	}
	for _, v := range priorityArray {
		var e = findReplaceOpt(express, v, &nodes)
		if e != nil {
			return nil, e
		}
	}
	if len(nodes) == 0 || nodes[0] == nil {
		return nil, errors.New("parser node fail!")
	}
	return nodes[0], nil
}

func checkeNodes(express string, nodes []Node) error {
	var nodesLen = len(nodes)
	for nIndex, n := range nodes {
		if n.Type() == NOpt {
			var after Node
			var befor Node

			if nIndex > 0 {
				befor = nodes[nIndex-1]
			}
			if nIndex < (nodesLen - 1) {
				after = nodes[nIndex+1]
			}
			if after != nil && after.Type() == NOpt {
				return errors.New("express have more than 2 opt!express=" + express)
			}
			if befor != nil && befor.Type() == NOpt {
				return errors.New("express have more than 2 opt!express=" + express)
			}
		}
	}
	return nil
}

func parserNode(express string, v Operator) (Node, error) {
	if v == Nil || v == Null {
		var inode = NilNode{
			t: NNil,
		}
		return inode, nil
	}
	if NotSupportOptMap[v] {
		return nil, errors.New("find not support opt = '" + v + "',express=" + express)
	}
	if isBinaryOperator(v) {
		var optNode = OptNode{
			value: v,
			t:     NOpt,
		}
		return optNode, nil
	}
	if v == "true" || v == "false" {
		b, e := strconv.ParseBool(v)
		if e == nil {
			var inode = BoolNode{
				value: b,
				t:     NBool,
			}
			return inode, nil
		}
	}
	if strings.Index(v, "'") == 0 && strings.LastIndex(v, "'") == (len(v)-1) {
		var inode = StringNode{
			value: string([]byte(v)[1 : len(v)-1]),
			t:     NString,
		}
		return inode, nil
	}
	if strings.Index(v, "\"") == 0 && strings.LastIndex(v, "\"") == (len(v)-1) {
		var inode = StringNode{
			value: string([]byte(v)[1 : len(v)-1]),
			t:     NString,
		}
		return inode, nil
	}
	if strings.Index(v, "`") == 0 && strings.LastIndex(v, "`") == (len(v)-1) {
		var inode = StringNode{
			value: string([]byte(v)[1 : len(v)-1]),
			t:     NString,
		}
		return inode, nil
	}
	i, e := strconv.ParseInt(v, 0, 64)
	if e == nil {
		var inode = IntNode{
			express: v,
			value:   int64(i),
			t:       NInt,
		}
		return inode, nil
	}
	u, _ := strconv.ParseUint(v, 0, 64)
	if e == nil {
		var inode = UIntNode{
			express: v,
			value:   u,
			t:       NUInt,
		}
		return inode, nil
	}
	f, e := strconv.ParseFloat(v, 64)
	if e == nil {
		var inode = FloatNode{
			express: v,
			value:   f,
			t:       NFloat,
		}
		return inode, nil
	}
	e = nil

	var values = trimRemoveEmptyEntries(strings.Split(v, "."))
	valuesLen := len(values)
	if valuesLen == 0 {
		return nil, errors.New("no values found")
	}
	var argNode = ArgNode{
		value:     v,
		values:    values,
		valuesLen: valuesLen,
		t:         NArg,
	}
	return argNode, nil
}

func trimRemoveEmptyEntries(values []string) []string {
	curIndex := 0
	for _, value := range values {
		value = strings.TrimSpace(value)
		if len(value) == 0 {
			continue
		}
		values[curIndex] = value
		curIndex++
	}
	return values[0:curIndex]
}

func buildUnaryNode(values []string) Node {
	var argNode = ArgNode{
		value:     values[0],
		values:    values[0:1],
		valuesLen: 1,
		t:         NArg,
	}
	var lastNode Node
	lastNode = argNode
	for i := 1; i < len(values); i++ {
		var biNode = UnaryNode{
			node: lastNode,
			opt:  values[i],
			t:    NUnary,
		}
		lastNode = biNode
	}
	return lastNode
}

func findReplaceOpt(express string, operator Operator, nodearg *[]Node) error {
	var nodes = *nodearg
	for nIndex, n := range nodes {
		if n.Type() == NOpt {
			var opt = n.(OptNode)
			if opt.value != operator {
				continue
			}
			var newNode = BinaryNode{
				left:  nodes[nIndex-1],
				right: nodes[nIndex+1],
				opt:   opt.value,
				t:     NBinary,
			}
			var newNodes []Node
			newNodes = append(nodes[:nIndex-1], newNode)
			newNodes = append(newNodes, nodes[nIndex+2:]...)

			if haveOpt(newNodes) {
				findReplaceOpt(express, operator, &newNodes)
			}
			*nodearg = newNodes
			break
		}
	}

	return nil
}

func haveOpt(nodes []Node) bool {
	for _, v := range nodes {
		if v.Type() == NOpt {
			return true
		}
	}
	return false
}

func ParserOperators(express string) []Operator {
	var newResult []Operator
	src := []byte(express)
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	s.Init(file, src, nil, 0)
	var lastToken token.Token
	var index = 0
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF || lit == "\n" {
			break
		}
		//fmt.Printf("%-6s%-8s%q\n", fset.Position(pos), tok, lit)
		var s = toStr(lit, tok)
		if lit == "" && tok != token.ILLEGAL {
			lastToken = tok
		}
		//当前 tok 为 ')' 上一个 lit 必须是 '('
		if tok == token.RPAREN {
			resultLen := len(newResult)
			if resultLen >= 2 {
				lastLit := newResult[resultLen-1]
				if lastLit == "(" {
					newResult = newResult[:resultLen-1]
					newResult[resultLen-2] = newResult[resultLen-2] + "()"
					if index > 0 {
						index -= 1
					}
					continue
				} else {
					panic("[express] '()' must be in pair")
				}
			} else {
				panic("[express] should not start with '()'")
			}
		}
		//
		if tok == token.PERIOD || lastToken == token.PERIOD {
			//append to last token
			newResult[len(newResult)-1] = newResult[len(newResult)-1] + s
			continue
		}

		if index >= 1 && isNumber(s) && newResult[index-1] == "-" {
			if index == 1 {
				newResult = []string{}
				s = "-" + s
				index -= 1
			} else {
				if index > 2 && isBinaryOperator(newResult[index-2]) {
					newResult = newResult[:2]
					s = "-" + s
					index -= 1
				}
			}
		}

		newResult = append(newResult, s)
		index += 1
	}
	return newResult
}

func isNumber(s string) bool {
	var o0 = rune([]byte("0")[0])
	var o1 = rune([]byte("1")[0])
	var o2 = rune([]byte("2")[0])
	var o3 = rune([]byte("3")[0])
	var o4 = rune([]byte("4")[0])
	var o5 = rune([]byte("5")[0])
	var o6 = rune([]byte("6")[0])
	var o7 = rune([]byte("7")[0])
	var o8 = rune([]byte("8")[0])
	var o9 = rune([]byte("9")[0])
	var o10 = rune([]byte("9")[0])
	var o11 = rune([]byte(".")[0])
	for _, v := range s {
		if o0 != v &&
			o1 != v &&
			o2 != v &&
			o3 != v &&
			o4 != v &&
			o5 != v &&
			o6 != v &&
			o7 != v &&
			o8 != v &&
			o9 != v &&
			o10 != v &&
			o11 != v {
			return false
		}
	}
	return true
}

func toStr(lit string, tok token.Token) string {
	if lit == "" {
		return tok.String()
	} else {
		return lit
	}
}

func contains[T comparable](source []T, item T) bool {
	for _, t := range source {
		if t == item {
			return true
		}
	}
	return false
}
