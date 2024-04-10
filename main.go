package main

import (
	"fmt"
	"math"
	"os"
	"slices"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
)

// usage: go run <json file>
func main() {
	resp := &tempopb.SearchResponse{}
	buff, err := os.ReadFile(os.Args[1])
	bailOnError(err)

	err = jsonpb.UnmarshalString(string(buff), resp)
	bailOnError(err)

	tree := treeNode{
		name:  "root",
		left:  math.MinInt64,
		right: math.MaxInt64,
	}

	for _, trace := range resp.Traces {
		if len(trace.SpanSets) != 1 {
			bailOnError(fmt.Errorf("there should be only 1 spanset!"))
		}

		ss := trace.SpanSets[0]
		// sort by nestedSetLeft
		slices.SortFunc(ss.Spans, func(s1, s2 *tempopb.Span) int {
			return int(nestedSetLeft(s1)) - int(nestedSetLeft(s2))
		})

		// reset curNode to root each loop to re-overlay the next trace
		curNode := &tree
		// left/right is only valid w/i a trace, so reset it each loop
		resetLeftRight(&tree)
		for _, span := range ss.Spans {
			// walk up the tree until we find a node that is a parent of this span
			for curNode.parent != nil {
				if curNode.isChild(span) {
					break
				}
				curNode = curNode.parent
			}

			// is there an already existing child that matches the span?
			if child := curNode.findMatchingChild(span); child != nil {
				child.addSpan(span)
				// to the next span!
				continue
			}

			// if not, create a new child node and make it the cur node
			newNode := node(span)
			curNode.addChild(newNode)
			curNode = newNode
		}
	}

	dumpTree(&tree, 0)
}

type treeNode struct {
	name  string
	spans []*tempopb.Span

	left  int64
	right int64

	children []*treeNode
	parent   *treeNode
}

func (t *treeNode) addSpan(s *tempopb.Span) {
	// expand our left/right based on this span
	t.left = min(nestedSetLeft(s), t.left)
	t.right = max(nestedSetRight(s), t.right)
	t.spans = append(t.spans, s)
}

func (t *treeNode) addChild(n *treeNode) {
	n.parent = t
	t.children = append(t.children, n)
}

func (t *treeNode) isChild(s *tempopb.Span) bool {
	return nestedSetLeft(s) > t.left && nestedSetRight(s) < t.right
}

func (t *treeNode) findMatchingChild(s *tempopb.Span) *treeNode {
	name := nodeName(s)

	for _, c := range t.children {
		if c.name == name {
			return c
		}
	}

	return nil
}

func dumpTree(t *treeNode, depth int) {
	for i := 0; i < depth; i++ {
		fmt.Print("  ")
	}

	fmt.Println(t.name, len(t.spans))

	for _, c := range t.children {
		dumpTree(c, depth+1)
	}
}

func resetLeftRight(t *treeNode) {
	t.left = math.MaxInt64
	t.right = math.MinInt64

	for _, c := range t.children {
		resetLeftRight(c)
	}
}

func node(s *tempopb.Span) *treeNode {
	return &treeNode{
		left:  nestedSetLeft(s),
		right: nestedSetRight(s),
		name:  nodeName(s),
		spans: []*tempopb.Span{s},
	}
}

func nodeName(s *tempopb.Span) string {
	var svcName string
	for _, a := range s.Attributes {
		if a.Key == "service.name" {
			svcName = a.Value.GetStringValue()
		}
	}

	return svcName + ":" + s.Name
}

func nestedSetLeft(span *tempopb.Span) int64 {
	for _, a := range span.Attributes {
		if a.Key == "nestedSetLeft" {
			return a.Value.GetIntValue()
		}
	}

	bailOnError(fmt.Errorf("nestedSetLeft not found!"))
	return 0
}

func nestedSetRight(span *tempopb.Span) int64 {
	for _, a := range span.Attributes {
		if a.Key == "nestedSetRight" {
			return a.Value.GetIntValue()
		}
	}

	bailOnError(fmt.Errorf("nestedSetRight not found!"))
	return 0
}

func bailOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
