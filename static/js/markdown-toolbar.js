/*! Markdown Toolbar for PADD
 * This is a simple Markdown toolbar that can assist in writing Markdown content for PADD.
 *
 * To add a new toolbar button:
 * 1. add a new entry to the TOOLBAR_CONFIG object
 * 2. add a new method to the ActionHandlers class
 * 3. You may need to add a new CSS class to the toolbar for the icon
 */
'use strict';

(() => {
  /**
   * Configuration object for toolbar buttons
   */
  const TOOLBAR_CONFIG = {
    groups: [
      {
        name: 'formatting',
        buttons: [
          {
            action: 'link',
            title: 'Add Link',
            icon: 'link',
            label: 'Link',
            shortcut: 'k'
          },
          {
            action: 'image',
            title: 'Add Image',
            icon: 'image',
            label: 'Image'
          },
          {
            action: 'icon',
            title: 'Add Icon',
            icon: 'star',
            label: 'Icon'
          },
          {
            action: 'bold',
            title: 'Bold Text (Ctrl+B)',
            icon: 'bold',
            label: 'Bold',
            shortcut: 'b',
            markdown: { before: '**', after: '**' }
          },
          {
            action: 'italic',
            title: 'Italic Text (Ctrl+I)',
            icon: 'italic',
            label: 'Italic',
            shortcut: 'i',
            markdown: { before: '*', after: '*' }
          },
          {
            action: 'code',
            title: 'Code',
            icon: 'code',
            label: 'Code',
            shortcut: '`',
            markdown: { before: '`', after: '`' }
          }
        ]
      },
      {
        name: 'actions',
        buttons: [
          {
            action: 'save',
            title: 'Save File (Ctrl+S)',
            icon: 'save',
            label: 'Save',
            class: 'primary',
            shortcut: 's'
          },
          {
            action: 'cancel',
            title: 'Cancel',
            icon: 'cancel',
            label: 'Cancel'
          }
        ]
      }
    ]
  };

  /**
   * Action handlers for toolbar buttons
   */
  class ActionHandlers {
    constructor(toolbar) {
      this.toolbar = toolbar;
    }

    // Simple markdown formatting actions
    bold() { this.toolbar.wrapSelection('**', '**'); }
    italic() { this.toolbar.wrapSelection('*', '*'); }
    code() { this.toolbar.wrapSelection('`', '`'); }

    // Dialog-based actions
    link() { this.toolbar.showDialog('link'); }
    image() { this.toolbar.showDialog('image'); }
    icon() { this.toolbar.showDialog('icon'); }

    // Form actions
    save() { this.toolbar.handleFormAction('save'); }
    cancel() { this.toolbar.handleFormAction('cancel'); }
  }

  /**
   * Dialog configurations
   */
  const DIALOG_CONFIGS = {
    link: {
      title: 'Add Link',
      fields: [
        { id: 'text', label: 'Link Text', type: 'text', placeholder: 'Enter link text', autofocus: true },
        { id: 'url', label: 'URL', type: 'url', placeholder: 'https://example.com' }
      ],
      onInsert: (fields) => `[${fields.text || 'Link'}](${fields.url || '#'})`
    },

    image: {
      title: 'Add Image',
      fields: [
        { id: 'alt', label: 'Alt Text', type: 'text', placeholder: 'Describe the image', autofocus: true },
        { id: 'url', label: 'Image URL', type: 'url', placeholder: '/images/example.jpg' },
        {
          id: 'upload',
          label: 'Or, upload an image',
          type: 'file',
          accept: 'image/*',
          help: 'Image will get converted to a base64 data-uri.',
          handler: 'handleImageUpload'
        }
      ],
      onInsert: (fields) => `![${fields.alt || 'Image'}](${fields.url || '/images/example.jpg'})`
    },

    icon: {
      title: 'Add Icon',
      custom: true, // Uses custom rendering
      onInsert: (iconName) => `::${iconName}::`
    }
  };

  /**
   * Main MarkdownToolbar class
   */
  class MarkdownToolbar extends HTMLElement {
    #textarea = null;
    #toolbar = null;
    #autogrowElement = null;
    #observer = null;
    #availableIcons = [];
    #config = {
      cancelUrl: '/',
      iconsApiUrl: '/api/icons'
    };

    constructor() {
      super();
      this.actions = new ActionHandlers(this);
    }

    connectedCallback() {
      this.init();
    }

    disconnectedCallback() {
      this.#observer?.disconnect();
    }

    async init() {
      if (this.hasAttribute('is-ready')) return;

      this.#initializeElements();
      this.#loadConfig();

      if (!this.#textarea || !this.#autogrowElement) {
        console.warn('markdown-toolbar: Required elements not found');
        return;
      }

      await this.#loadAvailableIcons();
      this.#render();
      this.#setupBehaviors();
      this.setAttribute('is-ready', '');
    }

    #initializeElements() {
      this.#textarea = this.querySelector('textarea');
      this.#autogrowElement = this.querySelector('kelp-autogrow');
    }

    #loadConfig() {
      this.#config.cancelUrl = this.getAttribute('cancel-url') || '/';
      this.#config.iconsApiUrl = this.getAttribute('icons-api-url') || '/api/icons';
    }

    async #loadAvailableIcons() {
      try {
        const response = await fetch(this.#config.iconsApiUrl);
        if (response.ok) {
          this.#availableIcons = await response.json();
        }
      } catch (error) {
        console.warn('Failed to load icons:', error);
      }
    }

    #render() {
      this.#toolbar = document.createElement('div');
      this.#toolbar.className = 'markdown-toolbar';
      this.#toolbar.innerHTML = this.#generateToolbarHTML();
      this.insertBefore(this.#toolbar, this.#autogrowElement);
    }

    #generateToolbarHTML() {
      return TOOLBAR_CONFIG.groups.map(group =>
        `<div class="toolbar-group">
          ${group.buttons.map(button => this.#generateButtonHTML(button)).join('')}
        </div>`
      ).join('');
    }

    #generateButtonHTML(button) {
      const className = `plain ${button.class || ''}`.trim();
      return `
        <button type="button" class="${className}" data-action="${button.action}" title="${button.title}">
          <span class="toolbar-icon toolbar-icon-${button.icon}"></span>
          <span class="toolbar-label">${button.label}</span>
        </button>
      `;
    }

    #setupBehaviors() {
      this.#setupStickyBehavior();
      this.#setupKeyboardShortcuts();
      this.addEventListener('click', this.#handleClick.bind(this));
    }

    #setupStickyBehavior() {
      const sentinel = document.createElement('div');
      sentinel.className = 'toolbar-sentinel';
      this.insertBefore(sentinel, this.#toolbar);

      this.#observer = new IntersectionObserver(entries => {
        entries.forEach(entry => {
          this.#toolbar.classList.toggle('toolbar-sticky', !entry.isIntersecting);
        });
      });

      this.#observer.observe(sentinel);
    }

    #setupKeyboardShortcuts() {
      const shortcutMap = this.#buildShortcutMap();

      this.#textarea.addEventListener('keydown', (e) => {
        if (e.ctrlKey || e.metaKey) {
          const action = shortcutMap[e.key];
          if (action && (e.key !== 'k' || !document.querySelector('input[name="q"]'))) {
            e.preventDefault();
            this.executeAction(action);
          }
        } else if (e.key === 'Tab') {
          e.preventDefault();
          this.insertAtCursor('  ');
        }
      });
    }

    #buildShortcutMap() {
      const shortcuts = {};
      TOOLBAR_CONFIG.groups.forEach(group => {
        group.buttons.forEach(button => {
          if (button.shortcut) {
            shortcuts[button.shortcut] = button.action;
          }
        });
      });
      return shortcuts;
    }

    #handleClick(event) {
      const button = event.target.closest('[data-action]');
      if (button) {
        event.preventDefault();
        this.executeAction(button.getAttribute('data-action'));
      }
    }

    executeAction(action) {
      const handler = this.actions[action];
      if (handler) {
        handler.call(this.actions);
      }
    }

    // Public API methods
    wrapSelection(before, after) {
      const { selectionStart: start, selectionEnd: end, value } = this.#textarea;
      const selectedText = value.substring(start, end);
      const replacement = before + selectedText + after;

      this.#textarea.setRangeText(replacement, start, end);

      const newPosition = start + before.length + selectedText.length;
      this.#textarea.setSelectionRange(newPosition, newPosition);
      this.#textarea.focus();
    }

    insertAtCursor(text) {
      const { selectionStart: start, selectionEnd: end } = this.#textarea;

      this.#textarea.setRangeText(text, start, end);

      const newPosition = start + text.length;
      this.#textarea.setSelectionRange(newPosition, newPosition);
      this.#textarea.focus();
    }

    showDialog(type) {
      const config = DIALOG_CONFIGS[type];
      if (!config) return;

      if (config.custom) {
        this.#showCustomDialog(type, config);
      } else {
        this.#showFieldDialog(type, config);
      }
    }

    #showFieldDialog(type, config) {
      const dialog = this.#createDialog(`${type}-dialog`, config.title,
        this.#generateDialogContent(type, config));

      // Setup field handlers
      config.fields.forEach(field => {
        if (field.handler) {
          const element = document.getElementById(`${type}-${field.id}`);
          if (element && this[field.handler]) {
            element.addEventListener('change', (e) => {
              if (e.target.files?.[0]) {
                this[field.handler](e.target.files[0], dialog);
              }
            });
          }
        }
      });

      // Setup insert button
      document.getElementById(`insert-${type}`).addEventListener('click', () => {
        const fields = this.#collectDialogFields(type, config.fields);
        const markdown = config.onInsert(fields);
        this.insertAtCursor(markdown);
        this.#hideDialog(`${type}-dialog`);
      });

      this.#showDialogElement(dialog);
    }

    #showCustomDialog(type, config) {
      if (type === 'icon') {
        this.#showIconDialog(config);
      }
    }

    #showIconDialog(config) {
      if (!this.#availableIcons.length) {
        alert('No icons available');
        return;
      }

      const iconGrid = this.#availableIcons.map(icon =>
        `<button type="button" class="markdown-icon-option" data-icon="${icon}" title="${icon}">
          <img src="/images/icons/${icon}.svg" alt="${icon}" width="20" height="20">
          <span>${icon}</span>
        </button>`
      ).join('');

      const dialog = this.#createDialog('icon-dialog', config.title, `
        <div class="field">
          <label for="icon-search">Search Icons:</label>
          <input type="text" id="icon-search" placeholder="Type to filter icons..." autofocus>
        </div>
        <div class="markdown-icon-grid">${iconGrid}</div>
        <div class="cluster gap-xs">
          <button type="button" class="btn outline" command="close" commandfor="icon-dialog">Cancel</button>
        </div>
      `);

      this.#setupIconDialogBehavior(dialog, config);
      this.#showDialogElement(dialog);
    }

    #setupIconDialogBehavior(dialog, config) {
      const searchInput = dialog.querySelector('#icon-search');
      const iconOptions = dialog.querySelectorAll('.markdown-icon-option');

      // Search functionality
      searchInput.addEventListener('input', (e) => {
        const searchTerm = e.target.value.toLowerCase();
        iconOptions.forEach(option => {
          const iconName = option.getAttribute('data-icon').toLowerCase();
          option.style.display = iconName.includes(searchTerm) ? 'flex' : 'none';
        });
      });

      // Icon selection
      iconOptions.forEach(option => {
        option.addEventListener('click', () => {
          const iconName = option.getAttribute('data-icon');
          const markdown = config.onInsert(iconName);
          this.insertAtCursor(markdown);
          this.#hideDialog('icon-dialog');
        });
      });
    }

    #generateDialogContent(type, config) {
      const fieldsHTML = config.fields.map(field => {
        const fieldId = `${type}-${field.id}`;
        let fieldHTML = `
          <div class="field">
            <label for="${fieldId}">${field.label}:</label>
        `;

        if (field.type === 'file') {
          fieldHTML += `
            <input type="${field.type}" id="${fieldId}" accept="${field.accept || ''}" ${field.autofocus ? 'autofocus' : ''}>
            ${field.help ? `<div class="size-xs text-muted">${field.help}</div>` : ''}
          `;
        } else {
          fieldHTML += `
            <input type="${field.type}" id="${fieldId}" placeholder="${field.placeholder || ''}" ${field.autofocus ? 'autofocus' : ''}>
          `;
        }

        return fieldHTML + '</div>';
      }).join('');

      return `
        ${fieldsHTML}
        ${config.fields.some(f => f.type === 'file') ? '<div id="upload-status" class="upload-status" style="display: none;"></div>' : ''}
        <div class="cluster gap-xs">
          <button type="button" class="btn primary" id="insert-${type}">Insert</button>
          <button type="button" class="btn outline" command="close" commandfor="${type}-dialog">Cancel</button>
        </div>
      `;
    }

    #collectDialogFields(type, fieldConfigs) {
      const fields = {};
      fieldConfigs.forEach(field => {
        const element = document.getElementById(`${type}-${field.id}`);
        if (element) {
          fields[field.id] = element.value;
        }
      });
      return fields;
    }

    #createDialog(id, title, content) {
      // Remove existing dialog
      document.getElementById(id)?.remove();

      const dialog = document.createElement('dialog');
      dialog.id = id;
      dialog.className = 'markdown-toolbar-dialog';
      dialog.setAttribute('closedby', 'any');
      dialog.innerHTML = `
        <header><h3>${title}</h3></header>
        <div class="dialog-content">${content}</div>
      `;

      document.body.appendChild(dialog);
      return dialog;
    }

    #showDialogElement(dialog) {
      dialog.showModal();
      dialog.querySelector('input')?.focus();
    }

    #hideDialog(id) {
      const dialog = document.getElementById(id);
      if (dialog) {
        dialog.close();
        dialog.remove();
      }
    }

    handleFormAction(action) {
      if (action === 'save') {
        const form = this.closest('form');
        if (form?.hasAttribute('hx-post') || form?.hasAttribute('hx-put') || form?.hasAttribute('hx-patch')) {
          window.htmx?.trigger(form, 'submit');
        } else {
          form?.submit();
        }
      } else if (action === 'cancel') {
        window.location.href = this.#config.cancelUrl;
      }
    }

    // Keep existing image upload handler for compatibility
    async handleImageUpload(file, dialog) {
      const statusEl = document.getElementById('upload-status');
      const urlInput = document.getElementById('image-url');
      const altInput = document.getElementById('image-alt');

      try {
        statusEl.style.display = 'block';
        statusEl.textContent = 'Uploading...';
        statusEl.className = 'upload-status uploading';

        const formData = new FormData();
        formData.append('image', file);

        const response = await fetch('/api/images/upload', {
          method: 'POST',
          body: formData
        });

        const result = await response.json();

        if (result.success) {
          urlInput.value = result.dataUri;
          if (!altInput.value) {
            altInput.value = file.name.replace(/\.[^/.]+$/, '');
          }
          statusEl.textContent = 'Upload successful!';
          statusEl.className = 'upload-status success';
          document.getElementById('image-upload').value = '';
        } else {
          throw new Error(result.message || 'Upload failed');
        }
      } catch (error) {
        statusEl.textContent = `Error: ${error.message}`;
        statusEl.className = 'upload-status error';
      }
    }
  }

  customElements.define('markdown-toolbar', MarkdownToolbar);
})();
