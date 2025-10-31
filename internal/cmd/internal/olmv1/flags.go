package olmv1

import "github.com/spf13/pflag"

type getOptions struct {
	Output   string
	Selector string
}

func bindGetFlags(fs *pflag.FlagSet, o *getOptions) {
	fs.StringVarP(&o.Output, "output", "o", "", "output format. One of: (json, yaml)")
	fs.StringVarP(&o.Selector, "selector", "l", "", "selector (label query) to filter on, "+
		"supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 "+
		"in (value3)). Matching objects must satisfy all of the specified label constraints.")

}
