# Go Version Selector

Version switcher for my go on z/OS dev enviroment


![Screenshot of the interface](https://github.com/dustin-ward/go-select/blob/master/images/ui.png)

## Usage:

`go-select <path-to-go-installs>`

The program will produce a `selected` file within the installs directory that can be sourced to set `GOROOT`

I use it in my `.localenv` like this:

```
### GO SELECT

function go-select() {
    if [ $# -eq 0 ]; then
        echo "No install directory provided..."
        echo "Usage: go-select /path/to/goInstalls"
        return 1
    fi

    # Install directory is the first arg
    GO_DIR=$1

    # This is the actual go-select executable written in Go...
    # I've just renamed it to avoid conflict with the name of this function
    go-select_main $GO_DIR
    selected=$GO_DIR/selected
    if [[ -f $selected ]]; then
        # Sets GOROOT
        source $selected

        # Location for final Go 'executable'
        go_wrapper=/home/dustinw/bin/go

        # Create wrapper script
        echo "export _BPXK_AUTOCVT=ON" > $go_wrapper
        if [[ -f $GOROOT/go-build-zos/bin/goz-env ]]; then
            # Do goz-env -o manually because not all versions of goz-env have -o...
            echo "eval \$(${GOROOT}/go-build-zos/bin/goz-env) > /dev/null" >> $go_wrapper
            export PATH=$GOROOT/go-build-zos/bin:$PATH
        elif [[ -f $GOROOT/go-build-zos/envsetup ]]; then
            # Create wrapper manually for 1.18 or older...
            echo "source ${GOROOT}/go-build-zos/envsetup > /dev/null" >> $go_wrapper
            export PATH=$GOROOT/go-build-zos/bin:$PATH
        else
            echo "Couldnt find go-build-zos..."
        fi
        echo "exec \"${GOROOT}/bin/go\" \"\$@\"" >> $go_wrapper
        chmod +x $go_wrapper

        rm $selected
    else
        echo "No Go version selected"
    fi
}

# Don't call if in non-interactive shell (sftp)
if [[ $- == *i* ]]; then
    go-select /home/dustinw/GoVersions
fi
```
