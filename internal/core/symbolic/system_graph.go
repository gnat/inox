package internal

import "errors"

var (
	ANY_SYSTEM_GRAPH            = NewSystemGraph()
	ANY_SYSTEM_GRAPH_NODES      = NewSystemGraphNodes()
	ANY_SYSTEM_GRAPH_NODE       = NewSystemGraphNode()
	SYSTEM_GRAPH_PROPNAMES      = []string{"nodes"}
	SYSTEM_GRAPH_NODE_PROPNAMES = []string{"name", "type_name"}

	_ = []Iterable{(*SystemGraphNodes)(nil)}
	_ = []PotentiallySharable{(*SystemGraph)(nil), (*SystemGraphNodes)(nil), (*SystemGraphNode)(nil)}
)

// An SystemGraph represents a symbolic SystemGraph.
type SystemGraph struct {
	_ int
}

func NewSystemGraph() *SystemGraph {
	return &SystemGraph{}
}

func (g *SystemGraph) Test(v SymbolicValue) bool {
	other, ok := v.(*SystemGraph)
	if ok {
		return true
	}
	_ = other
	return false
}

func (g *SystemGraph) Prop(memberName string) SymbolicValue {
	switch memberName {
	case "nodes":
		return ANY_SYSTEM_GRAPH_NODES
	}
	panic(FormatErrPropertyDoesNotExist(memberName, g))
}

func (g *SystemGraph) SetProp(name string, value SymbolicValue) (IProps, error) {
	return nil, errors.New(FmtCannotAssignPropertyOf(g))
}

func (g *SystemGraph) WithExistingPropReplaced(name string, value SymbolicValue) (IProps, error) {
	return nil, errors.New(FmtCannotAssignPropertyOf(g))
}

func (g *SystemGraph) PropertyNames() []string {
	return SYSTEM_GRAPH_PROPNAMES
}

func (g *SystemGraph) IsSharable() bool {
	return true
}

func (g *SystemGraph) Share(originState *State) PotentiallySharable {
	return g
}

func (g *SystemGraph) IsShared() bool {
	return true
}

func (g *SystemGraph) Widen() (SymbolicValue, bool) {
	return nil, false
}

func (g *SystemGraph) IsWidenable() bool {
	return false
}

func (g *SystemGraph) String() string {
	return "system-graph"
}

func (g *SystemGraph) WidestOfType() SymbolicValue {
	return ANY_SYSTEM_GRAPH
}

// An SystemGraphNodes represents a symbolic SystemGraphNodes.
type SystemGraphNodes struct {
	_ int
}

func NewSystemGraphNodes() *SystemGraphNodes {
	return &SystemGraphNodes{}
}

func (g *SystemGraphNodes) Test(v SymbolicValue) bool {
	other, ok := v.(*SystemGraphNodes)
	if ok {
		return true
	}
	_ = other
	return false
}

func (n *SystemGraphNodes) IsSharable() bool {
	return true
}

func (n *SystemGraphNodes) Share(originState *State) PotentiallySharable {
	return n
}

func (n *SystemGraphNodes) IsShared() bool {
	return true
}

func (d *SystemGraphNodes) Widen() (SymbolicValue, bool) {
	return nil, false
}

func (d *SystemGraphNodes) IsWidenable() bool {
	return false
}

func (d *SystemGraphNodes) IteratorElementKey() SymbolicValue {
	return ANY
}
func (d *SystemGraphNodes) IteratorElementValue() SymbolicValue {
	return ANY_SYSTEM_GRAPH_NODE
}

func (d *SystemGraphNodes) String() string {
	return "system-graph-nodes"
}

func (d *SystemGraphNodes) WidestOfType() SymbolicValue {
	return ANY_SYSTEM_GRAPH_NODES
}

// An SystemGraphNode represents a symbolic SystemGraphNode.
type SystemGraphNode struct {
	_ int
}

func NewSystemGraphNode() *SystemGraphNode {
	return &SystemGraphNode{}
}

func (n *SystemGraphNode) Test(v SymbolicValue) bool {
	other, ok := v.(*SystemGraphNode)
	if ok {
		return true
	}
	_ = other
	return false
}

func (n *SystemGraphNode) Prop(memberName string) SymbolicValue {
	switch memberName {
	case "name", "type_name":
		return ANY_STR
	}
	panic(FormatErrPropertyDoesNotExist(memberName, n))
}

func (n *SystemGraphNode) SetProp(name string, value SymbolicValue) (IProps, error) {
	return nil, errors.New(FmtCannotAssignPropertyOf(n))
}

func (n *SystemGraphNode) WithExistingPropReplaced(name string, value SymbolicValue) (IProps, error) {
	return nil, errors.New(FmtCannotAssignPropertyOf(n))
}

func (n *SystemGraphNode) PropertyNames() []string {
	return SYSTEM_GRAPH_NODE_PROPNAMES
}

func (n *SystemGraphNode) IsSharable() bool {
	return true
}

func (n *SystemGraphNode) Share(originState *State) PotentiallySharable {
	return n
}

func (n *SystemGraphNode) IsShared() bool {
	return true
}

func (n *SystemGraphNode) Widen() (SymbolicValue, bool) {
	return nil, false
}

func (n *SystemGraphNode) IsWidenable() bool {
	return false
}

func (n *SystemGraphNode) String() string {
	return "system-graph-node"
}

func (n *SystemGraphNode) WidestOfType() SymbolicValue {
	return ANY_SYSTEM_GRAPH_NODE
}
