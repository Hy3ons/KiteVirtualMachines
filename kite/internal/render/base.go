package render

import (
	"bytes"
	"io"
	"path/filepath"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// 모든 렌더러가 공통으로 가질 기능을 정의합니다.
type BaseRenderer struct {
	TemplatePath    string
	TemplateName    string
	TemplateContent string
}

// NewRenderer creates a renderer that reads a YAML template from disk.
// path is the template file path used by template.ParseFiles.
// The returned renderer is useful for local file-based rendering flows.
func NewRenderer(path string) BaseRenderer {
	return BaseRenderer{
		TemplatePath: path,
	}
}

// NewRendererFromTemplate creates a renderer that reads a YAML template from memory.
// name is used as the template name in parse errors.
// content is the YAML template text, usually provided by go:embed.
// This function is used by render packages that must work inside a built controller binary.
func NewRendererFromTemplate(name string, content string) BaseRenderer {
	return BaseRenderer{
		TemplateName:    name,
		TemplateContent: content,
	}
}

// Render injects data into a YAML template and returns the first Kubernetes object.
// data contains the template fields used by the renderer-specific YAML file.
// The returned object is nil only when rendering or decoding fails.
// This function is used by single-resource renderers.
func (b *BaseRenderer) Render(data any) (*unstructured.Unstructured, error) {
	objects, err := b.RenderAll(data)
	if err != nil {
		return nil, err
	}

	if len(objects) == 0 {
		return nil, io.EOF
	}

	return objects[0], nil
}

// RenderAll injects data into a YAML template and decodes every YAML document.
// data contains the template fields used by the renderer-specific YAML file.
// The returned objects are unstructured Kubernetes resources, including all documents separated by ---.
// This function is used by controller reconcile code that applies multi-resource templates.
func (b *BaseRenderer) RenderAll(data any) ([]*unstructured.Unstructured, error) {
	tmpl, err := b.parseTemplate()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(&buf, 4096)
	objects := make([]*unstructured.Unstructured, 0)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		if len(obj.Object) == 0 {
			continue
		}

		objects = append(objects, obj)
	}

	return objects, nil
}

// parseTemplate builds the Go template used by BaseRenderer.
// The receiver provides either embedded template content or a file path.
// The returned template is ready to execute with renderer-specific data.
// This helper keeps RenderAll independent from how each template is loaded.
func (b *BaseRenderer) parseTemplate() (*template.Template, error) {
	if b.TemplateContent != "" {
		name := b.TemplateName
		if name == "" {
			name = "template.yaml"
		}

		return template.New(name).Parse(b.TemplateContent)
	}

	tmpl, err := template.ParseFiles(b.TemplatePath)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(b.TemplatePath)
	if parsed := tmpl.Lookup(name); parsed != nil {
		return parsed, nil
	}

	return tmpl, nil
}
