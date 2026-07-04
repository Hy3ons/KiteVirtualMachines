import React from 'react';
import { Layout } from 'antd';
import { Link, useNavigate } from 'react-router-dom';
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
          <span className="global-header-subtitle">Opensource Private Cloud</span>
        </span>
      </div>

      <div className="global-header-actions">
        <nav className={rightContent ? 'global-header-links with-session' : 'global-header-links'} aria-label="Kite resources">
          <a href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" rel="noreferrer" className="global-header-docs-link">
            <GithubOutlined />
            Repository
          </a>
          <Link to="/kite-docs" className="global-header-docs-link">
            <BookOutlined />
            Kite Docs
          </Link>
        </nav>
        <div className="global-header-session">{rightContent}</div>
      </div>
    </Header>
  );
};
