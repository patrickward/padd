# PADD - Personal Assistant for Daily Documentation

A simple, local, markdown-based personal information system for capturing and organizing thoughts, tasks, and knowledge. PADD serves as a personal command center for managing daily workflow and information.

### Why?

I needed a simple, distraction-free way to capture and organize my notes, tasks, and ideas without relying on complex software or cloud services. Plus, I needed an excuse to practice with Go's new `os.OpenRoot` approach for safe file handling, and I wanted to try out the KelpUI framework for building a web-based interface.

## Core Concepts

PADD uses a simple capture → process → execute → store workflow with three core files and a resources directory:

- **inbox.md**: Quick capture point for notes, ideas, and tasks - dump everything here without thinking
- **active.md**: A dashboard of current commitments - what needs attention now  
- **daily.md**: Append-only log for activities, thoughts, and reflections - a temporal record
- **resources/**: Organized reference material by topic - where processed information lives long-term

## Workflow

1. Everything starts in `inbox.md` - capture first, organize later
2. During regular processing, move items from inbox to either:
   - `active.md` (needs action)
   - `resources/` (reference material)  
   - trash (no longer needed)
3. Work from `active.md` as your command center
4. When done, move notes to the appropriate `resources/` folder
5. `daily.md` captures the journey - what happened and when - as a permanent chronological record

## Resources Organization

The `resources/` directory supports hierarchical organization:

```

resources/
├── someday.md           # Ideas and projects for the future
├── people/              # Notes about colleagues, contacts
│   └── john-smith.md
├── projects/            # Completed or reference project notes
│   ├── website-redesign.md
│   └── 2024-planning.md
├── reference/           # General reference material
│   ├── commands.md      # Useful commands and snippets
│   └── workflows.md     # Process documentation
├── learning/            # Course notes, articles, research
│   └── python-notes.md
└── meetings/            # Meeting notes and decisions
└── 2024-01-standup.md
```

## Installation and Usage

1. Clone or download the repository
2. Build the application: `go build`
3. Run the server: `./padd`
4. Open http://localhost:8080 in your browser

## Data Directory Configuration

PADD uses a tiered approach to determine where to store your markdown files:

1. **Command-line flag** (`-data`) - highest precedence
2. **Environment variable** (`PADD_DATA_DIR`) - if flag not set and variable is defined
3. **XDG standard location** - fallback to `$XDG_DATA_HOME/padd` or `$HOME/.local/share/padd`

Examples:

```bash
# Use specific directory
./padd -data /path/to/my/notes

# Use environment variable
export PADD_DATA_DIR=/path/to/my/notes
./padd

# Use default XDG location
./padd
```

## Command Line Options

```
-data string    Directory to store markdown files
-port int       Port to run the server on (default 8080)
-addr string    Address to bind the server to (default "localhost")
```

## Image and SVG Handling

Images and SVGs can be placed in the "images/" directory within the data directory. Then, reference them in your markdown files like so:

```markdown
    ![Alt text](/images/my-image.png)
```

There are some default embedded images. See the source code for details. The embedded images can be overridden by placing files with the same name in the "images/" directory. You reference them the same way, with user-defined images taking precedence.

For example, the `mastodon-fill.svg` icon is embedded by default.

```markdown
![Mastodon](/images/icons/mastodon-fill.svg)
```

To override it, place your own `mastodon-fill.svg` in the `images/` directory.

### Icons 

PADD includes a set of default icons located in the `images/icons/` directory of the source. You can use these icons in your markdown files by referencing them with the appropriate path. You can also place your own icons in the `images/icons/` directory within your data directory to override the defaults or add new ones.

For example, to use the Mastodon icon, you can include it in your markdown like this:

```markdown
![Mastodon](/images/icons/mastodon-fill.svg)
```

### SVGs are Inlined 

PADD inlines SVG files for better performance and styling flexibility. When you reference an SVG file in your markdown using markdown syntax, PADD will embed the SVG content directly into the HTML output instead of linking to it as an external file. This allows for easier customization with CSS and ensures that the SVG scales properly with your layout.

For example, if you have an SVG file named `example.svg` in your `images/` directory, you can include it in your markdown like this:

```markdown
![Example](/images/example.svg)
```

PADD will inline the SVG content when rendering the markdown, allowing you to style it with CSS as needed.

### Icon Shortcodes 

Because icons often need some styling and used frequently, PADD supports a simple shortcode syntax for including icons in your markdown files. The shortcode format is as follows: `::icon-name::`. This will render the specified icon with default styling by looking for the corresponding SVG file in the `images/icons/` directory, either in the embedded resources or your data directory.

For example, to include the Mastodon icon, you can use the following shortcode in your markdown:

```markdown
 This is an icon: ::mastodon-fill::
```

That will get converted to:

```html
 This is an icon: <span class="icon"><svg ...>...</svg></span>
```

## WikiLink Shortcodes

PADD supports a simple wiki-style link syntax for linking between markdown files. The shortcode format is as follows: `[[page-name]]`. This will create a link to the corresponding markdown file in your data directory, converting spaces to hyphens and making the link lowercase.

It currently assumes the target file exists as a core file (inbox.md, active.md, daily.md) or in the resources directory. Future enhancements may include more robust handling of nested directories and non-existent files, but this works for now. It also assumes 
you are using the normalized naming convention (lowercase, letters and numbers, with hyphens) for your markdown files. You do not have to add the `.md` extension in the link. It will be added automatically. You also do not need to add the `resources/` prefix for files in that directory. It will first search the core files, then the resources directory.

If a file does not exist, it will show a red error message where the link would be, but it will not break the rest of the markdown rendering.

For example, to link to a page named "Project Ideas" in resources, you can use the following shortcode in your markdown:

```markdown 
    See more details in [[project-ides]].
```

Which will get converted to:

```html
    See more details in [Project Ideas](/resources/project-ideas) before the markdown is rendered.
```

To link to a nested directory, just add the path in the link:

```markdown
    See more details in [[projects/website-redesign]].
```

This will get converted to:

```html
    See more details in [Projects/Website Redesign](/resources/projects/website-redesign) before the markdown is rendered.
```

To link to a core file, just use the name:

```markdown
    See my current tasks in [[active]].
```

Which will get converted to:

```html
    See my current tasks in [Active](/active) before the markdown is rendered.
```

## Possible Future Enhancements

- Enhanced search functionality (currently uses a very simple "contains" search across all markdown files)
- Tagging and linking between notes
- Custom Theme support
- Export to other formats (PDF, HTML)
- Synchronization options (e.g., Git integration, cloud backup)
- Automated reminders for tasks in `active.md`
- Collaboration features for shared notes

## Credits 

- CSS framework: [KelpUI](https://kelpui.com)
- Markdown rendering: [Goldmark](https://github.com/yuin/goldmark) 
- Embedded icons: [Remix Icon](https://remixicon.com/)
