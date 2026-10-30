// designTokens mirrors the named CSS palette in styles/tokens.css for Ant Design.
// Keeping numeric radii here lets Ant Design derive its nested control geometry.
export const designTokens = {
  ink: '#151719', secondary: '#5c626c', muted: '#6b727c', canvas: '#ffffff',
  workspace: '#f7f8fa', surface: '#f1f3f5', hover: '#e9edf2', line: '#e5e8ec',
  border: '#cbd1d9', accent: '#255bd8', accentSoft: '#eaf0ff',
  success: '#217647', warning: '#986016', danger: '#bd333b',
  terminal: '#171b22', terminalInk: '#e3e8ef',
  radiusSmall: 10, radius: 16, radiusLarge: 24,
  font: '"Pretendard Variable", Pretendard, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  shadow: '0 12px 40px rgba(21,23,25,.12)',
} as const;
