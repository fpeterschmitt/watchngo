# Watch'n'Go

**WORK IN PROGRESS**

 * Watch a single file
 * Watch files recursively in a directory, with an optional pattern
 * Run a command on modifications through `/bin/sh -c <command>`

See `watchngo.sample.ini` file for more details on configuration.

## Install

```
go install github.com/Leryan/watchngo/cmd/watchngo
```

## Usage

```
watchngo [-conf watchngo.ini]
```

## Configuration

```ini
[my cool watcher]
match = only_one_file.py
command = python only_one_file.py

[my other watcher]
match = workdir
command = pip install -U workdir && run_your_tests_or_whatever
```

## TODO

 * [x] Recursive directory watching
 * [ ] Multiple files on the `match` key
 * [x] Match files using `path/filepath.Glob()`
 * [ ] Override the default command (`/bin/sh -c <command>`) that starts the actual command by configuration
 * [ ] Command interpolation: `%match`, `%filter`, `%event.file`, `%event.op`
