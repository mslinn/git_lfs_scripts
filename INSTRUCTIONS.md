# Instructions for AI assistant

## Background

My development server shows the work in progress of a subproject of my website at
`http://localhost:4001/git/index.html`.
The public version of that same page is viewable at `https://mslinn.com/git/index.html`.
The source of these websites is on this machine at
`/var/sitesUbuntu/www.mslinn.com/`, and we are mostly focused on the `collections/_git` subdirectory.
We will be modifying the local copy of the website in the branch called `claude`.

On the aforementioned web page, under the heading "Git Large File System":

- Read all the articles in that section,
  except the web page "Include Testing" at `http://localhost:4001/git/5000-git-lfs-test-page.html`,
  which only exists for testing purposes.
- This subproject of my website is an unfinished work, which I would like your help with.

The test software at `https://github.com/mslinn/git_lfs_scripts` is stored locally as
`/mnt/f/work/git/git_lfs_scripts`.
We will be modifying the local copy of the `git_lfs_scripts` in a branch called `claude`.

- Currently, `git_lfs_scripts` is mostly written in Bash with some Ruby,
  but those languages seems like poor choices for the data collection and reporting
  that is required for this subproject.
  That code should be written in Go and use a small portable database like SQLite.
  Continue to use Bash for trivial scripts, but use Go whenever practicable.
  I envision the Go project having several executables.
- None of the code has been tested; consider all of it aspirational and possibly disjointed.
  The text in the web pages is more likely to be accurate and complete than the software.
  Consider the text to be your specification.

However, the specification is imperfect.
For example, some scenarios are unlikely to work as described.
These pointless scenarios need to be culled; this includes modifications to scripts and Jekyll HTML.
The scenarios are constructed with Liquid in the file
`/var/sitesUbuntu/www.mslinn.com/_includes/gitScenarios.html`

When you read that file, pay special attention to the instructions between
`{% if include.show_explanation %}` and `{% endif %}`.



## Execution Plan

### Step 1

I would like you to begin by verifying and discussing with me how to complete a test plan just for the Git LFS Server.
Ask me questions to clarify the requirements.
Do not make any edits to documents until we reach agreement that the requirements are properly stated.
If you can, please take a quick peek at `gojira` so you are aware of the mount points.

```shell
$ ping gojira
PING gojira (192.168.1.183) 56(84) bytes of data.
64 bytes from gojira (192.168.1.183): icmp_seq=1 ttl=64 time=3.38 ms
^C
```

Once we have a version of the documents that we like,
I will tell you to go to the next step.

### Step 2

In step 2, which you are not authorized to perform yet,
we will test all the other LFS servers, and progressively build the test harness and reporting mechanism in Go.

### Step 3

In step 3, which you are not authorized to perform yet,
we will update the articles on my website so the plan is explained to users at a medium level of detail,
and maintain consistency throughout.

### Step 4

Once the documentation and the test scripts make sense to me, we will run and debug them.
Ensure the scripts support debug output.
I favor `-d` as a flag for enabling debug output.

### Step 5

Summarize and publish the results and the source code.


## Standing Orders

- Claude, you and I have collaborated very effectively on 3 nontrivial projects.
  You have been most helpful, however, you consistently mess up the current working
  directory when running shell commands. A good workaround for Git that you should ALWAYS
  use is to specify the -C option, so you can explicitly state the directory the command
  should run in. Do something similar with all other command line programs launched
  from a shell, if possible.

- Posix standards are to be respected. For command-line Go programs, this means using pflags
  instead of flags.

- Executables should verify their dependencies are available before attempting to
  utilize them. If a dependency can be installed, it should automatically be installed
  and a message shown to the user.

- All programs need a comprehensive help message, responsive to -h/--help.
  The message should include the software version.

- Go programs need a release program.
  Following are two good examples of structure, implementation, and documentation.
  These examples should be followed closely:
  - local: `/mnt/f/work/dl`; GitHub: `https://github.com/mslinn/dl`
  - local: `/mnt/f/work/git/git_tree`; GitHub: `https://github.com/mslinn/git_tree`

- All programs need comprehensive unit tests.

- Make a commit for every action you take with a  message.
