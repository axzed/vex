// Copyright 2022 Xue WenChao. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// license that can be found in the LICENSE file.

package vex

import "strings"

type treeNode struct {
	name     string
	children []*treeNode
}

// put path: /user/get/:id
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
				isMatch = true
				t = node
				break
			}
		}
		if !isMatch {
			node := &treeNode{name: name, children: make([]*treeNode, 0)}
			children = append(children, node)
			t.children = children
			t = node
		}
	}
	t = root
}

// get path: /user/get/11
func (t *treeNode) Get(path string) *treeNode {
	strs := strings.Split(path, "/")
	for index, name := range strs {
		if index == 0 {
			continue
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			if node.name == name ||
				node.name == "*" ||
				strings.Contains(node.name, ":") {
				isMatch = true
				t = node
				if index == len(strs)-1 {
					return node
				}
				break
			}
		}
		if !isMatch {
			for _, node := range children {
				if node.name == "**" {
					return node
				}
			}
		}
	}
	return nil
}
