My development server shows the work in progress of a subproject of my website at
`http://localhost:4001/git/index.html`.
The public version of that same page is viewable at `https://mslinn.com/git/index.html`.
The source of these websites is on this machine at
`/var/sitesUbuntu/www.mslinn.com/collections/_git`

We will be modifying the local copy of the website in a new branch called `gemini`.

On the aforementioned web page, under the heading "Git Large File System":

- Read all the articles in that section,
  except the web page "Include Testing" at `http://localhost:4001/git/5000-git-lfs-test-page.html`,
  which only exists for testing purposes.
- This subproject of my website is an unfinished work, which I would like your help with.

The test software at `https://github.com/mslinn/git_lfs_scripts` is stored locally as
`/mnt/f/work/git/git_lfs_scripts`.
We will be modifying the local copy of the `git_lfs_scripts` in a new branch called `gemini`.

- Currently, `git_lfs_scripts` is mostly written in Bash with some Ruby,
  but those languages seems like poor choices for the data collection and reporting
  that is required for this subproject.
  That code should be written in Go and use a small portable database like SQLite.
  Continue to use Bash for very short and simple scripts,
  but anything requiring non-trivial logic should be written in Go.
- None of the code has been tested; consider all of it aspirational and possibly disjointed.
  The text in the web pages is more likely to be accurate and complete than the software.
  Consider the text to be your specification.

However, the specification is imperfect.
For example, some scenarios are unlikely to work as described.
These pointless scenarios need to be culled; this includes modifications to scripts and Jekyll HTML.
The scenarios are constructed with Liquid in the file
`/var/sitesUbuntu/www.mslinn.com/_includes/gitScenarios.html`

When you read that file, notice `{% if include.show_explanation %}` this portion because it is important `{% endif %}`

I would like you to verify and complete the test plan, update the articles so the plan is explained to users at a medium level of detail, and maintain consistency throughout. Ask me questions to clarify the requirements.
Do not make any edits until we reach agreement that the requirements are properly stated.

Once the documentation and the test scripts make sense to me, I will run and debug them.
Ensure the scripts support debug output; I favor `-d` as a flag for enabling debug output.
