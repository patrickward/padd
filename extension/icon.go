package extension

import (
	"io/fs"
	"path/filepath"
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

var iconRegexp = regexp.MustCompile(`::([a-zA-Z0-9\-_]+)::`)

// IconChecker is an interface for checking if an icon exists.
type IconChecker interface {
	IconExists(iconName string) bool
}

// FileExistsChecker is an interface for checking if a file exists.
type FileExistsChecker interface {
	FileExists(filename string) bool
}

type iconParser struct {
	iconChecker IconChecker
}

// NewIconParser creates a new icon parser with the given icon checker.
func NewIconParser(iconChecker IconChecker) parser.InlineParser {
	return &iconParser{iconChecker: iconChecker}
}

func (p *iconParser) Trigger() []byte {
	return []byte{':'}
}

func (p *iconParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	line, _ := block.PeekLine()
	m := iconRegexp.FindSubmatchIndex(line)
	if m == nil || m[0] != 0 {
		return nil
	}

	iconName := string(line[m[2]:m[3]])
	iconName = strings.TrimSpace(iconName)

	// Skip empty lines
	if iconName == "" {
		return nil
	}

	if p.iconChecker != nil && !p.iconChecker.IconExists(iconName) {
		return nil
	}

	block.Advance(m[1])
	return ast.NewIcon(iconName)
}

// IconHMTMLRenderer is a renderer for the Icon node.
type IconHMTMLRenderer struct {
	html.Config
}

// NewIconHTMLRenderer creates a new IconHTMLRenderer.
func NewIconHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &IconHMTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *IconHMTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindIcon, r.renderIcon)
}

func (r *IconHMTMLRenderer) renderIcon(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}

	n := node.(*ast.Icon)

	iconName := strings.TrimSuffix(n.Name, ".svg")
	_, _ = w.WriteString(`<span class="icon">`)
	_, _ = w.WriteString(`<img alt="` + iconName + `" src="/images/icons/` + n.Name + `.svg" />`)
	_, _ = w.WriteString(`</span>`)

	return gast.WalkContinue, nil
}

// DefaultIconChecker is a no-op icon checker that always returns true.
type DefaultIconChecker struct {
	fileManager FileExistsChecker
	staticFS    fs.FS
}

// NewDefaultIconChecker creates a new DefaultIconChecker with the given file manager.
func NewDefaultIconChecker(fileManager FileExistsChecker, staticFS fs.FS) *DefaultIconChecker {
	return &DefaultIconChecker{fileManager: fileManager, staticFS: staticFS}
}

// IconExists checks if the icon exists in the user's directory or in the embedded static files.
func (c *DefaultIconChecker) IconExists(iconName string) bool {
	// Add .svg if not present
	if !strings.HasSuffix(iconName, ".svg") {
		iconName = iconName + ".svg"
	}

	// Check the user's path first
	if c.fileManager != nil {
		userSVGPath := filepath.Join("images", "icons", iconName)
		if c.fileManager.FileExists(userSVGPath) {
			return true
		}
	}

	// Fallback to static embedded files
	staticPath := "static/images/icons/" + iconName
	if file, err := c.staticFS.Open(staticPath); err == nil {
		defer func(file fs.File) {
			_ = file.Close()
		}(file)

		if stat, err := file.Stat(); err == nil && !stat.IsDir() {
			return true
		}
	}

	return false
}

// IconExtension is a Goldmark extension for handling icon shortcodes.
type iconExtension struct {
	iconChecker IconChecker
}

// Icon implements the Goldmark Extension interface.
var Icon = &iconExtension{iconChecker: &DefaultIconChecker{}}

// NewIconExtension creates a new IconExtension with a custom icon checker.
func NewIconExtension(iconChecker IconChecker) goldmark.Extender {
	return &iconExtension{iconChecker: iconChecker}
}

func (e *iconExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(NewIconParser(e.iconChecker), 200),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewIconHTMLRenderer(), 500),
	))
}
