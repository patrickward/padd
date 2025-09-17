# PADD - Personal Assistant for Daily Documentation

A simple, local, markdown-based personal information system for capturing and organizing thoughts, tasks, and
knowledge. PADD serves as a personal command center for managing my daily workflow and information.

### Why?

I needed a simple, distraction-free way to capture and organize my notes, tasks, and ideas without relying on complex
software or cloud services. Plus, I needed an excuse to practice with Go's new `os.OpenRoot` approach for safe file
handling, and I wanted to try out the KelpUI framework for building a web-based interface.

> This is a personal project and not intended to be a full-featured application. It's designed to be simple,
> lightweight, and easy to use for my needs. It works for me, but it probably won't work for you. I'm okay with all the
> limitations and trade-offs.

## Core Concepts

PADD uses a simple capture → process → execute → store workflow with core files and organized archives:

- **inbox.md**: Quick capture point for notes, ideas, and tasks - dump everything here without thinking
- **active.md**: A dashboard of current commitments - what needs attention now
- **Daily Archive**: Temporal archive organized by year and month (e.g., `data/daily/2025/01-january.md`) -
  chronological record of activities
- **Journal Archive**: Personal reflections organized by year and month (e.g., `data/journal/2025/01-january.md`) -
  deeper thoughts and insights
- **resources/**: Organized reference material by topic - where processed information lives long-term

## Encryption Features

PADD supports encryption of files using the [age](https://github.com/FiloSottile/age) encryption library. 

While Go has excellent support for encryption, it also requires the user to consider other security and file handling
concerns, such as key management and the format of the encrypted file. Age does a lot of that for you
and it's well maintained by people much smarter than I am when it comes to cryptography. In addition, by using the `age`
library to encrypt files, the files themselves are not tied to `padd`. You can use the `age` command line tool to
decrypt any of the files that `padd` encrypts. So, if for some reason `padd` stops working, you can still decrypt
files using the `age` command line tool. This is much simpler and more robust that building a `padd` specific approach
to encryption.

### Encryption Configuration

You enable encryption by creating a pair of public and private key files and specifying their locations.

Use the `-generate-keys` flag to generate a new pair of keys within the keys directory.

The `keys-dir` flag is used to specify the directory where the public and private keys are stored. Or you can set the
`PADD_KEYS_DIR` environment variable. If neither the flag nor the environment variable are set, PADD will attempt to
find the keys directory in the default data directory location (e.g., `~/.local/share/padd/keys`).

The `identity` flag is used to specify the path to the identity file. Or, you can set the `PADD_IDENTITIES_FILE`
environment variable. Note that this can be used to specify a single identity file outside of the keys directory.

The `recipient` flag is used to specify the path to the recipient file. Or, you can set the `PADD_RECIPIENTS_FILE`
environment variable. Note that this can be used to specify a single recipient file outside of the keys directory.

If no identity file or recipient file is specified, PADD will attempt to find the default identity and recipient files
in the `keys-dir` directory. The default files are called `key.pub` for the recipient file and `key.txt` for the
identities file.

Note that both the identity and recipient files can contain multiple keys. Each line in the file represents a key. Empty
lines and lines starting with `#` are ignored.

## Using Encryption

- In a markdown file, set the `encrypted` metadata field to `true` to encrypt the file.

```markdown
---
encrypted: true
---

# My Secret Note
```

This will tell the application to encrypt the file using the recipient file public keys when saving the file.

When loading an encrypted file, the application will attempt to decrypt the file using any of the private keys in the
identities file.

## Workflow

1. Everything starts in `inbox.md` - capture first, organize later
2. During regular processing, move items from inbox to either:
    - `active.md` (needs action)
    - `resources/` (reference material)
    - trash (no longer needed)
3. Work from `active.md` as the command center
4. When done, move notes to the appropriate `resources/` folder
5. Use the **Daily Archive** for chronological logging and the **Journal Archive** for personal reflections
    - Both automatically organize entries by year and month
    - Navigate through `/daily/archive` and `/journal/archive` to browse historical entries

## Task Management

PADD provides interactive task management within any markdown file. Tasks use standard markdown checkbox syntax and a
few additional features:

- **Interactive Checkboxes**: Click to toggle completion status
- **In-line Editing**: Edit task text directly in the browser
- **Automatic Timestamping**: Completed tasks get `@done(YYYY-MM-DD)` tags
- **Task Archiving**: Move completed tasks from any file to a current daily log entry
- **Individual Operations**: Edit, delete, or toggle individual tasks

### Task Syntax

```markdown
- [ ] Uncompleted task
- [x] Completed task @done(2025-01-15)
```

### Task Archiving

The "Archive Completed" feature moves all completed tasks from a file to the current day's daily log. This keeps active
task lists clean while preserving a record of what was accomplished.

Future enhancement: move to the daily log file associated with the @done date.

Archived tasks appear in daily logs with source context:

```markdown
## Tuesday, March 4, 2025

### 14:32:15

**Archived completed tasks** (from Active Tasks):

- Fix login bug @done(2025-03-01)
- Update project plan @done(2025-03-02)
```

## Temporal Archive System

PADD automatically organizes daily and journal entries in a temporal archive structure:

```
data/
├── daily/
│   └── 2025/
│       ├── 01-january.md
│       ├── 02-february.md
│       └── ...
├── journal/
│   └── 2025/
│       ├── 01-january.md
│       ├── 02-february.md
│       └── ...
├── inbox.md
├── active.md
└── resources/
└── ...
```

- **Automatic Organization**: When you add daily or journal entries, they're saved to the appropriate monthly file
- **Current Month Access**: Visiting `/daily` or `/journal` redirects to the current month's file
- **Archive Navigation**: Use `/daily/archive` and `/journal/archive` to browse all available entries by year and month
- **Monthly Files**: Each month gets its own file (e.g., `01-january.md`, `02-february.md`)

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
4. Open <http://localhost:8080> in your browser

## There are some Makefile targets to help with development and local installation:

```text
Usage of make:
  version              print the version of the padd application
  help                 print this help message
  audit                run quality control checks
  test                 run all tests
  test/cover           run all tests and display coverage
  upgradeable          list direct dependencies that have upgrades available
  tidy                 tidy modfiles and format .go files
  build                build the padd application
  run                  run the padd application
  run/live             run the application with reloading on file changes
  install              install the padd application using go install
  install-service      install the service management script
  install-all          install both the application and service script
  service-start        start the padd service
  service-stop         stop the padd service
  service-restart      restart the padd service
  service-status       show padd service status
  update-and-restart   install updated binary and restart service
```

## Data Directory Configuration

PADD uses a tiered approach to determine where to store markdown files:

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
-data, -d string    Directory to store markdown files
-port, -p int       Port to run the server on (default 8080)
-addr, -a string    Address to bind the server to (default "localhost")
-version, -v       Show version information
-help, -h          Show help message
```

## Image and SVG Handling

Images and SVGs can be placed in the "images/" directory within the data directory. Then, reference them in your
markdown files like so:

```markdown
    ![Alt text](/images/my-image.png)
```

There are some default embedded images. See the source code for details. The embedded images can be overridden by
placing files with the same name in the "images/" directory. You reference them the same way, with user-defined images
taking precedence.

For example, the `heart-fill.svg` icon is embedded by default.

```markdown
![Heart Icon](/images/icons/heart-fill.svg)
```

To override it, place your own `heart-fill.svg` in the `images/` directory.

### Icons

PADD includes a set of default icons located in the `images/icons/` directory of the source. You can use these icons in
your markdown files by referencing them with the appropriate path. You can also place your own icons in the
`images/icons/` directory within your data directory to override the defaults or add new ones.

For example, to use the Heart icon, you can include it in your markdown like this:

```markdown
![Heart Icon](/images/icons/heart-fill.svg)
```

### SVGs are Inlined

PADD inlines SVG files for better performance and styling flexibility. When you reference an SVG file in your markdown
using markdown syntax, PADD will embed the SVG content directly into the HTML output instead of linking to it as an
external file. This allows for easier customization with CSS and ensures that the SVG scales properly with your layout.

For example, if you have an SVG file named `example.svg` in your `images/` directory, you can include it in your
markdown like this:

```markdown
![Example](/images/example.svg)
```

PADD will inline the SVG content when rendering the markdown, allowing you to style it with CSS as needed.

### Icon Shortcodes

Because icons often need some styling and used frequently, PADD supports a simple shortcode syntax for including icons
in your markdown files. The shortcode format is as follows: `::icon-name::`. This will render the specified icon with
default styling by looking for the corresponding SVG file in the `images/icons/` directory, either in the embedded
resources or your data directory.

For example, to include the Heart icon, you can use the following shortcode in your markdown:

```markdown
 This is an icon: ::heart-fill::
```

That will get converted to:

```html
 This is an icon: <span class="icon"><svg ...>...</svg></span>
```

## WikiLink Shortcodes

PADD supports a simple wiki-style link syntax for linking between markdown files. The shortcode format is as follows:
`[[page-name]]`. This will create a link to the corresponding markdown file in your data directory, converting spaces to
hyphens and making the link lowercase.

It currently assumes the target file exists as a core file (inbox.md, active.md, daily.md) or in the resources
directory. Future enhancements may include more robust handling of nested directories and non-existent files, but this
works for now. It also assumes you are using the normalized naming convention (lowercase, letters, and numbers,
with hyphens) for your markdown files. You do not have to add the `.md` extension in the link. It will be added
automatically. You also do not need to add the `resources/` prefix for files in that directory. It will first search
the core files, then the resources directory.

If a file does not exist, it will show a red error message where the link would be, but it will not break the rest of
the markdown rendering.

For example, to link to a page named "Project Ideas" in resources, you can use the following shortcode in your markdown:

```markdown
    See more details in [[project-ideas]].
```

Which will get converted to the following before rendering:

```markdown
    See more details in [Project Ideas](/resources/project-ideas).
```

To link to a nested directory, just add the path in the link:

```markdown
    See more details in [[projects/website-redesign]].
```

This will get converted to the following before rendering:

```markdown
    See more details in [Projects/Website Redesign](/resources/projects/website-redesign).
```

To link to a core file, just use the name:

```markdown
    See tasks in [[active]].
```

Which will get converted to the following before rendering:

```markdown
    See tasks in [Active](/active). 
```

## Metadata

Markdown files can include optional YAML front matter for metadata. This is useful for setting titles, dates, and other
properties that can be used in rendering and organization. The currently supported metadata fields are:

- `title`: Sets the title of the page. If not provided, the filename (without extension) will be used as the title or
  the first H1 header if present.
- `description`: A brief description of the content.
- `category`: A category label for the content.
- `status`: A status label (e.g., "in-progress", "completed").
- `priority`: A priority label (e.g., "high", "medium", "low").
- `due_date`: A due date for tasks or projects.
- `created_at`: The creation date of the document.
- `updated_at`: The last updated date of the document.
- `author`: The author of the document.
- `tags`: A list of tags associated with the document.
- `contexts`: A list of contexts associated with the document.

### Status and Priority Colors

There are default colors associated with common status and priority values. You can customize these colors in the source
code if desired.

The colors use the KelpUI color scheme. See [KelpUI](https://kelpui.com/) for details.

| Status      | Color           |
|-------------|-----------------|
| draft       | neutral muted   |
| in-progress | primary muted   |
| review      | secondary muted |
| completed   | success muted   |
| complete    | success muted   |
| on-hold     | warning muted   |
| held        | warning muted   |
| cancelled   | danger muted    |
| canceled    | danger muted    |

| Priority | Color         |
|----------|---------------|
| low      | success muted |
| medium   | warning muted |
| high     | danger muted  |

There are some single color options available as well:

| Due           | Color           |
|---------------|-----------------|
| due_color     | danger muted    |
| tag_color     | primary muted   |
| context_color | secondary muted |

### Overriding or Adding New Colors

If a `metadata.json` file is found within the user data directory, it can be used to override or add new status and
priority colors. The file should contain a JSON object with `status_colors` and `priority_colors` mappings.

Example `metadata.json`:

```json
{
  "status_colors": {
    "in-review": "info muted",
    "blocked": "danger muted"
  },
  "priority_colors": {
    "urgent": "danger muted",
    "normal": "primary muted"
  },
  "due_color": "danger muted"
}
```

## Limitations

See [limitations.md](limitations.md)

## Possible Future Enhancements

- Enhanced search functionality (currently uses a very simple "contains" search across all markdown files)
- Tagging and linking between notes
- Custom Theme support
- Export to other formats (PDF, HTML)
- Synchronization options (e.g., Git integration, cloud backup)
- Automated reminders for tasks in `active.md`
- Collaboration features for shared notes
- Move the preprocess step for wikilinks and headers to a proper markdown extension
- Move the svg processing to a proper goldmark extension
- Make each line in a file editable in place? Would require more JavaScript, but could be useful for quick edits without
  leaving the page or having to enter full edit mode.
- Make lines and sections (H2) draggable for easy reordering of tasks and notes. This would also require more
  JavaScript, but could greatly enhance the usability of each file as a command center.
- Add ability to reorder date sections within a temporal file (daily/journal). This is currently a
  manual process, but could be automated with a button or command. For example, if, for some reason, August 3rd is
  listed before August 30th, a "reorder" button could reorder the sections correctly.
- Add tests for the various components and functions.
- Task templates and recurring tasks
- Cross-file task dependencies and relationships
- When archiving completed tasks, move them to the daily entry associated with the `@done(YYYY-MM-DD)` tag.

## Credits

- CSS framework: [KelpUI](https://kelpui.com)
- Markdown rendering: [Goldmark](https://github.com/yuin/goldmark)
- Embedded icons: [Remix Icon](https://remixicon.com/)
