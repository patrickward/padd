package ast

// Credit: Original from github.com/yuin/goldmark/extension/ast/tasklist
// This is a modified version to let the checkboxes be more interactive for the padd app

import (
	"fmt"

	gast "github.com/yuin/goldmark/ast"
)

// A TaskCheckBox struct represents a checkbox of a task list.
type TaskCheckBox struct {
	gast.BaseInline
	IsChecked  bool
	CheckboxID int
	Label      string
}

// Dump implements Node.Dump.
func (n *TaskCheckBox) Dump(source []byte, level int) {
	m := map[string]string{
		"Checked": fmt.Sprintf("%v", n.IsChecked),
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// KindTaskCheckBox is a NodeKind of the TaskCheckBox node.
var KindTaskCheckBox = gast.NewNodeKind("TaskCheckBox")

// Kind implements Node.Kind.
func (n *TaskCheckBox) Kind() gast.NodeKind {
	return KindTaskCheckBox
}

// NewTaskCheckBox returns a new TaskCheckBox node.
func NewTaskCheckBox(checked bool, checkboxID int, label string) *TaskCheckBox {
	return &TaskCheckBox{
		IsChecked:  checked,
		CheckboxID: checkboxID,
		Label:      label,
	}
}
