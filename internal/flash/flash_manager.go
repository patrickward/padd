package flash

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// Flash represents a single flash message
type Flash struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Manager handles flash message operations using cookies
type Manager struct {
	cookieName string
	maxAge     int
	path       string
}

// NewManager creates a new Manager with sensible defaults
func NewManager() *Manager {
	return &Manager{
		cookieName: "padd_flash_message",
		maxAge:     300, // 5 minutes
		path:       "/",
	}
}

// Set stores a flash message in a cookie
func (fm *Manager) Set(w http.ResponseWriter, msgType, message string) {
	flash := Flash{
		Type:    msgType,
		Message: message,
	}

	// JSON encode the flash message
	flashData, err := json.Marshal(flash)
	if err != nil {
		// Fallback to simple message format if JSON encoding fails
		flashData = []byte(message)
	}

	// Create cookie
	cookie := &http.Cookie{
		Name:     fm.cookieName,
		Value:    url.QueryEscape(string(flashData)),
		Path:     fm.path,
		MaxAge:   fm.maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}

// SetSuccess is a convenience method for success messages
func (fm *Manager) SetSuccess(w http.ResponseWriter, message string) {
	fm.Set(w, "success", message)
}

// SetError is a convenience method for error messages
func (fm *Manager) SetError(w http.ResponseWriter, message string) {
	fm.Set(w, "danger", message)
}

// Get retrieves and clears a flash message from cookies
func (fm *Manager) Get(w http.ResponseWriter, r *http.Request) *Flash {
	cookie, err := r.Cookie(fm.cookieName)
	if err != nil {
		return nil
	}

	// Clear the cookie immediately by setting it to expire
	clearCookie := &http.Cookie{
		Name:     fm.cookieName,
		Value:    "",
		Path:     fm.path,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, clearCookie)

	// Decode the cookie value
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil
	}

	// Try to parse as JSON first
	var flash Flash
	if err := json.Unmarshal([]byte(decodedValue), &flash); err == nil {
		return &flash
	}

	// Fallback: treat as plain message with success type
	return &Flash{
		Type:    "success",
		Message: decodedValue,
	}
}

// HasFlash checks if there's a flash message without consuming it
func (fm *Manager) HasFlash(r *http.Request) bool {
	_, err := r.Cookie(fm.cookieName)
	return err == nil
}

// Peek gets a flash message without clearing it (useful for debugging)
func (fm *Manager) Peek(r *http.Request) *Flash {
	cookie, err := r.Cookie(fm.cookieName)
	if err != nil {
		return nil
	}

	// Decode the cookie value
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil
	}

	// Try to parse as JSON
	var flash Flash
	if err := json.Unmarshal([]byte(decodedValue), &flash); err == nil {
		return &flash
	}

	// Fallback
	return &Flash{
		Type:    "success",
		Message: decodedValue,
	}
}
