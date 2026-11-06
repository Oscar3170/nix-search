# Color previews

Snapshots of the CLI output with ANSI colors enabled, so the palette can be
reviewed with `cat`:

```console
$ cat examples/help.ansi
$ cat examples/pkgs.ansi
$ cat examples/options-long.ansi
```

`cat -v` shows the escape sequences instead of rendering them, and
`less -R examples/help.ansi` pages through the rendered colors.

| file                | command                                                    |
| ------------------- | ---------------------------------------------------------- |
| `help.ansi`         | `nix-search --color=always --help`                         |
| `pkgs.ansi`         | `nix-search --color=always -n 5 ripgrep`                   |
| `pkgs-long.ansi`    | `nix-search --color=always -n 3 -o long firefox`           |
| `options.ansi`      | `nix-search --color=always -n 5 nixos 'nginx enable'`      |
| `options-long.ansi` | `nix-search --color=always -n 3 -o long nixos 'nginx enable'` |
| `short.ansi`        | `nix-search --color=always -o short -n 5 pkgs ripgrep`     |

These are snapshots of the live backend, so versions and descriptions drift
over time; rerun the commands above to refresh them.

## Palette

| element                      | style            |
| ---------------------------- | ---------------- |
| package and option names      | bold             |
| package versions             | yellow           |
| provided programs            | green            |
| option types                 | magenta          |
| option defaults              | yellow           |
| descriptions                 | dim              |
| option source files          | cyan             |
| help section headers         | bold             |
| help subcommands and flags   | bold cyan        |
| help metavars                | yellow           |
| help example comments        | dim              |
