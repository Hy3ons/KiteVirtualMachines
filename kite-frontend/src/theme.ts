import type { ThemeConfig } from 'antd';

export const kiteTheme: ThemeConfig = {
  token: {
    colorPrimary: '#8B7355', // A deeper, richer primary color (classic premium brown)
    colorInfo: '#C9B59C',
    colorBgBase: '#F9F8F6',
    colorBorderSecondary: '#EFE9E3',
    colorBorder: '#D9CFC7',
    colorTextBase: '#2C2A29', // Softer, deeper dark grey instead of #333
    borderRadius: 2, // Slight 2px radius looks much more premium than harsh 0px
    fontFamily: "'Pretendard Variable', Pretendard, -apple-system, BlinkMacSystemFont, system-ui, Roboto, sans-serif",
    boxShadow: '0 8px 24px rgba(0,0,0,0.04)', // Soft, premium shadow
    boxShadowSecondary: '0 16px 40px rgba(0,0,0,0.08)', // Deeper shadow for popups
    controlHeight: 44, // Taller buttons and inputs for a spacious, high-end feel
  },
  components: {
    Layout: {
      bodyBg: '#F9F8F6',
      headerBg: '#FFFFFF',
    },
    Card: {
      colorBgContainer: '#FFFFFF',
      boxShadowTertiary: '0 4px 12px rgba(0,0,0,0.03)', 
    },
    Table: {
      headerBg: '#F9F8F6', // Slight offset for table headers
      headerColor: '#8B7355',
      rowHoverBg: '#FDFCFB',
      borderColor: '#EFE9E3',
    },
    Button: {
      borderRadius: 2,
    }
  }
};
