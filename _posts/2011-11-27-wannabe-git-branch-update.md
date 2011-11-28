---
layout: post
title: "wannabe-git branch update"
---

This is my first posting on replican-sync, there is probably some catching up 
to do wrt to explaining the project, how I've gotten here, etc etc. but I'm 
just going to jump right into the latest stuff for now.

For almost a month I've been developing in the wannabe-git branch. 
The self-deprecating name hints at the nature of the development, but 
it deserves some explanation.

I'm working on automatic synchronization between folders. I want to keep the 
master branch of usable high-quality, and it's not up to that standard yet.

I've broken down automatic synchronization into several subtasks; all of it 
in the replican/track package.

### Checkpoint logging

This is where the 'wannabe-git' branch name comes from. We're already 
making directories, files, and blocks addressable by their hash. The next step 
is addressing the state of a directory structure at a given point in time.

I swear, my intention with the checkpoint log is not to create yet another DVCS 
(as interesting as that might be).

I need a checkpoint log to keep track of which commits have been merged from 
other folders. Replican peers will fetch the log summary from a remote, 
find the last merged commit, and then pull everything after that merge to 
synchronize.

Conflicts happen. I plan to auto-resolve by creating multiple, 
timestamped versions of the file, and let the user resolve it later. As a side 
note to myself, I need to figure out how to handle these conflict-created files 
on subsequent syncs... will that be a problem?

The checkpoint log will not actually store any of the data it describes. The 
"working copy" of the actual directory is the only source of the actual content.

However, this raises the interesting possibility of future support for backups.
What is a backup, but just another block store?

### Polling/watching

After getting the checkpoint log in a basically functional state, I realized I 
need to monitor the filesystem for changes and append them to the checkpoint 
log. Currently I have developed a Poller task which periodically scans a 
directory structure for changes in mtime, and reports on the files and 
directories which have changed.

This is a very simple filesystem monitor. I plan to supplement with 
inotify/winfsnotify/fsnotify -based solutions later.

### Tracking

I think of the tracker as the "tree-keeper". It keeps an up-to-date index of 
the filesystem: recieving filesystem updates from the Poller and incrementally
updating an index representation of the content (fs.Dir).

I've just gotten the tracker basically functional this weekend.

### Putting it all together

This week's development will be integrating all these pieces into a fully 
automatic filesystem synchronizer.

The synchronizer will start a poller and a tracker. When the tracker updates 
its index, it'll compare it against the checkpoint log head. If they differ, 
"auto-commit" the changes and append the updated index to the log.

Merges between peers will need to silence the tracker and poller, so that we 
don't trip over ourselves in the middle of a merge as the local filesystem is 
changing.

