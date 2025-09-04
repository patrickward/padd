package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/patrickward/padd"
)

func customFuncs() template.FuncMap {
	return template.FuncMap{
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"toLower":   strings.ToLower,
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
}

func parseTemplates() (*template.Template, error) {
	funcsMap := customFuncs()
	return template.New("").Funcs(funcsMap).ParseFS(padd.TemplateFS,
		"templates/layouts/*.html",
		"templates/partials/*.html",
	)
}

// executePage renders a full page template with the given data
func (s *Server) executePage(w http.ResponseWriter, page string, data PageData) error {
	// Clone the base template to avoid altering it
	tmpl, err := s.baseTempl.Clone()
	if err != nil {
		return err
	}

	// Add .html extension if missing
	if !strings.HasSuffix(page, ".html") {
		page = page + ".html"
	}

	// Parse the specific page template
	pagePattern := fmt.Sprintf("templates/pages/%s", page)
	tmpl, err = tmpl.ParseFS(padd.TemplateFS, pagePattern)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, page, data)
}

// executeSnippet renders a snippet template (partial) with the given data (no layout)
func (s *Server) executeSnippet(w http.ResponseWriter, page string, data map[string]any) error {
	// Clone the base template to avoid altering it
	tmpl, err := s.baseTempl.Clone()
	if err != nil {
		return err
	}

	// Add .html extension if missing
	if !strings.HasSuffix(page, ".html") {
		page = page + ".html"
	}

	// Parse the specific snippet template
	snippetPattern := fmt.Sprintf("templates/snippets/%s", page)
	tmpl, err = tmpl.ParseFS(padd.TemplateFS, snippetPattern)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, page, data)
}
