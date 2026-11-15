// Package semantic maps measure and dimension names to SQL. Model files are
// trusted (they ship in the binary); request values are always bound as
// parameters and never reach the SQL text.
package semantic

import (
	"embed"
	"fmt"
	"io/fs"
	"regexp"

	"gopkg.in/yaml.v3"
)

//go:embed models/*.yml
var modelFiles embed.FS

var nameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Field is one measure or dimension.
type Field struct {
	Name   string `json:"name"`
	SQL    string `json:"-"`
	Label  string `json:"label"`
	Format string `json:"format,omitempty"`
}

// Model is one queryable source.
type Model struct {
	Name       string
	Label      string
	Source     string
	TimeColumn string
	measures   []Field
	dimensions []Field
	byMeasure  map[string]Field
	byDim      map[string]Field
}

// Description is what the UI needs to build pickers.
type Description struct {
	Name       string   `json:"name"`
	Label      string   `json:"label"`
	Grains     []string `json:"grains"`
	Measures   []Field  `json:"measures"`
	Dimensions []Field  `json:"dimensions"`
}

// Describe lists fields in file order.
func (m *Model) Describe() Description {
	return Description{Name: m.Name, Label: m.Label, Grains: []string{"month", "quarter", "year"},
		Measures: m.measures, Dimensions: m.dimensions}
}

type rawModel struct {
	Name       string    `yaml:"name"`
	Label      string    `yaml:"label"`
	Source     string    `yaml:"source"`
	TimeColumn string    `yaml:"time_column"`
	Measures   yaml.Node `yaml:"measures"`
	Dimensions yaml.Node `yaml:"dimensions"`
}

// LoadEmbedded parses every model shipped in the binary, keyed by name.
func LoadEmbedded() (map[string]*Model, error) {
	out := map[string]*Model{}
	files, err := fs.Glob(modelFiles, "models/*.yml")
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		data, err := modelFiles.ReadFile(f)
		if err != nil {
			return nil, err
		}
		m, err := parse(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		out[m.Name] = m
	}
	return out, nil
}

func parse(data []byte) (*Model, error) {
	var raw rawModel
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if !nameRE.MatchString(raw.Name) || raw.Source == "" || raw.TimeColumn == "" {
		return nil, fmt.Errorf("name, source and time_column are required")
	}
	m := &Model{Name: raw.Name, Label: raw.Label, Source: raw.Source, TimeColumn: raw.TimeColumn,
		byMeasure: map[string]Field{}, byDim: map[string]Field{}}
	var err error
	if m.measures, err = fields(&raw.Measures, m.byMeasure); err != nil {
		return nil, fmt.Errorf("measures: %w", err)
	}
	if m.dimensions, err = fields(&raw.Dimensions, m.byDim); err != nil {
		return nil, fmt.Errorf("dimensions: %w", err)
	}
	if len(m.measures) == 0 {
		return nil, fmt.Errorf("at least one measure is required")
	}
	return m, nil
}

// fields reads a YAML mapping in document order.
func fields(node *yaml.Node, index map[string]Field) ([]Field, error) {
	if node.Kind != yaml.MappingNode {
		return nil, nil
	}
	var out []Field
	for i := 0; i+1 < len(node.Content); i += 2 {
		var f struct {
			SQL    string `yaml:"sql"`
			Label  string `yaml:"label"`
			Format string `yaml:"format"`
		}
		name := node.Content[i].Value
		if err := node.Content[i+1].Decode(&f); err != nil {
			return nil, err
		}
		if !nameRE.MatchString(name) || f.SQL == "" {
			return nil, fmt.Errorf("field %q needs a lowercase name and sql", name)
		}
		if f.Label == "" {
			f.Label = name
		}
		field := Field{Name: name, SQL: f.SQL, Label: f.Label, Format: f.Format}
		index[name] = field
		out = append(out, field)
	}
	return out, nil
}
