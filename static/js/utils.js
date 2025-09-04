'use strict';

(() => {
  // Helper function to get app specific data from meta-tags
  window.getAppMeta = function getAppMeta (name) {
    const metaTag = document.querySelector(`meta[name="app:${name}"]`)
    return metaTag ? metaTag.content : null
  }

  // Keyboard shortcuts for search and edit
  document.addEventListener('keydown', function (e) {
    // Ctrl+K to focus search
    if (e.ctrlKey && e.key === 'k' || e.metaKey && e.key === 'k') {
      e.preventDefault()
      document.querySelector('input[name="q"]').focus()
    }

    // Ctrl+E to edit (when viewing)
    if (e.ctrlKey && e.key === 'e' && !document.querySelector('.editor')) {
      const editBtn = document.querySelector('a[href*="/edit"]')
      if (editBtn) {
        e.preventDefault()
        editBtn.click()
      }
    }
  })

  // Auto-scroll to search match if present
  document.addEventListener('DOMContentLoaded', function () {
    const searchMatch = window.getAppMeta('search-match')
    if (searchMatch > 0) {
      const targetElement = document.getElementById('search-match-' + searchMatch)
      if (targetElement) {
        // Scroll to the element
        setTimeout(() => {
          targetElement.scrollIntoView({
            behavior: 'smooth',
            block: 'center'
          })
        }, 200) // Delay to ensure rendering is complete
      }
    }
  })

  // Add custom headers to all htmx requests
  document.addEventListener("htmx:configRequest", (evt) => {
    // Add the page id to the evt.detail.headers object as a name/value pair
    const fileId = document.querySelector('meta[name="app:file-id"]').content;
    if (fileId) {
      evt.detail.headers['X-PADD-File-ID'] = fileId;
    }
  });

})()
