# Watch'n'Go

 * Watch a single file
 * Watch files recursively in a directory, with an optional pattern
 * Store configuration in INI file or use only the command line
 * Run a command on modifications through `/bin/sh -c <command>`

## Install

```
go get -u github.com/Leryan/watchngo/cmd/watchngo
go install github.com/Leryan/watchngo/cmd/watchngo
```

## Usage

```
watchngo [-conf watchngo.ini] [-command <your command> -match <match> [-filter <filter>] [-debug]]
```

When using `-command -match -filter` options, configuration will be ignored. This makes it possible to use `watchngo` without writing a configuration file.

## Configuration

See [watchngo.sample.ini](watchngo.sample.ini) configuration example.

## TODO

 * [ ] Override the default command (`/bin/sh -c <command>`) that starts the actual command by configuration
