package versions

type Info struct {
	Name    string
	Version string
}

var SELECTED *Info

func (v Info) FilterValue() string { return v.Name }
