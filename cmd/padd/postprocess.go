package main

// postProcessContent applies post-processing to the given content after initial Markdown rendering.
func (s *Server) postProcessContent(content string) string {
	return s.processInlineSVG(content)
}
