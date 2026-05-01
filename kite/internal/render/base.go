package render

import (
	"bytes"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// 모든 렌더러가 공통으로 가질 기능을 정의합니다.
type BaseRenderer struct {
	TemplatePath string
}

func NewRenderer (path string) BaseRenderer {
	return BaseRenderer{
		TemplatePath: path,
	}
}

// Render: 실제 템플릿에 데이터를 주입하여 Unstructured 객체로 변환합니다.
func (b *BaseRenderer) Render(data any) (*unstructured.Unstructured, error) {
	// 1. 템플릿 파일 파싱
	tmpl, err := template.ParseFiles(b.TemplatePath)
	if err != nil {
		return nil, err
	}

	// 2. 데이터 주입
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	// 3. YAML -> Unstructured 변환
	obj := &unstructured.Unstructured{}
	decoder := yaml.NewYAMLOrJSONDecoder(&buf, 4096)
	if err := decoder.Decode(obj); err != nil {
		return nil, err
	}

	return obj, nil
}