package lru

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type lruSuite struct {
	suite.Suite
}

func TestLRUSuite(t *testing.T) {
	suite.Run(t, new(lruSuite))
}
func (s *lruSuite) TestInsertAtHead() {
	tt := []struct {
		name     string
		addNodes []*Node
		wantLL   []string
	}{
		{
			name: "0. One inserted",
			addNodes: []*Node{
				{
					Key: "key00",
				},
			},
			wantLL: []string{
				"HEAD", "key00", "TAIL",
			},
		},
		{
			name: "1. Some inserted",
			addNodes: []*Node{
				{
					Key: "key00",
				},
				{
					Key: "key01",
				},
				{
					Key: "key02",
				},
			},
			wantLL: []string{
				"HEAD", "key02", "key01", "key00", "TAIL",
			},
		},
	}
	for _, v := range tt {
		s.Run(v.name, func() {
			// 0. Initialize linkedList
			linkedList := NewLRULinkedList()

			// 1. InsertAtHead
			for _, node := range v.addNodes {
				linkedList.InsertAtHead(node)
			}

			// 2. Retreiving linkedList footprint
			gotLL := linkedList.getState()

			// 3. Comparing results
			s.Equal(v.wantLL, gotLL)
		})
	}
}

func (s *lruSuite) TestRemove() {
	tt := []struct {
		name          string
		Nodes         []*Node
		removeNodeIdx int
		wantLL        []string
	}{
		{
			name: "0. Node is present. Single Node",
			Nodes: []*Node{
				{
					Key: "key00",
				},
			},
			removeNodeIdx: 0,
			wantLL: []string{
				"HEAD", "TAIL",
			},
		},
		{
			name: "1. Node is present. Some nodes",
			Nodes: []*Node{
				{
					Key: "key00",
				},
				{
					Key: "key01",
				},
				{
					Key: "key02",
				},
			},
			removeNodeIdx: 1,
			wantLL: []string{
				"HEAD", "key02", "key00", "TAIL",
			},
		},
		{
			name: "2. Empty node",
			Nodes: []*Node{
				{
					Key: "key00",
				},
				{
					Key: "key01",
				},
				{
					Key: "key02",
				},
			},
			removeNodeIdx: -1,
			wantLL: []string{
				"HEAD", "key02", "key01", "key00", "TAIL",
			},
		},
	}
	for _, v := range tt {
		s.Run(v.name, func() {
			// 0. Initialize linkedList
			linkedList := NewLRULinkedList()

			// 1. InsertAtHead
			for _, node := range v.Nodes {
				linkedList.InsertAtHead(node)
			}

			// 2.1 Getting node
			nodeToRemove := &Node{}

			if v.removeNodeIdx >= 0 && v.removeNodeIdx < len(v.Nodes) {
				nodeToRemove = v.Nodes[v.removeNodeIdx]
			}

			// 2.1 Remove
			linkedList.Remove(nodeToRemove)

			// 3. Retreiving linkedList footprint
			gotLL := linkedList.getState()

			// 4. Comparing results
			s.Equal(v.wantLL, gotLL)
		})
	}
}

func (s *lruSuite) TestMoveToHead() {
	tt := []struct {
		name        string
		Nodes       []*Node
		moveNodeIdx int
		wantLL      []string
	}{
		{
			name: "0. Node is present. Single Node",
			Nodes: []*Node{
				{
					Key: "key00",
				},
			},
			moveNodeIdx: 0,
			wantLL: []string{
				"HEAD", "key00", "TAIL",
			},
		},
		{
			name: "1. Node is present. Some nodes",
			Nodes: []*Node{
				{
					Key: "key00",
				},
				{
					Key: "key01",
				},
				{
					Key: "key02",
				},
			},
			moveNodeIdx: 1,
			wantLL: []string{
				"HEAD", "key01", "key02", "key00", "TAIL",
			},
		},
		{
			name: "2. Empty node",
			Nodes: []*Node{
				{
					Key: "key00",
				},
				{
					Key: "key01",
				},
				{
					Key: "key02",
				},
			},
			moveNodeIdx: -1,
			wantLL: []string{
				"HEAD", "key02", "key01", "key00", "TAIL",
			},
		},
	}
	for _, v := range tt {
		s.Run(v.name, func() {
			// 0. Initialize linkedList
			linkedList := NewLRULinkedList()

			// 1. InsertAtHead
			for _, node := range v.Nodes {
				linkedList.InsertAtHead(node)
			}

			// 2.1 Getting node
			nodeToMove := &Node{}

			if v.moveNodeIdx >= 0 && v.moveNodeIdx < len(v.Nodes) {
				nodeToMove = v.Nodes[v.moveNodeIdx]
			}

			// 2.1 Remove
			linkedList.MoveToHead(nodeToMove)

			// 3. Retreiving linkedList footprint
			gotLL := linkedList.getState()

			// 4. Comparing results
			s.Equal(v.wantLL, gotLL)
		})
	}
}

func (l *LinkedList) getState() []string {
	gotLL := []string{"HEAD"}
	LRUNode := l.Head

	for LRUNode.Next.Next != nil {
		LRUNode = LRUNode.Next
		gotLL = append(gotLL, LRUNode.Key)
	}

	gotLL = append(gotLL, "TAIL")

	return gotLL
}
