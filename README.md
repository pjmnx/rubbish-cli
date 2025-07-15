# Rubbish

Process for the linux users that use the terminal o that connect to linux servers. This is a "Move to trash" process. The objective is to give the options to the user instead of  remove the file/directory directly from the file system, place the file in a repository that will rest for a period and if the user need it, can restore it. Eventually, the file will be fully remove when the period elapsed.

It's like, instead of using the command "rm FILE" the user can use "toss FILE". The command "toss" stand for "Move To Trash".

**Key features:**

- It is intended to be use as a command, for shell/terminal.
- The default time to wait for each file to be remove, is 30 days.
- Can modify the waiting time for each times at the moment of execution. Like with an option "--wait=90d" (The term "wait" can be change for an appropriate one).
- Display a log of deleted files or directory that where delete from the current directory or sub-directories. Like the log that GIT shows to de user. I should include the remaining time each file/directory has left before vanishing from the file system.
- It need a service that will process the vanishing of files after their time has elapsed. A service because it has to run automatically, without user intervention.
- If the system has a GUI or Windows manager, the service should send a system notification 3 days in advance with a summary of times that will be remove (vanish). If there is nothing to vanish, no notification will be sent.
- It also allow the user to "restore" files/directories from the trash. I should be an option "-r --restore". Here are some types of restoring modes: a particular file, to a particular date (some integration with the log), a particular point in the log. All restores should be bound to what has been sent to the trash from the working directory or sub-directory.
- The user can only "Move to Trash" or "Restore" items that he has full permissions, especially the modify permission. some types of restoring modes: a particular file, to a particular date (some integration with the log), a particular point in the log. All restores should be bound to what has been sent to the trash from the working directory or sub-directory.
- The user can only "Move to Trash" or "Restore" items that he has full permissions, especially the modify permission.
