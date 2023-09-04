package nxjgo

import (
	"testing"
)

func TestTreeNode(t *testing.T) {
	root := &treeNode{"/", make([]*treeNode, 0), "", false}

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
