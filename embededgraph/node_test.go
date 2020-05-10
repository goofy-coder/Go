package embededgraph

import (
	"fmt"
	"testing"
)

type Employee struct {
	Title      string
	Department string
	IsManager  bool
}

func initTest() {
	Nodes = make(map[string]map[string]*Node)
	Edges = make(map[string]map[string]map[string]bool)
	index = make(map[string]map[string]map[string]bool)
}
func TestAddInvalidData(t *testing.T) {
	initTest()
	if _, err := UpsertNode("Fool", "1", 1); err == nil {
		t.Error("Adding non-struct node data did not fail")
	}

	if len(Nodes) != 0 {
		t.Errorf("Number of nodes not equal to 0. Actual number is %d", len(Nodes))
	}
}

func TestAddNodeWOName(t *testing.T) {
	initTest()
	n, err := UpsertNode("", "1", Employee{})
	if err != nil {
		t.Error("Adding node without name failed")
	}

	if len(Nodes) != 1 {
		t.Errorf("Number of nodes not equal to 1. Actual number is %d", len(Nodes))
	}

	if Nodes["embededgraph.Employee"]["1"] != n {
		t.Error("Added node is not equal to the original value")
	}
}

func TestAddNodeIdempotency(t *testing.T) {
	initTest()
	n1, _ := UpsertNode("Employee", "123", Employee{Title: "Senior Manager", IsManager: true})
	n2, _ := UpsertNode("Employee", "123", Employee{Title: "Senior Manager", IsManager: true})
	if *n1 != *n2 {
		t.Error("Idempotency broke")
	}

	if len(Nodes) != 1 {
		t.Errorf("Number of nodes not equal to 1. Actual number is %d", len(Nodes))
	}

	if *GetNode("Employee", "123") != *n1 {
		t.Error("Added node is not equal to the original value")
	}
}

func TestDeleteNodeIdempotency(t *testing.T) {
	initTest()

	n, _ := UpsertNode("Employee", "123", Employee{Title: "Senior Manager", IsManager: true})
	DeleteNode(n.Name, n.Id)
	DeleteNode(n.Name, n.Id)

	if GetNode(n.Name, n.Id) != nil {
		t.Errorf("Node did not get deleted properly")
	}
}

func TestAddEdge(t *testing.T) {
	initTest()
	n1, _ := UpsertNode("Employee", "123", Employee{Title: "Senior Manager", IsManager: true})
	n2, _ := UpsertNode("Employee", "456", Employee{Title: "Senior Engineer", IsManager: false})
	AddEdge("manage", n1, n2)

	if len(Edges) != 1 {
		t.Errorf("Expect 1 edge(s) but returned %d", len(Edges))
	}

	if !Exists("manage", n1, n2) {
		t.Error("Expected edge does not exist")
	}
}

func TestAddEdgeWithoutNodes(t *testing.T) {
	initTest()
	n1 := Node{Name: "Employee", Id: "123", Data: Employee{Title: "Senior Manager", IsManager: true}}
	n2 := Node{Name: "Employee", Id: "456", Data: Employee{Title: "Senior Engineer", IsManager: false}}
	if err := AddEdge("manage", &n1, &n2); err == nil {
		t.Error("non existent \"from\" node did not yield error")
	}
}

func TestDeleteEdge(t *testing.T) {
	initTest()
	n1, _ := UpsertNode("Employee", "123", Employee{Title: "Senior Manager", IsManager: true})
	n2, _ := UpsertNode("Employee", "456", Employee{Title: "Senior Engineer", IsManager: false})
	AddEdge("manage", n1, n2)

	DeleteEdge("manage", n1, n2)

	if Exists("manage", n1, n2) {
		t.Error("Edge did not get deleted")
	}

	if GetNode(n1.Name, n1.Id) == nil {
		t.Error("From node not found")
	}

	if GetNode(n2.Name, n2.Id) == nil {
		t.Error("To node not found")
	}
}

func TestSearch(t *testing.T) {
	initTest()

	UpsertNode("Employee", "123", Employee{Title: "Senior Manager", Department: "A-12", IsManager: true})
	UpsertNode("Employee", "124", Employee{Title: "Senior Manager", Department: "R-17", IsManager: true})
	filters := map[string]interface{}{"Title": "Senior Manager"}
	if r := SearchNode("Employee", filters); len(r) != 2 {
		t.Errorf("Expect 2 nodes but received %d", len(r))
	}

	filters = map[string]interface{}{"Title": "Senior Manager", "Department": "A-12"}
	if r := SearchNode("Employee", filters); len(r) != 1 {
		t.Errorf("Expect 1 nodes but received %d", len(r))
	}

	filters = map[string]interface{}{"Title": "Senior Manager", "Department": "A-13"}
	if r := SearchNode("Employee", filters); len(r) != 0 {
		t.Errorf("Expect 0 nodes but received %d", len(r))
	}

	filters = map[string]interface{}{"IsManager": true}
	if r := SearchNode("Employee", filters); len(r) != 2 {
		t.Errorf("Expect 2 nodes but received %d", len(r))
	}

	UpsertNode("", "125", Employee{Title: "Senior Developer", Department: "R-17", IsManager: false})
	//PrintIndexes()

	filters = map[string]interface{}{"IsManager": false}
	if r := SearchNode("embededgraph.Employee", filters); len(r) != 1 {
		for _, x := range r {
			fmt.Println(x)
		}
		t.Errorf("Expect 1 nodes but received %d", len(r))
	}
}

func TestUpdate(t *testing.T) {
	initTest()
	UpsertNode("Employee", "234", Employee{Title: "Senior Engineer", IsManager: false, Department: "Platform"})
	n1, _ := UpsertNode("Employee", "234", Employee{Title: "Manager", IsManager: true, Department: "Platform"})

	n2 := GetNode("Employee", "234")
	if n1 != n2 {
		t.Error("Update failed")
	}

	ns := SearchNode("Employee", map[string]interface{}{"Title": "Manager"})
	if len(ns) != 1 || ns[0] != n2 {
		t.Errorf("Search result is not correct")
	}

	ns = SearchNode("Employee", map[string]interface{}{"Title": "Senior Engineer"})
	if len(ns) != 0 {
		t.Error(*ns[0])
		t.Errorf("Updated node not found as expected (found %d)", len(ns))
	}
}

func PrintNodes() {
	for name := range Nodes {
		for _, n := range Nodes[name] {
			fmt.Println(n)
		}
	}
}

func PrintIndexes() {
	//fmt.Println(index)
	for name, _ := range index {
		fmt.Println(name)
	}

}
