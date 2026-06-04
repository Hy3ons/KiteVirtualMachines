# Kite Frontend Design Convention

이 문서는 Kite 대시보드(Frontend) 개발 시 준수해야 할 UI/UX 및 디자인 규칙을 정의합니다.

## 1. UI Framework & Philosophy
- **Framework**: Ant Design (AntD)
- **Philosophy**: 전문적인 VM 및 인프라 관리 도구(Professional Tool)로서의 무게감과 신뢰감을 줍니다. 불필요한 애니메이션이나 화려한 장식을 배제하고, 효율적인 정보 전달과 조작 편의성에 집중합니다.

## 2. Theme & Color Palette
- **Theme Mode**: Bright (Light) Mode
- **Color Concept**: 크림(Cream) 및 베이지 톤 기반. 눈이 편안하면서도 클래식하고 정돈된 엔지니어링 도구의 느낌을 줍니다.
  - **Main Background**: `#F9F8F6` (전체 앱의 메인 바탕색)
  - **Sub Background / Cards**: `#EFE9E3` (컨텐츠 영역, 카드 배경 등)
  - **Borders / Dividers**: `#D9CFC7` (영역 구분선, 컴포넌트 테두리)
  - **Primary / Active**: `#C9B59C` (버튼 활성화, 주요 강조 색상)
  - **Text Colors**: 완전한 검은색 대신 크림 톤과 자연스럽게 어울리는 짙은 차콜/브라운 사용
    - Primary Text: `#2C251F` (기본 텍스트, 제목 등)
    - Secondary Text: `#594E46` (보조 텍스트, 부가 설명 등)
  - **Semantic / Status Colors**: VM 상태(Running, Stopped 등) 표시에 사용하며, 전체 테마의 톤앤매너를 유지하도록 채도를 약간 낮춘(Muted) 톤 사용
    - Success (Running 등): `#7A9C74` (Muted Green)
    - Warning (Starting, Pending 등): `#D4A373` (Muted Orange/Yellow)
    - Error (Stopped, Failed 등): `#C86B6B` (Muted Red)

## 3. Shape & Geometry
- **Border Radius**: `0px` (완전한 직각)
- 타협 없는 완전한 평면/직각 디자인을 추구합니다. Ant Design의 기본 테마를 덮어씌워 버튼, 인풋 창, 모달, 카드 등 **모든 UI 컴포넌트에서 둥근 모서리(rounded)를 완전히 배제**합니다.
- **Shadows & Elevation**: 기본적으로 평면(Flat) 디자인을 지향하나, 모달(Modal)이나 팝업, 드롭다운(Dropdown) 등 화면 위에 겹쳐서 떠오르는(Floating) UI 요소의 경우 시각적 깊이감을 위해 예외적으로 AntD 기본의 **부드러운 드롭 섀도우(Drop shadow)를 허용**합니다.

## 4. Typography
- 가독성이 가장 중요하므로 간결한 산세리프(Sans-serif) 폰트 패밀리를 사용합니다.
- `font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;`

## 5. Spacing & Layout
- **8px Grid System**: 컴포넌트 간의 여백(Margin, Padding) 및 레이아웃 구성 시 기본 단위를 **8px** 배수로 설정합니다.
  - 예시: `8px`, `16px`, `24px`, `32px`, `40px` 등
  - 컴포넌트 내부의 미세한 간격 조정이 필요한 경우에 한하여 예외적으로 4px 단위(4, 12, 20...)를 사용합니다.

## 6. Ant Design Theme Override 예시
향후 프로젝트에서 ConfigProvider를 통해 다음과 같이 테마를 전역적으로 덮어씌워서 사용합니다.
```typescript
{
  token: {
    // Colors
    colorPrimary: '#C9B59C',
    colorBgBase: '#F9F8F6',
    colorBorder: '#D9CFC7',
    colorTextBase: '#2C251F',
    colorTextSecondary: '#594E46',
    
    // Semantic Colors
    colorSuccess: '#7A9C74',
    colorWarning: '#D4A373',
    colorError: '#C86B6B',
    colorInfo: '#C9B59C',

    // Shape
    borderRadius: 0,
    
    // ... 추가 토큰 정의
  }
}
```
