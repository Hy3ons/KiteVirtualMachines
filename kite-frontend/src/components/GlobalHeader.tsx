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
    <Header className="global-header">
      <div className="global-header-brand" onClick={() => navigate('/')}>
        <img src="/favicon.png" alt="Kite Logo" className="global-header-logo" />
        <span className="global-header-divider" aria-hidden="true" />
        <span className="global-header-wordmark">
          <span className="global-header-title">Kite</span>
          <span className="global-header-subtitle">OpenSource Private Cloud</span>
        </span>
      </div>

      <Space size="large" wrap className="global-header-actions">
        <Space size="middle" wrap className={rightContent ? 'global-header-links with-session' : 'global-header-links'}>
          <Button type="link" icon={<GithubOutlined />} href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" style={{ color: '#8B7355' }}>
            Repository
          </Button>
          <Button type="link" icon={<BookOutlined />} href="https://kubernetes.io/docs/home/" target="_blank" style={{ color: '#8B7355' }}>
            K8s Docs
          </Button>
        </Space>
        <div className="global-header-session">{rightContent}</div>
      </Space>
    </Header>
  );
};
