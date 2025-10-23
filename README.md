# Git LFS Scripts

Generally useful scripts for Git LFS.

These scripts were written along with the miniseries of articles about
[Git LFS on `mslinn.com`](https://www.mslinn.com/git/5100-git-lfs.html).
The articles explain how to install and use these scripts.


## Commands

* `delete_github_repo` Deletes the given GitHub repo without prompting
* `giftless` Run `giftless` server
* `git_lfs_trace` A Git client extension that reports activity between the Git client and the Gif LFS server.
* `ls-files` Frontend for `git ls-files`
* `new_bare_repo` Creates a bare repo
* `nonlfs` lists files that are not in Git LFS
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
