/*! Markdown Toolbar for kelp.js integration */
'use strict';

(() => {
  // Template constants to avoid string concatenation in methods
  const TOOLBAR_TEMPLATE = `
    <div class="toolbar-group">
        <button type="button" class="plain" data-action="link" title="Add Link">
            <span class="toolbar-icon toolbar-icon-link"></span>
            <span class="toolbar-label">Link</span>
        </button>
        <button type="button" class="plain" data-action="image" title="Add Image">
            <span class="toolbar-icon toolbar-icon-image"></span>
            <span class="toolbar-label">Image</span>
        </button>
        <button type="button" class="plain" data-action="icon" title="Add Icon">
            <span class="toolbar-icon toolbar-icon-star"></span>
            <span class="toolbar-label">Icon</span>
        </button>
        <button type="button" class="plain" data-action="bold" title="Bold Text (Ctrl+B)">
            <span class="toolbar-icon toolbar-icon-bold"></span>
            <span class="toolbar-label">Bold</span>
        </button>
        <button type="button" class="plain" data-action="italic" title="Italic Text (Ctrl+I)">
            <span class="toolbar-icon toolbar-icon-italic"></span>
            <span class="toolbar-label">Italic</span>
        </button>
        <button type="button" class="plain" data-action="code" title="Code">
            <span class="toolbar-icon toolbar-icon-code"></span>
            <span class="toolbar-label">Code</span>
        </button>
    </div>
    <div class="toolbar-group">
        <button type="button" class="plain primary" data-action="save" title="Save File (Ctrl+S)">
            <span class="toolbar-icon toolbar-icon-save"></span>
            <span class="toolbar-label">Save</span>
        </button>
        <button type="button" class="plain" data-action="cancel" title="Cancel">
            <span class="toolbar-icon toolbar-icon-cancel"></span>
            <span class="toolbar-label">Cancel</span>
        </button>
    </div>
  `

  customElements.define('markdown-toolbar', class extends HTMLElement {
    /** @type HTMLTextAreaElement | null */
    #textarea
    /** @type HTMLElement | null */
    #toolbar
    /** @type HTMLElement | null */
    #autogrowElement
    /** @type IntersectionObserver | null */
    #observer
    /** @type Array<string> */
    #availableIcons = []
    /** @type string */
    #cancelUrl
    /** @type string */
    #iconsApiUrl

    // Initialize on connect
    connectedCallback () {
      this.init()
    }

    // Cleanup on disconnect
    disconnectedCallback () {
      if (this.#observer) {
        this.#observer.disconnect()
      }
    }

    // Initialize the component
    async init () {
      if (this.hasAttribute('is-ready')) return

      this.#textarea = this.querySelector('textarea')
      this.#autogrowElement = this.querySelector('kelp-autogrow')
      this.#cancelUrl = this.getAttribute('cancel-url') || '/'
      this.#iconsApiUrl = this.getAttribute('icons-api') || '/api/icons'

      if (!this.#textarea || !this.#autogrowElement) {
        console.warn('markdown-toolbar: No textarea or kelp-autogrow found')
        return
      }

      // Load available icons
      await this.#loadAvailableIcons()

      this.render()
      this.#setupStickyBehavior()
      this.#setupKeyboardShortcuts()
      this.addEventListener('click', this)
      this.setAttribute('is-ready', '')
    }

    /**
     * Setup keyboard shortcuts for common markdown operations
     */
    #setupKeyboardShortcuts () {
      this.#textarea.addEventListener('keydown', (e) => {
        // Handle Ctrl/Cmd shortcuts
        if (e.ctrlKey || e.metaKey) {
          const actions = {
            'b': () => this.#wrapSelection('**', '**'),
            'i': () => this.#wrapSelection('*', '*'),
            'k': () => this.#showLinkDialog(),
            's': () => this.#handleSave()
          }

          const action = actions[e.key]
          if (action && (e.key !== 'k' || !document.querySelector('input[name="q"]'))) {
            e.preventDefault()
            action()
          }
        } else if (e.key === 'Tab') {
          e.preventDefault()
          this.#insertAtCursor('  ') // 2 spaces for indentation
        }
      })
    }

    /**
     * Load available icons from the API
     */
    async #loadAvailableIcons () {
      try {
        const response = await fetch(this.#iconsApiUrl)
        if (response.ok) {
          this.#availableIcons = await response.json()
        }
      } catch (error) {
        console.warn('Failed to load icons:', error)
        this.#availableIcons = []
      }
    }

    // Render the toolbar
    render () {
      this.#toolbar = document.createElement('div')
      this.#toolbar.className = 'markdown-toolbar'
      this.#toolbar.innerHTML = TOOLBAR_TEMPLATE

      // Insert toolbar before the kelp-autogrow element
      this.insertBefore(this.#toolbar, this.#autogrowElement)
    }

    /**
     * Setup sticky behavior for the toolbar
     */
    #setupStickyBehavior () {
      // Create a sentinel element to detect when toolbar should stick
      const sentinel = document.createElement('div')
      sentinel.className = 'toolbar-sentinel'
      this.insertBefore(sentinel, this.#toolbar)

      this.#observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
          if (entry.isIntersecting) {
            this.#toolbar.classList.remove('toolbar-sticky')
          } else {
            this.#toolbar.classList.add('toolbar-sticky')
          }
        })
      }, {
        root: null,
        rootMargin: '0px',
        threshold: 0
      })

      this.#observer.observe(sentinel)
    }

    /**
     * Handle events using event delegation
     * @param {Event} event The event object
     */
    handleEvent (event) {
      if (event.type === 'click') {
        this.#onClick(event)
      }
    }

    /**
     * Handle click events
     * @param {Event} event The event object
     */
    #onClick (event) {
      const toolbarBtn = event.target.closest('[data-action]')

      if (toolbarBtn) {
        event.preventDefault()
        this.#executeAction(toolbarBtn.getAttribute('data-action'))
      }
    }

    /**
     * Execute toolbar action
     * @param {string} action The action to execute
     */
    #executeAction (action) {
      const actions = {
        'link': () => this.#showLinkDialog(),
        'image': () => this.#showImageDialog(),
        'icon': () => this.#showIconDialog(),
        'bold': () => this.#wrapSelection('**', '**'),
        'italic': () => this.#wrapSelection('*', '*'),
        'code': () => this.#wrapSelection('`', '`'),
        'save': () => this.#handleSave(),
        'cancel': () => this.#handleCancel()
      }

      const actionFn = actions[action]
      if (actionFn) {
        actionFn()
      }
    }

    /**
     * Handle save action
     */
    #handleSave () {
      const form = this.closest('form')
      if (form) {
        // If the form element has a hx-post, hx-put, or hx-patch attribute,
        if (form.hasAttribute('hx-post') || form.hasAttribute('hx-put') || form.hasAttribute('hx-patch')) {
          window.htmx.trigger(form, 'submit')
          return
        }

        form.submit()
      }
    }

    /**
     * Handle cancel action
     */
    #handleCancel () {
      window.location.href = this.#cancelUrl
    }

    /**
     * Show link dialog
     */
    #showLinkDialog () {
      const dialog = this.#createDialog('link-dialog', 'Add Link', `
        <div class="field">
            <label for="link-text">Link Text:</label>
            <input type="text" id="link-text" placeholder="Enter link text" autofocus>
        </div>
        <div class="field">
            <label for="link-url">URL:</label>
            <input type="url" id="link-url" placeholder="https://example.com">
        </div>
        <div class="cluster gap-xs">
            <button type="button" class="btn primary" id="insert-link">Insert</button>
            <button type="button" class="btn outline" command="close" commandfor="link-dialog">Cancel</button>
        </div>
      `)

      // Add event listener for the insert button - this is what was missing!
      document.getElementById('insert-link').addEventListener('click', () => {
        const text = document.getElementById('link-text').value || 'Link'
        const url = document.getElementById('link-url').value || '#'
        this.#insertAtCursor(`[${text}](${url})`)
        this.#hideDialog('link-dialog')
      })

      this.#showDialog(dialog)
    }

    /**
     * Show image dialog
     */
    #showImageDialog () {
      const dialog = this.#createDialog('image-dialog', 'Add Image', `
        <div class="field">
            <label for="image-alt">Alt Text:</label>
            <input type="text" id="image-alt" placeholder="Describe the image" autofocus>
        </div>
        <div class="field">
            <label for="image-url">Image URL:</label>
            <input type="url" id="image-url" placeholder="/images/example.jpg">
        </div>
        <div class="cluster gap-xs">
            <button type="button" class="btn primary" id="insert-image">Insert</button>
            <button type="button" class="btn outline" command="close" commandfor="image-dialog">Cancel</button>
        </div>
      `)

      // Add event listener for the insert button - this is what was missing!
      document.getElementById('insert-image').addEventListener('click', () => {
        const alt = document.getElementById('image-alt').value || 'Image'
        const url = document.getElementById('image-url').value || '/images/example.jpg'
        this.#insertAtCursor(`![${alt}](${url})`)
        this.#hideDialog('image-dialog')
      })

      this.#showDialog(dialog)
    }

    /**
     * Show icon dialog
     */
    #showIconDialog () {
      if (!this.#availableIcons.length) {
        alert('No icons available')
        return
      }

      const iconGrid = this.#availableIcons.map(icon =>
        `<button type="button" class="markdown-icon-option" data-icon="${icon}" title="${icon}">
          <img src="/images/icons/${icon}.svg" alt="${icon}" width="20" height="20">
          <span>${icon}</span>
        </button>`
      ).join('')

      const dialog = this.#createDialog('icon-dialog', 'Add Icon', `
        <div class="field">
            <label for="icon-search">Search Icons:</label>
            <input type="text" id="icon-search" placeholder="Type to filter icons..." autofocus>
        </div>
        <div class="markdown-icon-grid">
            ${iconGrid}
        </div>
        <div class="cluster gap-xs">
            <button type="button" class="btn outline" command="close" commandfor="icon-dialog">Cancel</button>
        </div>
      `)

      // Add search functionality
      const searchInput = document.getElementById('icon-search')
      const iconOptions = dialog.querySelectorAll('.markdown-icon-option')

      searchInput.addEventListener('input', (e) => {
        const searchTerm = e.target.value.toLowerCase()
        iconOptions.forEach(option => {
          const iconName = option.getAttribute('data-icon').toLowerCase()
          option.style.display = iconName.includes(searchTerm) ? 'flex' : 'none'
        })
      })

      // Add click handlers for icon selection
      iconOptions.forEach(option => {
        option.addEventListener('click', () => {
          const iconName = option.getAttribute('data-icon')
          this.#insertAtCursor(`::${iconName}::`)
          this.#hideDialog('icon-dialog')
        })
      })

      this.#showDialog(dialog)
    }

    /**
     * Create a dialog element
     * @param {string} id Dialog ID
     * @param {string} title Dialog title
     * @param {string} content Dialog content HTML
     * @returns {HTMLDialogElement}
     */
    #createDialog (id, title, content) {
      // Remove existing dialog if present
      const existing = document.getElementById(id)
      if (existing) existing.remove()

      const dialog = document.createElement('dialog')
      dialog.id = id
      dialog.className = 'markdown-toolbar-dialog'
      dialog.setAttribute('closedby', 'any')
      dialog.innerHTML = `
        <header>
            <h3>${title}</h3>
        </header>
        <div class="dialog-content">
            ${content}
        </div>
      `

      document.body.appendChild(dialog)
      return dialog
    }

    /**
     * Show dialog
     * @param {HTMLDialogElement} dialog
     */
    #showDialog (dialog) {
      dialog.showModal()
      // Focus first input if available
      const firstInput = dialog.querySelector('input')
      if (firstInput) firstInput.focus()
    }

    /**
     * Hide dialog
     * @param {string} id Dialog ID
     */
    #hideDialog (id) {
      const dialog = document.getElementById(id)
      if (dialog) {
        dialog.close()
        dialog.remove()
      }
    }

    /**
     * Wrap selected text with given strings
     * @param {string} before Text to add before selection
     * @param {string} after Text to add after selection
     */
    #wrapSelection (before, after) {
      const start = this.#textarea.selectionStart
      const end = this.#textarea.selectionEnd
      const selectedText = this.#textarea.value.substring(start, end)
      const replacement = before + selectedText + after

      this.#textarea.setRangeText(replacement, start, end)

      // Set cursor position
      const newPosition = start + before.length + selectedText.length
      this.#textarea.setSelectionRange(newPosition, newPosition)
      this.#textarea.focus()
    }

    /**
     * Insert text at cursor position
     * @param {string} text Text to insert
     */
    #insertAtCursor (text) {
      const start = this.#textarea.selectionStart
      const end = this.#textarea.selectionEnd

      this.#textarea.setRangeText(text, start, end)

      // Set cursor position after inserted text
      const newPosition = start + text.length
      this.#textarea.setSelectionRange(newPosition, newPosition)
      this.#textarea.focus()
    }
  })
})()
