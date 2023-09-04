package nxjgo

import (
	"strings"
)

type treeNode struct {
	name       string
	children   []*treeNode
	routerName string
	isEnd      bool
}

// Put path: /user/get/:id
func (t *treeNode) Put(path string) {
	root := t
	strs := strings.Split(path, "/")
	for index, name := range strs {
		if index == 0 {
			continue
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			if node.name == name {
				t = node
				isMatch = true
				break
			}
		}
		if !isMatch {
			isEnd := false
			if index == len(strs)-1 {
				isEnd = true
			}
			node := &treeNode{
				name:     name,
				children: make([]*treeNode, 0),
				isEnd:    isEnd,
			}
			children = append(t.children, node)
			t.children = children
			t = node
		}
	}
	t = root
}

// Get path: /user/get/1
func (t *treeNode) Get(path string) *treeNode {
	strs := strings.Split(path, "/")
	routerName := ""
	for index, name := range strs {
		if index == 0 {
			continue
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			if node.name == name || node.name == "*" || strings.Contains(node.name, ":") {
				isMatch = true
				t = node
				routerName += "/" + node.name
				node.routerName = routerName
				if index == len(strs)-1 {
					return node
				}
				break
			}
		}
		if !isMatch {
			for _, node := range children {
				if node.name == "**" {
					routerName += "/" + node.name
					node.routerName = routerName
					return node
				}
			}
			return nil
		}
	}
	return nil
}
