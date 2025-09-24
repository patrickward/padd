# Known Limitations for PADD

PADD is intentionally simple and designed for single-user, local operation. Several features are deliberately omitted or
handled manually. Some of this may be improved in future versions, but many limitations are by design to keep the system
lightweight and maintainable. These are notes to myself, to remind me of the limitations and to help me think through
possible future enhancements.

### Synchronization

- **No built-in sync**: Use external tools like Syncthing, iCloud, Git, or Dropbox to sync files across devices
- **No conflict resolution**: If the same file is edited simultaneously on different devices, manual conflict resolution
  is required
- **No real-time collaboration**: Multiple users cannot edit the same files simultaneously

### File Management

- **Manual organization**: Moving items between files requires copy/paste or manual editing
- **No automatic cleanup**: Old or unused files must be deleted manually
- **Limited file validation**: PADD assumes well-formed markdown and doesn't validate file structure
- **Directories sometimes conflict with files**: If there is a file with the same name as a directory, the directory
  will be ignored. For example, if there is a file, "foobar.md", and a directory, "foobar", the directory will be
  ignored.
  This is a limitation of the `os.OpenRoot` approach for safe file handling.
- **No file versioning**: Files are overwritten each time they are saved
- **No file history**: No way to view previous versions of files (use Git for version control)

### Task Management

- **No task dependencies**: Tasks are independent - no automatic handling of prerequisite relationships
- **No recurring tasks**: Repeated tasks must be created manually each time
- **No task priorities**: All tasks are treated equally - prioritization is handled through file organization
- **No cross-file task tracking**: Tasks in different files are not linked or aggregated

### CSV File Management

- **No CSV File Creation**: Currently, you must create a new file under resources and manage it there.
- **No CSV Cell or Row Updating**: Currently, you can edit the entire file, but you can't edit a single cell or row.
- **Metadata is manual**: You must manually add or maintain metadata to a `*.csv.meta.json` file manually. That's okay,
  it works for now.

The CSV file management is simple and manual, but it's fine. For now, if I need to edit a CSV file, I can use edit the
text directly or open it in another app like Numbers.

### Search and Navigation

- **Simple text search**: Search looks for exact text matches across all files
- **No tagging system**: Organization relies on file hierarchy and manual categorization

### Interface

- **Single session (no logins)**: No user accounts or authentication

These limitations are intentional design choices to keep PADD simple, predictable, and maintainable.

