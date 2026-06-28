import React from 'react';
import { Button, Space, Typography } from 'antd';
import { GithubOutlined } from '@ant-design/icons';

const { Text } = Typography;

export const GlobalFooter: React.FC = () => (
  <footer className="global-footer">
    <div className="global-footer-inner">
      <div className="global-footer-brand">
        <img src="/favicon.png" alt="Kite Logo" className="global-footer-logo" />
        <div>
          <Text strong className="global-footer-title">Kite</Text>
          <Text className="global-footer-subtitle">OpenSource Private Cloud</Text>
        </div>
      </div>

      <p className="global-footer-purpose">
        Kite는 Kubernetes와 KubeVirt 위에서 개인과 팀이 필요한 VM을 더 쉽게 빌리고 관리할 수 있도록 만든 오픈소스 도구입니다.
        작은 인프라도 공정하게 나누고, 학습과 실험의 진입 장벽을 낮추는 마음으로 만들고 있습니다.
      </p>

      <Space size={16} wrap className="global-footer-meta">
        <Button
          type="link"
          icon={<GithubOutlined />}
          href="https://github.com/Hy3ons/KiteVirtualMachines"
          target="_blank"
          className="global-footer-link"
        >
          GitHub Repository
        </Button>
        <Text className="global-footer-credit">Created by Hy3ons</Text>
        <Text className="global-footer-license">MIT License</Text>
      </Space>
    </div>
  </footer>
);
