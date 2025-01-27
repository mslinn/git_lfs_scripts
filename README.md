# Git LFS Scripts

These scripts were written along with the miniseries of articles about
[Git LFS on `mslinn.com`](https://www.mslinn.com/git/5100-git-lfs.html).
The articles explain how to install and use these scripts.


## Commands

* `checksums` Computes and / or compares CRCs with previous values
* `delete_github_repo` Deletes the given GitHub repo without prompting
* `set_remotes` Saves `remotes.origin.url` and `lfs.url`
* `giftless` Run `giftless` server
* `git_lfs_test_data` Downloads test data and synthesizes small files
* `git_lfs_test_server` Run `Git LFS Test Server
* `ls-files` Frontend for `git ls-files`
* `new_bare_repo` Creates a bare repo
* `nonlfs` lists files that are not in Git LFS
* `setup_git_lfs_eval_server` Calls other commands to create directory structures
  for each evaluation scenario, including bare Git repositories and Git LFS
  repositories
* `setup_git_lfs_eval_client` Calls other commands to download test data and
  clone Git repositories for each scenario
* `unmigrate` Reverses `git lfs migrate import` for a given wildmatch pattern


## Installation

```shell
$ git clone https://github.com/mslinn/git_lfs_scripts.git
$ echo "$(pwd)/git_lfs_scripts/bin:\$PATH" >> ~/.bashrc
$ source ~/.bashrc
$ sudo ln -s "$( which ls-files )" /usr/local/bin/lfs-files
$ sudo ln -s "$( which ls-files )" /usr/local/bin/track
$ sudo ln -s "$( which ls-files )" /usr/local/bin/untrack
```
