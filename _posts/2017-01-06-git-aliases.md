---
layout: post
title: Git Aliases
tags: code git environment bash
---

Git is great, but creating some git aliases is a great way to make your usages even more efficient.

To add any of these you can either copy and paste into the `[alias]` section of your `.gitconfig` file or run `git config --global alias.NAME 'COMMAND'` replacing `NAME` with the alias to use, and `COMMAND` with what to run.

So without further ado, here are the ones I have created and use on a very regular basis.

# Constant usage

* `git s` - an alias for `git status`.  Have to save those 5 keypresses!

  ```
  s = status
  ```

* `git cm "some commit message"` - shorthand for commit with a message

  ```
  cm = commit -m
  ```

* `git dc` - diff files staged for commit

  ```
  dc = diff --cached
  ```

* `git scrub` - deletes everything not tracked by git (`git clean -dxf`) except the `packages` and `node_modules` directories

  ```
  scrub = clean -dxf --exclude=packages --exclude=node_modules
  ```

# Context switching, rebasing on dirty HEAD

I rebase my changes onto the current branches often, but rebasing requires a clean repository to work on.  The following two aliases are used something like this: `git save && git pull --rebase && git undo`

* `git save` - adds and commits everything in the repository, with the commit message `SAVEPOINT`

  ```
  save = !git add -A && git commit -m 'SAVEPOINT'
  ```

* `git undo` - undoes the last commit, leaving everything as it was before committing.  Mostly used to undo a `git save` call

  ```
  undo = reset HEAD~1 --mixed
  ```

I also use these if I need to save my work to work on a bug fix on a different branch.

# What have I done?

Often I want commits I have pending, either to the local master, or a remote tracking branch.  These both give an output like this:

![Git Pending](/images/git-pending.png)

* `git pending` - shows the commits on the current branch compared to the `origin/master` branch

  ```
  pending = log origin/master..HEAD --pretty=oneline --abbrev-commit --format='%Cgreen%cr:%Creset %C(auto)%h%Creset %s'
  ```

* `git pendingup` - shows the commits on the current branch compared to its tracking branch

  ```
  pendingup = "!git log origin/\"$(git rev-parse --abbrev-ref HEAD)\"..HEAD --pretty=oneline --abbrev-commit --format='%Cgreen%cr:%Creset %C(auto)%h%Creset %s'"
  ```

# More?

I have some others not documented here, but are in my [config repo](https://github.com/Pondidum/config/blob/master/configs/.gitconfig) on Github.
