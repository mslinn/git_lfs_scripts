# Git LFS Scripts

These scripts were written along with the miniseries of articles about
[Git LFS on `mslinn.com`](https://www.mslinn.com/git/5100-git-lfs.html).
The articles explain how to install and use these scripts.


## Installation

```shell
$ git clone https://github.com/mslinn/git_lfs_scripts.git
$ echo "$(pwd)/git_lfs_scripts/bin:\$PATH" >> ~/.bashrc
$ source ~/.bashrc
$ sudo ln -s "$( which ls-files )" /usr/local/bin/lfs-files
$ sudo ln -s "$( which ls-files )" /usr/local/bin/track
$ sudo ln -s "$( which ls-files )" /usr/local/bin/untrack
```
