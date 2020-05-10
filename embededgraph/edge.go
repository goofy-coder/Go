package embededgraph

import "errors"

type Edge struct {
	Name string
	From *Node
	To   *Node
}

func AddEdge(name string, from, to *Node) error {
	if n := GetNode(from.Name, from.Id); n == nil {
		return errors.New("from node does not exist")
	}

	if n := GetNode(to.Name, to.Id); n == nil {
		return errors.New("to node does not exist")
	}

	if _, ok := Edges[name]; !ok {
		Edges[name] = make(map[string]map[string]bool)
	}

	if _, ok := Edges[name][from.key()]; !ok {
		Edges[name][from.key()] = make(map[string]bool)
	}

	Edges[name][from.key()][to.key()] = true
	return nil
}

func DeleteEdge(name string, from, to *Node) {
	if v, ok := Edges[name][from.key()][to.key()]; !ok || !v {
		// e does not exit or alreadt (soft) deleted ; nothing to do
	}

	Edges[name][from.key()][to.key()] = false
}

func Exists(name string, from, to *Node) bool {
	v, ok := Edges[name][from.key()][to.key()]
	return ok && v
}
