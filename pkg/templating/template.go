package templating

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
)

func Render(data map[string]interface{}) ([]byte, error) {
	tmpl, err := template.New("name").Funcs(funcMap()).Option("missingkey=error").Parse("string(t.content)")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// https://github.com/kubernetes/helm/blob/dece57e0baa94abdba22c0e3ced0b6ea64a83afd/pkg/engine/engine.go
func funcMap() template.FuncMap {
	f := sprig.TxtFuncMap()
	f["def"] = func(m map[string]interface{}, key string) (interface{}, error) {
		_, ok := m[key]
		return ok, nil
	}
	delete(f, "env")
	delete(f, "expandenv")
	return f
}
