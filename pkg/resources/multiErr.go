package resources

import "strings"

type MultiErr struct {
	Errors []string
}

func (mer *MultiErr) Error() string {
	return "product installation errors : " + strings.Join(mer.Errors, ":  ")
}

//Add an error to the collection
func (mer *MultiErr) Add(err error) {
	if mer.Errors == nil {
		mer.Errors = []string{}
	}
	mer.Errors = append(mer.Errors, err.Error())
}
