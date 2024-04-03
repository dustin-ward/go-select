# Go Version Selector

Version switcher for my go on z/OS dev enviroment


![Screenshot of the interface](https://github.com/dustin-ward/go-select/blob/master/images/ui.png)

## Usage:

`go-select <path-to-go-installs>`

The program will produce a `selected` file within the installs directory that can be sourced to set `GOROOT`

I use it in my `.localenv` like this:

```
### GO SELECT

if [[ $- == *i* ]]; then
    GO_DIR=/home/dustinw/GoVersions
    go-select $GO_DIR
    selected=$GO_DIR/selected
    if [[ -f $selected ]]; then
        source $selected
    fi
    if [[ -f $GOROOT/go-build-zos/bin/goz-env ]]; then
          eval $($GOROOT/go-build-zos/bin/goz-env)
    else
          echo "Couldnt find go-build-zos..."
    fi
fi
```
