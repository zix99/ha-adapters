package stemplate

import (
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

type STemplate struct {
	tmpl *template.Template
}

var helpers = template.FuncMap{
	"now": time.Now,
}

func New(text string) (*STemplate, error) {
	t, err := template.New(uuid.NewString()).Funcs(helpers).Parse(text)
	if err != nil {
		return nil, err
	}

	return &STemplate{
		t,
	}, nil
}

func MustNew(text string) *STemplate {
	t, err := New(text)
	if err != nil {
		panic(err)
	}
	return t
}

func (s *STemplate) Execute(data any) string {
	var out strings.Builder
	err := s.tmpl.Execute(&out, data)
	if err != nil {
		panic(err)
	}
	return out.String()
}
