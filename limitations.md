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
  will be ignored. For example, if there is a file, "foobar.md", and a directory, "foobar", the directory will be ignored.
  This is a limitation of the `os.OpenRoot` approach for safe file handling.
- **No file versioning**: Files are overwritten each time they are saved
- **No file history**: No way to view previous versions of files (use Git for version control)

### Task Management

- **No task dependencies**: Tasks are independent - no automatic handling of prerequisite relationships
- **No recurring tasks**: Repeated tasks must be created manually each time
- **No task priorities**: All tasks are treated equally - prioritization is handled through file organization
- **No cross-file task tracking**: Tasks in different files are not linked or aggregated

### Search and Navigation

- **Simple text search**: Search looks for exact text matches across all files
- **No tagging system**: Organization relies on file hierarchy and manual categorization

### Interface

- **Single session (no logins)**: No user accounts or authentication

These limitations are intentional design choices to keep PADD simple, predictable, and maintainable.

