# Watch'n'Go

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

 * [ ] Multiple files on the `match` key
 * [ ] Match files with some kind of regex
 * [ ] Override the default command (`/bin/sh -c <command>`) that starts the actual command by configuration
