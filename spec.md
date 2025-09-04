# PADD Markdown Specification

## Version 1.0

This specification defines how PADD (Personal Assistant for Daily Documentation) processes, renders, and manages markdown files within its ecosystem.

---

## Table of Contents

1. [Overview](#overview)
2. [File Types and Structure](#file-types-and-structure)
3. [Header Processing](#header-processing)
4. [Content Processing Pipeline](#content-processing-pipeline)
5. [PADD-Specific Syntax](#padd-specific-syntax)
6. [Entry System](#entry-system)
7. [File Organization](#file-organization)
8. [Search and Navigation](#search-and-navigation)
9. [Goldmark Extensions](#goldmark-extensions)
10. [Template Integration](#template-integration)

---

## Overview

PADD extends standard CommonMark markdown with custom preprocessing, specialized syntax, and intelligent content management features. The system processes markdown through a multi-stage pipeline that extracts metadata, transforms custom syntax, and renders content for web display.

### Design Principles

- **Backward Compatibility**: Standard markdown works without modification
- **Progressive Enhancement**: PADD-specific features enhance but don't break basic functionality
- **File-Based**: All content stored as plain text markdown files
- **Zero Configuration**: Sensible defaults for immediate productivity

---

## File Types and Structure

### Core Files

PADD recognizes core files in the data directory root and temporal archives:

1. **`inbox.md`** - Quick capture point for all thoughts, notes, tasks
2. **`active.md`** - Current commitments and things needing attention
3. **Daily Archive** - Temporal files organized as `daily/YYYY/MM-monthname.md`
4. **Journal Archive** - Temporal files organized as `journal/YYYY/MM-monthname.md`

### Temporal Archive Structure

Daily and journal entries are automatically organized in a hierarchical temporal structure:
```

data/
├── daily/
│   └── {year}/
│       └── {MM-monthname}.md
├── journal/
│   └── {year}/
│       └── {MM-monthname}.md
```
**Examples:**
- `daily/2025/01-january.md` - January 2025 daily entries
- `journal/2025/03-march.md` - March 2025 journal entries

**Features:**
- **Automatic File Creation**: Monthly files are created automatically when entries are added
- **Current Month Routing**: `/daily` and `/journal` redirect to the current month
- **Archive Navigation**: `/daily/archive` and `/journal/archive` provide year-based browsing
- **Chronological Organization**: Entries within each monthly file maintain chronological order

### File Organization

#### Core Files (Data Directory Root)
- `inbox.md` and `active.md` remain in the root directory
- Support unlimited content and section-based organization

#### Temporal Archives
- **Location**: `daily/` and `journal/` subdirectories
- **Structure**: Year-based folders containing monthly markdown files
- **Naming**: `{MM-monthname}.md` format (e.g., `09-september.md`)
- **Navigation**: Archive pages list all available years and months

#### Resource Files
- Stored in the `resources/` directory with hierarchical organization
- Support unlimited nesting (e.g., `resources/projects/padd/notes.md`)
- Display with breadcrumb navigation
- Can reference each other via wikilinks

### File Naming Conventions

- Must contain only letters, numbers, dashes, underscores, periods, and slashes
- Automatic `.md` extension if not specified
- File IDs created by converting spaces to dashes and lowercasing

---

## Header Processing

### Title Extraction (H1)

PADD processes the first H1 header (`# Title`) specially:

1. **Extraction**: The first H1 is extracted from content during preprocessing
2. **Removal**: The H1 line is removed from rendered content to prevent duplication
3. **Display**: Used as the page title in the template `<h1>` tag
4. **Fallback**: If no H1 exists, uses the file's display name

**Example:**
```markdown
# My Important Notes

This content will be rendered, but the title above
will be used in the page header, not inline.
```


### Section Headers (H2)

H2 headers (`## Section Name`) serve dual purposes:

1. **Structure**: Organize content into logical sections
2. **Entry Targets**: Define insertion points for new entries
3. **Navigation**: Populate table of contents and section lists

**Example:**
```markdown
# Project Notes

## Ideas
- Initial concept
- Feature brainstorming

## Tasks
- [ ] Set up repository
- [ ] Write specification
```


### Subsection Headers (H3+)

H3 and deeper headers (`### Subsection`) provide additional content hierarchy:
- Used in daily.md for timestamp organization (`### 18:00:00`)
- Support unlimited nesting levels
- Included in table of contents generation

---

## Content Processing Pipeline

### Stage 1: Preprocessing (`preprocess.go`)

1. **Title Extraction**: Identifies and removes first H1 header
2. **Section Header Collection**: Catalogs all H2 headers for entry system
3. **Wikilink Processing**: Converts `[[page-name]]` to proper links
4. **Content Cleaning**: Removes extracted title line from content

### Stage 2: Goldmark Rendering (`markdown.go`)

1. **Standard Markdown**: Processes CommonMark syntax
2. **Custom Extensions**: Applies PADD-specific extensions
3. **HTML Generation**: Creates semantic HTML output

### Stage 3: Post-Processing

1. **SVG Inlining**: Embeds SVG icons directly in HTML
2. **Search Highlighting**: Adds highlight spans when searching
3. **Template Integration**: Combines with Go templates

---

## PADD-Specific Syntax

### Wikilinks

**Syntax**: `[[page-name]]`

**Processing**:
1. Converts to internal links if target exists
2. Searches both root and resources directories
3. Shows error message if target not found

**Examples**:
```markdown
See [[inbox]] for quick notes.
Check the [[projects/padd/roadmap]] for future plans.
Reference [[non-existent-page]] shows an error.
```


**Output**:
```html
See <a href="/inbox">Inbox</a> for quick notes.
Check the <a href="/resources-projects-padd-roadmap">Projects Padd Roadmap</a> for future plans.
Reference <span class="text-color danger">!! [[non-existent-page]] not found !!</span> shows an error.
```


### Icon Shortcodes

**Syntax**: `::icon-name::`

**Processing**: Custom Goldmark extension converts to SVG icons
- Supports Remix Icon set
- Falls back to embedded defaults
- Can reference user images in `data/images/icons/`

**Example**:
```markdown
::home:: Go to home page
::warning:: Important notice
```


### Enhanced Task Lists

**Syntax**: Standard markdown task syntax with enhanced rendering

```markdown
- [ ] Unchecked task
- [x] Completed task
- [X] Also completed (case insensitive)
```


**Processing**: Custom Goldmark extension adds proper checkbox styling and functionality.

---

## Entry System

### Entry Types

PADD supports three entry formatters:

1. **List Entry** (`listEntryFormatter`):
```
- Your entry text here
```


2. **Task Entry** (`taskEntryFormatter`):
```
- [ ] Your task text here
```


3. **Timestamped Entry** (`dailyEntryFormatter`):
```
- `15:04:05` Your entry text here
```


### Insertion Strategies

#### Section-Based Insertion

For most files, entries are inserted under specific H2 sections:

1. **Target Section**: Specified in `section_header` form field
2. **Default Behavior**: Inserts at top of section if found
3. **Section Creation**: Creates new section if not found
4. **Fallback**: Prepends to file top if no section specified

#### Date-Based Insertion (daily.md)

Special hierarchical insertion for daily files:

1. **Month Level**: `## January 2024`
2. **Day Level**: `### Monday, January 15, 2024`
3. **Entry Level**: `- \`15:04:05\` Entry content`

**Insertion Logic**:
1. Find or create month section
2. Find or create day subsection within month
3. Add timestamped entry to day section

### Entry Configuration

The `EntryConfig` struct defines behavior per file type:

```textmate
type EntryConfig struct {
    FileID         string                               // Target file identifier
    RedirectPath   string                              // Where to redirect after entry
    EntryFormatter func(string, time.Time) string      // How to format the entry
    SectionConfig  *SectionInsertionConfig             // Section targeting (nil = date logic)
}
```


---

## File Organization

### Directory Structure

```
data/
├── inbox.md              # Core: Quick capture
├── active.md             # Core: Current priorities  
├── daily.md              # Core: Chronological log
├── resources/            # Organized reference material
│   ├── projects/
│   │   └── padd/
│   │       ├── notes.md
│   │       └── roadmap.md
│   ├── people/
│   └── reference/
└── images/               # User images (override embedded)
    └── icons/
```


### File Access

- **Security**: Uses `os.OpenRoot` for safe file operations
- **Validation**: Prevents directory traversal attacks
- **Scope**: Restricts access to data directory tree

---

## Search and Navigation

### Search Features

1. **Full-Text Search**: Searches across all markdown files
2. **Highlighting**: Highlights matching terms in results
3. **Match Navigation**: Supports jumping between multiple matches
4. **Context Preservation**: Maintains list formatting in highlights

### Navigation Systems

1. **Core Files**: Quick access to inbox, active, daily
2. **Resource Tree**: Hierarchical browsing of resources directory
3. **Wikilinks**: Direct linking between related pages
4. **Breadcrumbs**: Path navigation for resource files
5. **Table of Contents**: Auto-generated from headers

---

## Goldmark Extensions

### Icon Extension (`extension/icon.go`)

- **Purpose**: Renders `::icon-name::` shortcodes as SVG icons
- **Integration**: Custom Goldmark AST transformation
- **Fallbacks**: Embedded defaults → user images → error display

### Tasklist Extension (`extension/tasklist.go`)

- **Purpose**: Enhanced checkbox rendering for task lists
- **Features**: Proper HTML form controls, accessibility support
- **Styling**: Integrates with KelpUI design system

---

## Template Integration

### Page Templates

PADD uses Go templates that integrate with processed markdown:

#### View Template (`view.html`)
```html
<h1>{{.Title}}</h1>                    <!-- Uses extracted H1 or filename -->
<div class="content-display">
    {{.Content}}                       <!-- Rendered markdown HTML -->
</div>
```


#### Edit Template (`edit.html`)
```html
<h1>Editing: {{.CurrentFile.Display}}</h1>
<textarea>{{.RawContent}}</textarea>   <!-- Unprocessed markdown -->
```


### Template Data

The `RenderedContent` struct provides:
```textmate
type RenderedContent struct {
    Title          string        // Extracted H1 title
    HTML           template.HTML // Processed markdown HTML
    SectionHeaders []string      // Available H2 sections
}
```


### Dynamic Features

1. **Entry Modals**: Context-aware entry forms based on file type
2. **Section Targeting**: Dropdown populated from H2 headers
3. **Conditional UI**: Different controls for different file types

---

## Implementation Notes

### Error Handling

- **Missing Files**: Graceful 404 responses
- **Invalid Syntax**: Fallback to plain text display
- **Broken Wikilinks**: Clear error indicators
- **Processing Failures**: Preserve original content

### Performance Considerations

- **Regex Compilation**: Compiled once per preprocessing session
- **SVG Inlining**: Cached and reused across requests
- **Content Processing**: Minimal parsing passes
- **Memory Usage**: Streaming processing where possible

### Extensibility

The preprocessing pipeline supports future enhancements:
- **Custom Shortcodes**: Additional syntax beyond icons and wikilinks
- **Metadata Extraction**: Future frontmatter support
- **Content Transformations**: Additional processing stages
- **Custom Extensions**: New Goldmark extensions
