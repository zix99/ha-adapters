package stemplate

import (
	"strings"
	"text/template"

	"github.com/google/uuid"
)

type STemplate struct {
	tmpl *template.Template
}

func New(text string) (*STemplate, error) {
	t, err := template.New(uuid.NewString()).Parse(text)
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
