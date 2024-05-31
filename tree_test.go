package nxjgo

import (
	"strings"
	"testing"
)

func TestTreeNode(t *testing.T) {
	//root := &treeNode{"/", make([]*treeNode, 0), "", false}
	//fmt.Println("[nxjgo] 2024/05/20 - 08:25:44 |  200  |            0s |       127.0.0.1  | GET    \"/nxj/test\"")
	root := &Trie{"/", make([]*Trie, 0), false}

	root.Put("/user/get/:id")
	root.Put("/user/create/*")
	root.Put("/user/test/hello")
	root.Put("/user/test/aaa")
	root.Put("/order/get/aaa")

	node1 := root.Get("/user/get/1")
	t.Logf("%+v", node1)
	node1 = root.Get("/user/create/*")
	t.Logf("%+v", node1)
	node1 = root.Get("/user/test/aaa")
	t.Logf("%+v", node1)
	node1 = root.Get("/order/get/aaa")
	t.Logf("%+v", node1)

}

type Trie struct {
	name     string
	children []*Trie
	isEnd    bool
}

// Put /user/get/1
func (t *Trie) Put(path string) {
	node := t
	strs := strings.Split(path, "/")
	for i, v := range strs {
		if i == 0 {
			continue
		}
		isMatch := false
		for _, ch := range node.children {
			if v == ch.name {
				node = ch
				isMatch = true
				break
			}
		}
		if !isMatch {
			isEnd := false
			if len(strs)-1 == i {
				isEnd = true
			}
			n := &Trie{
				name:     v,
				children: make([]*Trie, 0),
				isEnd:    isEnd,
			}
			node.children = append(node.children, n)
			node = n
		}
	}
}

// Get /user/get/:1
func (t *Trie) Get(path string) *Trie {
	node := t
	strs := strings.Split(path, "/")
	for i, v := range strs {
		if i == 0 {
			continue
		}
		for _, ch := range node.children {
			if ch.name == v || ch.name == "*" || strings.Contains(ch.name, ":") {
				node = ch
				break
			}
		}
		if len(strs)-1 == i {
			return node
		}
	}
	return nil
}
