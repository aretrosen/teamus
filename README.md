# teamus

Dead simple terminal music player written in go.

<a href="https://asciinema.org/a/p99SQoOJguh0XFajpVpR7Zmyt?autoplay=1"><img src="https://asciinema.org/a/p99SQoOJguh0XFajpVpR7Zmyt.png" width="836"/></a>

(Ignore the blocks in the video, I still havent figured out how to use my fonts in asciinema)

## Purpose

_Pretty Keyboard centric Terminal Music Player in Go using the popular Charm TUI, with Vim Keybindings_

Most Terminal Music Player today is either not pretty, or have the most unintuitive key bindings. This one is meant to be pretty, without being much resource intensive. Written in Go, it can search for all audio files in your Music Folder, and then Play the songs on `enter`. Check out the project, and click the question mark `?` to get more help. I love `cmus`, but it isn't pretty, so now we get `teamus`.

## Build Instructions

For now, you have to have the golang binary installed. For that, check out, check out ["How to Build and Install Go Programs"](https://www.digitalocean.com/community/tutorials/how-to-build-and-install-go-programs). After that just clone this repository, and type `go run .`. Or maybe you can build the repository using `go build`. Also, you can test this repo by using the command `go run github.com/aretrosen/teamus@latest`.

~**NOTE:** This was made in a Linux environment, with a Music directory. There are plans to add directories via json/yaml file later, but for now you **need** to have the `$HOME/Music` directory.~
As mentioned by @LeonardsonCC, this works perfectly fine on Windows, and now it does have the capability to scan other directories which is configured as shown below.

## Configuration

You can specify a list of song directories by creating a JSON file at `$HOME/.config/teamus/teamus.json` or `$HOME/.teamus.json`.
File example:

```json
{
  "directories": ["/home/user/Music", "/home/user/My_Musics"]
}
```

## To-dos

- [ ] Seamless playing, with shuffle
- [ ] Repeat Music / Playlist
- [x] Choose files from multiple directories, specified in json.
- [ ] Load from database, refresh only on key press.

### Additional Features

- [ ] show lyrics on key press.
- [ ] mouse support. Currently, not implemented as I don't use mouse much. If anybody does though, let me know. Also, I am happy to take contributions!!!
- [ ] play `m4a` audio. Currently, even not all `opus` files run correctly. I might have to write many of the audio frameworks from scratch.

## Collaboration

You're welcome to write features and report issues for this project.
