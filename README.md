# Watch'n'Go

**WORK IN PROGRESS**

 * Watch a single file
 * Watch files recursively in a directory, with an optional pattern
 * Run a command on modifications through `/bin/sh -c <command>`

## Install

```
go get -u github.com/Leryan/watchngo/cmd/watchngo
go install github.com/Leryan/watchngo/cmd/watchngo
```

## Usage

```
watchngo [-conf watchngo.ini]
```

## Configuration

See [watchngo.sample.ini](watchngo.sample.ini) configuration example.

## TODO

 * [x] Recursive directory watching
 * [ ] Multiple files on the `match` key
 * [x] Match files using `path/filepath.Glob()`
 * [ ] Override the default command (`/bin/sh -c <command>`) that starts the actual command by configuration
 * [ ] Command interpolation: `%match`, `%filter`, `%event.file`, `%event.op`
