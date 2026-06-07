import type { ThemeConfig } from 'antd';

export const kiteTheme: ThemeConfig = {
  token: {
    colorPrimary: '#8B7355',
    colorInfo: '#C9B59C',
    colorBgBase: '#F9F8F6',
    colorBorderSecondary: '#EFE9E3',
    colorBorder: '#D9CFC7',
    colorTextBase: '#2C2A29',
    colorTextSecondary: '#594E46',
    colorSuccess: '#7A9C74',
    colorWarning: '#D4A373',
    colorError: '#C86B6B',
    borderRadius: 0,
    fontFamily: "'Pretendard Variable', Pretendard, -apple-system, BlinkMacSystemFont, system-ui, Roboto, sans-serif",
    boxShadow: '0 4px 16px rgba(44, 37, 31, 0.06)',
    boxShadowSecondary: '0 12px 32px rgba(44, 37, 31, 0.10)',
    controlHeight: 44,
  },
  components: {
    Layout: {
      bodyBg: '#F9F8F6',
      headerBg: '#FFFFFF',
    },
    Card: {
      colorBgContainer: '#FFFFFF',
      boxShadowTertiary: '0 4px 12px rgba(44, 37, 31, 0.04)',
    },
    Table: {
      headerBg: '#F9F8F6',
      headerColor: '#8B7355',
      rowHoverBg: '#FDFCFB',
      borderColor: '#EFE9E3',
    },
    Button: {
      borderRadius: 0,
    }
  }
};
