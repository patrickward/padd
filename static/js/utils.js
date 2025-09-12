'use strict';

(() => {
  // Helper function to get app specific data from meta-tags
  window.getAppMeta = function getAppMeta (name) {
    const metaTag = document.querySelector(`meta[name="app:${name}"]`)
    return metaTag ? metaTag.content : null
  }

  if (!window.paddListenersAdded) {
    window.paddListenersAdded = true

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

    document.body.addEventListener('htmx:beforeSwap', function (evt) {
      if (evt.detail.xhr.status === 400 || evt.detail.xhr.status === 404 || evt.detail.xhr.status === 422 || evt.detail.xhr.status === 500) {
        // if the response code is 404, 422, or 500, we want to swap the content
        evt.detail.shouldSwap = true
        // set isError to 'false' to avoid error logging in console
        evt.detail.isError = false
      }
    })
  }

})()
