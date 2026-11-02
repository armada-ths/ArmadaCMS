package translations

import (
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Compare structure and non-translatable payloads, not the prose text or the
// number of text nodes (which can change naturally between languages).
func markdownShape(value string) []string {
	source := []byte(value)
	parser := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser()
	root := parser.Parse(text.NewReader(source))
	shape := []string{}
	var visit func(ast.Node)
	visit = func(node ast.Node) {
		if node.Kind() == ast.KindText || node.Kind() == ast.KindString {
			if t, ok := node.(*ast.Text); ok && t.HardLineBreak() {
				shape = append(shape, "hard-break")
			}
			return
		}
		marker := node.Kind().String()
		atomic := false
		switch n := node.(type) {
		case *ast.Heading:
			marker += fmt.Sprint(n.Level)
		case *ast.Emphasis:
			marker += fmt.Sprint(n.Level)
		case *ast.List:
			marker += fmt.Sprint(n.Marker, n.Start, n.IsTight)
		case *ast.Link:
			marker += string(n.Destination)
		case *ast.Image:
			marker += string(n.Destination)
		case *ast.AutoLink:
			marker += string(n.URL(source))
		case *extast.TaskCheckBox:
			marker += fmt.Sprint(n.IsChecked)
		case *extast.Table:
			marker += fmt.Sprint(n.Alignments)
		case *ast.CodeSpan:
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					marker += string(t.Segment.Value(source))
				}
			}
			atomic = true
		case *ast.FencedCodeBlock:
			if n.Info != nil {
				marker += string(n.Info.Segment.Value(source))
			}
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				marker += string(line.Value(source))
			}
			atomic = true
		case *ast.CodeBlock:
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				marker += string(line.Value(source))
			}
			atomic = true
		}
		shape = append(shape, "<"+marker+">")
		if !atomic {
			for c := node.FirstChild(); c != nil; c = c.NextSibling() {
				visit(c)
			}
		}
		shape = append(shape, "</"+node.Kind().String()+">")
	}
	visit(root)
	return shape
}
