import React from 'react';
import { Layout, Space, Button } from 'antd';
import { useNavigate } from 'react-router-dom';
import { GithubOutlined, BookOutlined } from '@ant-design/icons';

const { Header } = Layout;

interface GlobalHeaderProps {
  rightContent?: React.ReactNode;
}

export const GlobalHeader: React.FC<GlobalHeaderProps> = ({ rightContent }) => {
  const navigate = useNavigate();

  return (
    <Header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 40px', borderBottom: '1px solid #EFE9E3', background: '#FFFFFF' }}>
      <div style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }} onClick={() => navigate('/')}>
        <img src="/favicon.png" alt="Kite Logo" style={{ height: '64px', objectFit: 'contain', transform: 'scale(1.5)', transformOrigin: 'left center', marginRight: '28px' }} />
        <span style={{ fontSize: '26px', fontWeight: 900, color: '#8B7355', fontFamily: '"Outfit", "Inter", "Helvetica Neue", sans-serif', letterSpacing: '-1px' }}>Kite</span>
      </div>

      <Space size="large">
        <Space size="middle" style={rightContent ? { borderRight: '1px solid #EFE9E3', paddingRight: '16px' } : {}}>
          <Button type="link" icon={<GithubOutlined />} href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" style={{ color: '#8B7355' }}>
            Repository
          </Button>
          <Button type="link" icon={<BookOutlined />} href="https://kubernetes.io/docs/home/" target="_blank" style={{ color: '#8B7355' }}>
            K8s Docs
          </Button>
        </Space>
        {rightContent}
      </Space>
    </Header>
  );
};
