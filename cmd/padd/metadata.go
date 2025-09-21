package main

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/patrickward/padd"
)

type MetadataConfig struct {
	StatusColors   map[string]string `json:"status_colors"`
	PriorityColors map[string]string `json:"priority_colors"`
	DueColor       string
	TagColor       string
	ContextColor   string
}

func (s *Server) addMetadataToPageData(data padd.PageData, metadata map[string]any) padd.PageData {
	data.Encrypted = getMetadataBool(metadata, "encrypted", data.Encrypted)
	data.Description = getMetadataString(metadata, "description", data.Description)
	data.Category = getMetadataString(metadata, "category", data.Category)
	status := getMetadataString(metadata, "status", data.Status)
	data.Status = status
	data.StatusColor = s.getStatusColor(status)
	priority := getMetadataString(metadata, "priority", data.Priority)
	data.Priority = priority
	data.PriorityColor = s.getPriorityColor(priority)
	data.DueDate = getMetadataString(metadata, "due_date", data.DueDate)
	data.DueColor = s.getDueColor()
	data.TagColor = s.getTagColor()
	data.ContextColor = s.getContextColor()
	data.CreatedAt = getMetadataString(metadata, "created_at", data.CreatedAt)
	data.UpdatedAt = getMetadataString(metadata, "updated_at", data.UpdatedAt)
	data.Author = getMetadataString(metadata, "author", data.Author)
	data.Tags = getMetadataStringSlice(metadata, "tags")
	data.Contexts = getMetadataStringSlice(metadata, "contexts")
	return data
}

func (s *Server) setupMetadataConfig() {
	metadata := MetadataConfig{
		StatusColors: map[string]string{
			"draft":       "neutral muted",
			"in-progress": "primary muted",
			"review":      "secondary muted",
			"completed":   "success muted",
			"complete":    "success muted",
			"on-hold":     "warning muted",
			"held":        "warning muted",
			"cancelled":   "danger muted",
			"canceled":    "danger muted",
		},
		PriorityColors: map[string]string{
			"low":    "success muted",
			"medium": "warning muted",
			"high":   "danger muted",
		},
		DueColor:     "danger muted",
		TagColor:     "primary muted",
		ContextColor: "secondary muted",
	}

	// Find the "metadata.json" file from the user's data directory
	if s.rootManager.FileExists("metadata.json") {
		content, err := s.rootManager.ReadFile("metadata.json")
		if err == nil {
			// Read and parse the JSON content
			var fileMetadata map[string]any
			err := json.Unmarshal(content, &fileMetadata)
			if err != nil {
				log.Printf("Error parsing metadata.json: %v\n", err)
			}

			if err == nil {
				if statusColors, ok := fileMetadata["status_colors"].(map[string]any); ok {
					for k, v := range statusColors {
						if color, ok := v.(string); ok {
							metadata.StatusColors[k] = color
						}
					}
				}
				if priorityColors, ok := fileMetadata["priority_colors"].(map[string]any); ok {
					for k, v := range priorityColors {
						if color, ok := v.(string); ok {
							metadata.PriorityColors[k] = color
						}
					}
				}
				if dueColor, ok := fileMetadata["due_color"].(string); ok {
					metadata.DueColor = dueColor
				}
				if tagColor, ok := fileMetadata["tag_color"].(string); ok {
					metadata.TagColor = tagColor
				}
				if contextColor, ok := fileMetadata["context_color"].(string); ok {
					metadata.ContextColor = contextColor
				}
			}
		}
	}

	s.metadataConfig = metadata
}

func (s *Server) getStatusColor(status string) string {
	if color, ok := s.metadataConfig.StatusColors[status]; ok {
		return color
	}
	return "neutral muted"
}

func (s *Server) getPriorityColor(priority string) string {
	if color, ok := s.metadataConfig.PriorityColors[priority]; ok {
		return color
	}
	return "neutral muted"
}

func (s *Server) getDueColor() string {
	if s.metadataConfig.DueColor != "" {
		return s.metadataConfig.DueColor
	}

	return "danger muted"
}

func (s *Server) getTagColor() string {
	if s.metadataConfig.TagColor != "" {
		return s.metadataConfig.TagColor
	}

	return "primary muted"
}

func (s *Server) getContextColor() string {
	if s.metadataConfig.ContextColor != "" {
		return s.metadataConfig.ContextColor
	}

	return "secondary muted"
}

func getMetadataBool(metadata map[string]any, key string, defaultValue bool) bool {
	if value, ok := metadata[key]; ok {
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str) == "true" || str == "yes"
		} else if boolVal, ok := value.(bool); ok {
			return boolVal
		}
	}

	return defaultValue
}

func getMetadataString(metadata map[string]any, key string, defaultValue string) string {
	if value, ok := metadata[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getMetadataStringSlice(metadata map[string]any, key string) []string {
	if value, ok := metadata[key]; ok {
		if slice, ok := value.([]any); ok {
			var result []string
			for _, v := range slice {
				if str, ok := v.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return nil
}
