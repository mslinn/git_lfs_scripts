# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Critical Coding Requirements

### Git Operations - ALWAYS USE -C FLAG

You consistently mess up the current working directory when running shell commands. ALWAYS use `-C` option:

```go
// Correct
git.Run([]string{"-C", repoDir, "status"})

// Incorrect - will fail
git.Run([]string{"status"})
```

### Standards

- Use `spf13/pflag` (POSIX-style) not standard `flag` package
- All commands must support `-h/--help`, `-V/--version`, `-d/--debug`
- Every method needs inline documentation
- Write unit tests for complex logic rather than reasoning through it
- Make atomic git commits with clear messages for every action

## Project Context

### Authoritative Specification

The website articles are the authoritative spec (code is aspirational and may not match):

- **Public**: `https://mslinn.com/git/index.html` (section: "Git Large File System")
- **Dev server**: `http://localhost:4001/git/index.html`
- **Website source**: `/var/sitesUbuntu/www.mslinn.com/collections/_git`
- **Scenarios template**: `/var/sitesUbuntu/www.mslinn.com/_includes/gitScenarios.html`

Read website articles for context. Website text is more accurate than code.

### Development Workflow

- Work in `claude` branch for both code and website
- Project location: `/mnt/f/work/git/git_lfs_scripts` (or `/mnt/d/work/git/git_lfs_scripts`)
- Multi-machine testing: gojira (192.168.1.183) is primary test server; bear/camille are clients

### Authorization Phases

**Current: Step 1** - Git LFS Server testing only. Clarify requirements before making edits. Get approval before proceeding.

**Steps 2-5** - Not authorized yet (other LFS servers, website updates, debugging, publishing)
