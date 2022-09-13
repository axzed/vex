// Copyright 2022 Xue WenChao. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// license that can be found in the LICENSE file.

package vex

import "strings"

type treeNode struct {
	name       string
	children   []*treeNode
	routerName string
}

// put path: /user/get/:id
func (t *treeNode) Put(path string) {
	root := t
	strs := strings.Split(path, "/")
	for index, name := range strs {
		// ignore the prefix "_" before the first "/"
		if index == 0 {
			continue
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			// if match the url get to the next node
			if node.name == name {
				isMatch = true
				t = node
				break
			}
		}
		// if not match generate the node to save the name of url
		if !isMatch {
			node := &treeNode{name: name, children: make([]*treeNode, 0)}
			children = append(children, node)
			t.children = children
			t = node
		}
	}
	// back to root
	t = root
}

// get path: /user/get/11
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
			if node.name == name ||
				node.name == "*" ||
				strings.Contains(node.name, ":") {
				isMatch = true
				routerName += "/" + node.name
				node.routerName = routerName
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
