// Copyright 2019 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/token"
)

const keyDelim = "/"

func mapToNode(m map[string]interface{}, keyDelim string) ast.Node {
	lis := &ast.ObjectList{Items: make([]*ast.ObjectItem, 0, len(m))}
	keys := make(map[string]*ast.ObjectList, len(m))
	for k, v := range m {
		path := strings.Split(k, keyDelim)
		parent := lis
		for i := range path {
			subPath := strings.Join(path[:i+1], keyDelim)
			if i != 0 {
				parent = keys[subPath]
			}
			if i == len(path)-1 { // the last
				o := astObjectItem(path[i], v)
				if i == 0 {
					lis.Add(o)
				} else if parent != nil {
					parent.Add(o)
				}
			} else {
				lis := &ast.ObjectList{}
				o := &ast.ObjectType{List: lis}
				keys[subPath] = lis
				parent.Add(&ast.ObjectItem{
					Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.STRING, Text: k}}},
					Val:  o,
				})
			}
		}
	}
	return lis
}

var _ = nodeToMap

func nodeToMap(node ast.Node, keyDelim string) map[string]interface{} {
	_ = keyDelim
	m := make(map[string]interface{})
	switch x := node.(type) {
	case *ast.ObjectItem:
		m[x.Keys[0].Token.Text] = x.Val
	case *ast.ObjectList:
	}
	return m
}
func astObjectItem(k string, v interface{}) *ast.ObjectItem {
	return &ast.ObjectItem{
		Keys: []*ast.ObjectKey{{Token: token.Token{Type: token.STRING, Text: k}}},
		Val:  astV(v),
	}
}

func astV(v interface{}) ast.Node {
	switch x := v.(type) {
	case string:
		return &ast.LiteralType{Token: astToken(x)}
	case bool:
		return &ast.LiteralType{Token: astToken(x)}
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
		return &ast.LiteralType{Token: astToken(v)}
	case float32, float64:
		return &ast.LiteralType{Token: astToken(v)}
	case []string:
		lis := make([]ast.Node, len(x))
		for i, s := range x {
			lis[i] = astV(s)
		}
		return &ast.ListType{List: lis}
	case []bool:
		lis := make([]ast.Node, len(x))
		for i, s := range x {
			lis[i] = astV(s)
		}
		return &ast.ListType{List: lis}
	case []int:
		lis := make([]ast.Node, len(x))
		for i, s := range x {
			lis[i] = astV(s)
		}
		return &ast.ListType{List: lis}
	case []float64:
		lis := make([]ast.Node, len(x))
		for i, s := range x {
			lis[i] = astV(s)
		}
		return &ast.ListType{List: lis}
	default:
		panic(fmt.Sprintf("unsupported type %[1]T %[1]v", v))
	}
}

func astToken(v interface{}) token.Token {
	switch x := v.(type) {
	case string:
		return token.Token{Type: token.STRING, Text: x}
	case bool:
		return token.Token{Type: token.BOOL, Text: fmt.Sprintf("%t", x)}
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
		return token.Token{Type: token.NUMBER, Text: fmt.Sprintf("%d", v)}
	case float32, float64:
		return token.Token{Type: token.FLOAT, Text: fmt.Sprintf("%f", v)}
	}
	panic(fmt.Sprintf("unsupported type %[1]T %[1]v", v))
}
