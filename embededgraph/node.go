package embededgraph

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Nodes is map[name][id]*Node
var Nodes map[string]map[string]*Node = make(map[string]map[string]*Node)

// Edges is map[name][(from)name:id][(to)name:id]
var Edges map[string]map[string]map[string]bool = make(map[string]map[string]map[string]bool)

type Node struct {
	Name string
	Id   string
	Data interface{}
}

func UpsertNode(name, id string, data interface{}) (*Node, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Struct {
		return nil, errors.New("data field is not a struct")
	}

	if len(name) == 0 {
		name = t.String()
	}

	if _, ok := Nodes[name]; !ok {
		Nodes[name] = make(map[string]*Node)
	} else if Nodes[name][id] != nil && *Nodes[name][id] != data {
		removeIndex(Nodes[name][id])
	}

	Nodes[name][id] = &Node{name, id, data}
	addIndex(Nodes[name][id])
	return Nodes[name][id], nil
}

func GetNode(name, id string) (n *Node) {
	n, _ = Nodes[name][id]
	return
}

func DeleteNode(name, id string) {
	delete(Nodes[name], id)
}

func (n *Node) Tos(nodeName, nodeId, edgeName string) (tos []*Node) {
	tos = make([]*Node, 0, 10)
	if m, ok := Edges[edgeName][n.key()]; ok {
		for k, ok := range m {
			if ok {
				tos = append(tos, searchNodeByKey(k))
			}
		}
	}
	return
}

func SearchNode(nodeName string, filters map[string]interface{}) (ns []*Node) {
	ns = make([]*Node, 0, 10)

	fieldIndex := index[nodeName]
	if fieldIndex == nil {
		return
	}

	r := make(map[string]bool)
	indexKeys := make([]string, 0, len(filters))

	for fn, fv := range filters {
		indexKeys = append(indexKeys, fmt.Sprintf("%s:%s", fn, fv))
	}
	//fmt.Printf("search keys: %s\n", indexKeys)

	if n, _ := fieldIndex[indexKeys[0]]; n == nil {
		//fmt.Println("empty after first filter")
		return // empty after first filter
	} else {
		for k, v := range n {
			if v {
				r[k] = v
			}
			//fmt.Printf("Adding r[%s]%v\n", k, v)
		}
	}

	for _, k := range indexKeys[1:] {
		if n, _ := fieldIndex[k]; n == nil {
			//fmt.Printf("filter %v returns empty\n", k)
			return // empty result
		} else {
			for x := range r {
				if !n[x] {
					delete(r, x)
					//fmt.Printf("Delete %s\n", x)
				}
			}
		}
	}

	for x := range r {
		//fmt.Printf("Appending %s to result sets\n", x)
		ns = append(ns, searchNodeByKey(x))
	}
	return
}

// helper functions and variables

// index map[node-name][field-name:field-value][node-key]
var index map[string]map[string]map[string]bool = make(map[string]map[string]map[string]bool)

func (n *Node) key() string {
	return fmt.Sprintf("%s:%s", n.Name, n.Id)
}

func searchNodeByKey(key string) *Node {
	tokens := strings.Split(key, ":")
	if len(tokens) != 2 {
		return nil
	}
	return GetNode(tokens[0], tokens[1])
}

func addIndex(n *Node) {
	p := reflect.TypeOf(n.Data)

	fieldIndex := index[n.Name]
	if fieldIndex == nil {
		fieldIndex = make(map[string]map[string]bool)
		index[n.Name] = fieldIndex
	}

	v := reflect.ValueOf(n.Data)
	for i := 0; i < p.NumField(); i++ {
		switch p.Field(i).Type.Kind() {
		case reflect.String, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8, reflect.Float32, reflect.Float64,
			reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8, reflect.Bool:
			fieldName := p.Field(i).Name
			fieldValue := reflect.Indirect(v).FieldByName(fieldName)

			if x, _ := fieldIndex[fmt.Sprintf("%s:%s", fieldName, fieldValue)]; x == nil {
				fieldIndex[fmt.Sprintf("%s:%s", fieldName, fieldValue)] = make(map[string]bool)
			}
			fieldIndex[fmt.Sprintf("%s:%s", fieldName, fieldValue)][n.key()] = true
		}
	}
}

func removeIndex(n *Node) {
	fieldIndex := index[n.Name]
	t := reflect.TypeOf(n.Data)
	v := reflect.ValueOf(n.Data)
	for i := 0; i < t.NumField(); i++ {
		switch t.Field(i).Type.Kind() {
		case reflect.String, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8, reflect.Float32, reflect.Float64,
			reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8, reflect.Bool:
			fieldName := t.Field(i).Name
			fieldValue := reflect.Indirect(v).FieldByName(fieldName)
			//delete(fieldIndex[fmt.Sprintf("%s:%s", fieldName, fieldValue)], n.key())
			fieldIndex[fmt.Sprintf("%s:%s", fieldName, fieldValue)][n.key()] = false
		}
	}
}
