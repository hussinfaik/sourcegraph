package app

import (
	"bytes"
	"html/template"
	"testing"

	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
)

func TestTypeSystem(test *testing.T) {
	t := template.New("")
	t = t.Funcs(tmplFuncs)
	w := new(bytes.Buffer)
	t = template.Must(template.Must(t.ParseFiles("templates/common.html")).Parse(`{{template "PersonLink" $}}`))
	err := t.Execute(w, &sourcegraph.Person{PersonSpec: sourcegraph.PersonSpec{Login: "milton"}})
	if err != nil {
		test.Error("Type error in template PersonLink")
	}
}
