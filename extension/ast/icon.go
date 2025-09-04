package ast

import gast "github.com/yuin/goldmark/ast"

// An Icon struct represents an icon shortcode in the AST.
type Icon struct {
	gast.BaseInline
	Name string
}

// Dump implements Node.Dump.
func (n *Icon) Dump(source []byte, level int) {
	m := map[string]string{
		"Name": n.Name,
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// KindIcon is a NodeKind of the Icon node.
var KindIcon = gast.NewNodeKind("Icon")

// Kind implements Node.Kind.
func (n *Icon) Kind() gast.NodeKind {
	return KindIcon
}

// NewIcon returns a new Icon node.
func NewIcon(name string) *Icon {
	return &Icon{
		Name: name,
	}
}
