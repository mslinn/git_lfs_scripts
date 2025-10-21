# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Critical Coding Requirements

### Git Operations - ALWAYS USE -C FLAG

You consistently mess up the current working directory when running shell commands.
ALWAYS use `-C` option:

```go
// Correct
git.Run([]string{"-C", repoDir, "status"})

// Incorrect - will fail
git.Run([]string{"status"})
```

### Standards

- Use `spf13/pflag` (POSIX-style) not standard `flag` package
- All commands must support `-h/--help`, `-V/--version`, `-d/--debug`, `-v/--verbose`, and `-q/--quiet`
- Every method needs inline documentation
- Write unit tests for complex logic rather than reasoning through it
- Write methods and functions so they are testable.
- Test all non-trivial methods and classes
- Make atomic git commits with clear messages for every action
- Whenever more than one line of text needs to be shown to a user,
  use a multiline string and `https://github.com/lithammer/dedent` to align the strings with the text above and below it.

## Project Context

### Authoritative Specification

The website articles are the authoritative spec (code is aspirational and may not match):

- **Public**: `https://mslinn.com/git/index.html` (section: "Git Large File System")
- **Dev server**: `http://localhost:4001/git/index.html`
- **Website source**: `/var/sitesUbuntu/www.mslinn.com/collections/_git`
- **Scenarios template**: `/var/sitesUbuntu/www.mslinn.com/_includes/gitScenarios.html`

Read website articles for context.
Website text is more accurate than code.

### Development Workflow

- Multi-machine testing: `gojira` (192.168.1.183) is primary test server;
  `bear` and `camille` are clients
- Work in `claude` branch for both code and website
- Project location: `/mnt/f/work/git/git_lfs_scripts` on machine `bear`,
  on `/mnt/d/work/git/git_lfs_scripts` for machine `camille`,
  and on `/work/git/git_lfs_scripts` for machine `gojira`.

### Authorization Phases

Get approval before proceeding to next steps.

### Step 1 - Git LFS Server testing only.

Clarify requirements before making edits.

### Current - Step 2

Test all the other LFS servers,
and progressively build the test harness and reporting mechanism in Go.

### Step 3

You are not authorized to perform this step yet.
Update the articles on my website so the plan is explained to users
at a medium level of detail, and maintain consistency throughout.

### Step 4

You are not authorized to perform this step yet.
Once the documentation and the test scripts make sense to me, we will run and debug them.

### Step 5

You are not authorized to perform this step yet.
Summarize and publish the results and the source code.
