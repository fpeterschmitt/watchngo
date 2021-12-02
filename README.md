# Watch'n'Go

 * Watch a single file
 * Watch files recursively in a directory, with an optional pattern
 * Store configuration in INI file or use only the command line
 * Run a command on modifications through `/bin/sh -c <command>` by default
 * Can output on stdout so you do whatever you want (`fswatch`-like)

## Usage

```
watchngo [-conf watchngo.ini] [-command <your command> -match <file / directory / glob pattern> [-filter <filter>] [-debug] [-output unixshell|raw|stdout] -silent]
```

The configuration file is used only when `-command` and `-filter` parameter are in use.
This makes it possible to use `watchngo` without writing a configuration file.

### Configuration

See [watchngo.sample.ini](watchngo.sample.ini) configuration example.

## Install

### Binary

Checkout the [releases](https://github.com/Leryan/watchngo/releases) binaries and put it somewhere in your `$PATH`.

Quick win with the latest release (Linux amd64 only):

```
sudo curl https://github.com/Leryan/watchngo/releases/download/v2.0.0/watchngo -Lo /usr/local/bin/watchngo
sudo chmod 755 /usr/local/bin/watchngo
```

### Build from sources

```
git clone https://github.com/Leryan/watchngo
cd watchngo

make install
```

## Bugs, questions, suggestions?

Ask [on github](https://github.com/Leryan/watchngo/issues).
