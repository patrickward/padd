# PADD - Personal Assistant for Daily Documentation

A simple, local, markdown-based personal information system for capturing and organizing thoughts, tasks, and knowledge. PADD serves as a personal command center for managing daily workflow and information.

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
