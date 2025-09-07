package extension

// Credit: Original from github.com/yuin/goldmark/extension/tasklist
// This is a modified version to let the checkboxes be more interactive for the padd app

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/patrickward/padd/extension/ast"
)

var TasksCountKey = parser.NewContextKey()
var CompletedTasksCountKey = parser.NewContextKey()

var taskListRegexp = regexp.MustCompile(`^\[([\sxX])\](.*)$`)

type taskCheckBoxParser struct {
}

var defaultTaskCheckBoxParser = &taskCheckBoxParser{}

type TasksStatsInfo struct {
	Total     int
	Completed int
	Pending   int
}

// TaskStats returns the number of completed tasks and whether there are any tasks at all.
func TaskStats(pc parser.Context) TasksStatsInfo {
	total := 0
	pending := 0
	completed := 0
	if val := pc.Get(TasksCountKey); val != nil {
		if count, ok := val.(int); ok {
			total = count
			if val := pc.Get(CompletedTasksCountKey); val != nil {
				if completedCount, ok := val.(int); ok {
					completed = completedCount
					pending = total - completed
				}
			}
		}
	}

	return TasksStatsInfo{
		Total:     total,
		Completed: completed,
		Pending:   pending,
	}
}

// NewTaskCheckBoxParser returns a new  InlineParser that can parse
// checkboxes in list items.
// This parser must take precedence over the parser.LinkParser.
func NewTaskCheckBoxParser() parser.InlineParser {
	return defaultTaskCheckBoxParser
}

func (s *taskCheckBoxParser) Trigger() []byte {
	return []byte{'['}
}

func (s *taskCheckBoxParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	// Given AST structure must be like
	// - List
	//   - ListItem         : parent.Parent
	//     - TextBlock      : parent
	//       (current line)
	if parent.Parent() == nil || parent.Parent().FirstChild() != parent {
		return nil
	}

	if parent.HasChildren() {
		return nil
	}
	if _, ok := parent.Parent().(*gast.ListItem); !ok {
		return nil
	}
	line, _ := block.PeekLine()
	m := taskListRegexp.FindSubmatchIndex(line)
	if m == nil {
		return nil
	}

	// Get the number of checkboxes so far
	checkboxCount := 0
	if val := pc.Get(TasksCountKey); val != nil {
		if count, ok := val.(int); ok {
			checkboxCount = count
		}
	}
	checkboxCount++
	pc.Set(TasksCountKey, checkboxCount)

	completedCount := 0
	if val := pc.Get(CompletedTasksCountKey); val != nil {
		if count, ok := val.(int); ok {
			completedCount = count
		}
	}

	value := line[m[2]:m[3]][0]
	label := strings.TrimSpace(string(line[m[4]:]))
	block.Advance(m[1])
	checked := value == 'x' || value == 'X'
	if checked {
		completedCount++
		pc.Set(CompletedTasksCountKey, completedCount)
	}
	return ast.NewTaskCheckBox(checked, checkboxCount, template.HTMLEscapeString(label))
}

func (s *taskCheckBoxParser) CloseBlock(parent gast.Node, pc parser.Context) {
	// nothing to do
}

// TaskCheckBoxHTMLRenderer is a renderer.NodeRenderer implementation that
// renders checkboxes in list items.
type TaskCheckBoxHTMLRenderer struct {
	html.Config
}

// NewTaskCheckBoxHTMLRenderer returns a new TaskCheckBoxHTMLRenderer.
func NewTaskCheckBoxHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &TaskCheckBoxHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

// RegisterFuncs implements renderer.NodeRenderer.RegisterFuncs.
func (r *TaskCheckBoxHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindTaskCheckBox, r.renderTaskCheckBox)
}

func (r *TaskCheckBoxHTMLRenderer) renderTaskCheckBox(
	w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*ast.TaskCheckBox)

	// Add the wrapper
	_, _ = w.WriteString(fmt.Sprintf(`<div id="tasklist-item-%d" class="tasklist-item">`, n.CheckboxID))

	// Add the checkbox input
	if n.IsChecked {
		_, _ = w.WriteString(`<input checked="" type="checkbox"`)
	} else {
		_, _ = w.WriteString(`<input type="checkbox"`)
	}

	// Add the line number as a data attribute for potential use in the frontend
	_, _ = w.WriteString(fmt.Sprintf(` hx-patch="/api/tasks/toggle/%d" hx-swap="innerHTML" hx-target="#tasklist-label-%d" data-checkbox-id="%d"`, n.CheckboxID, n.CheckboxID, n.CheckboxID))

	if r.XHTML {
		_, _ = w.WriteString(" /> ")
	} else {
		_, _ = w.WriteString("> ")
	}

	// Add the label
	_, _ = w.WriteString(fmt.Sprintf(`<span hx-get="/api/tasks/edit/%d" hx-swap="innerHTML" hx-target="#tasklist-item-%d" id="tasklist-label-%d" class="tasklist-label fade-in">%s</span>`, n.CheckboxID, n.CheckboxID, n.CheckboxID, n.Label))

	_, _ = w.WriteString(`</div>`)

	return gast.WalkContinue, nil
}

type taskList struct {
}

// TaskList is an extension that allow you to use GFM task lists.
var TaskList = &taskList{}

func (e *taskList) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(NewTaskCheckBoxParser(), 0),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewTaskCheckBoxHTMLRenderer(), 500),
	))
}
