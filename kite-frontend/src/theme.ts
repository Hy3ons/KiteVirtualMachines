import type { ThemeConfig } from 'antd';
import { designTokens as t } from './design/tokens';

export const kiteTheme: ThemeConfig = {
  token: {
    colorPrimary: t.ink, colorInfo: t.accent, colorLink: t.accent,
    colorBgBase: t.canvas, colorBgLayout: t.workspace, colorBgContainer: t.canvas,
    colorBorderSecondary: t.line, colorBorder: t.border, colorTextBase: t.ink,
    colorTextSecondary: t.secondary, colorTextTertiary: t.muted,
    colorSuccess: t.success, colorWarning: t.warning, colorError: t.danger,
    borderRadius: t.radius, borderRadiusSM: t.radiusSmall, borderRadiusLG: t.radiusLarge,
    fontFamily: t.font, fontSize: 14, controlHeightSM: 32, controlHeight: 40, controlHeightLG: 48,
    boxShadow: t.shadow, boxShadowSecondary: t.shadow,
    wireframe: false,
  },
  components: {
    Layout: { bodyBg: t.workspace, headerBg: t.canvas },
    Card: { headerFontSize: 16, paddingLG: 24, boxShadowTertiary: 'none' },
    Table: { headerBg: t.workspace, headerColor: t.secondary, rowHoverBg: t.accentSoft, borderColor: t.line, cellPaddingBlock: 16, cellPaddingInline: 16, headerBorderRadius: 16 },
    Button: { primaryShadow: 'none', defaultShadow: 'none', fontWeight: 500, borderRadiusLG: t.radius, paddingInline: 20, paddingInlineLG: 24, paddingInlineSM: 12 },
    Input: { activeShadow: '0 0 0 3px rgba(37,91,216,.1)', borderRadiusLG: t.radius, paddingInline: 16, paddingInlineLG: 16, paddingInlineSM: 12 },
    InputNumber: { borderRadiusLG: t.radius, paddingInline: 16, paddingInlineLG: 16, paddingInlineSM: 12 },
    Select: { borderRadiusLG: t.radius, optionPadding: '8px 12px' },
    Popover: { borderRadiusLG: t.radius },
    Tooltip: { borderRadius: t.radiusSmall },
    Dropdown: { borderRadiusLG: t.radius },
    Alert: { borderRadius: t.radius, defaultPadding: '12px 16px', withDescriptionPadding: '16px 20px', withDescriptionIconSize: 20, fontSizeLG: 14 },
    Tabs: { inkBarColor: 'transparent', itemSelectedColor: t.accent, itemHoverColor: t.ink },
    Tag: { defaultBg: t.surface, defaultColor: t.secondary },
    Statistic: { contentFontSize: 32 },
  },
};
