package lru

type Node struct {
	Key  string
	Next *Node
	Prev *Node
}

type LinkedList struct {
	Head *Node
	Tail *Node
}

func NewLRULinkedList() LinkedList {
	head, tail := &Node{}, &Node{}
	head.Next = tail
	tail.Prev = head

	return LinkedList{
		Head: head,
		Tail: tail,
	}
}

func (l *LinkedList) Remove(node *Node) {
	if node.Prev == nil || node.Next == nil {

		return
	}

	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev
}

func (l *LinkedList) InsertAtHead(node *Node) {
	node.Prev = l.Head
	node.Next = l.Head.Next

	l.Head.Next.Prev = node
	l.Head.Next = node
}

func (l *LinkedList) MoveToHead(node *Node) {
	if node.Prev == nil || node.Next == nil {

		return
	}
	l.Remove(node)
	l.InsertAtHead(node)
}

func (l *LinkedList) RemoveTail() string {
	tailPrevKey := l.Tail.Prev.Key

	l.Remove(l.Tail.Prev)

	return tailPrevKey
}
