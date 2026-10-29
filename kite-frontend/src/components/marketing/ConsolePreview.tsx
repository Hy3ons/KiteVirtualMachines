import { CloudServerOutlined, CodeOutlined, FolderOutlined, SlidersOutlined } from '@ant-design/icons';

// ConsolePreview illustrates the product on the public landing page with clearly labelled examples.
// It is static presentation, so its visual navigation and resource labels are not interactive controls.
export function ConsolePreview() {
  return (
    <figure className="marketing-preview-stage">
      <div className="marketing-preview-window">
        <div className="marketing-preview-chrome"><span className="marketing-preview-wordmark">Kite <span>/ workspace</span></span><span>제품 화면 예시</span></div>
        <div className="marketing-preview-layout">
          <div className="marketing-preview-sidebar" aria-hidden="true">
            <span className="marketing-preview-nav-current"><CloudServerOutlined /> 가상머신</span>
            <span><FolderOutlined /> 내 작업 공간</span>
            <span><SlidersOutlined /> 계정 정보</span>
            <div className="marketing-preview-sidebar-note">Your infrastructure.<br />Your workspace.</div>
          </div>
          <div className="marketing-preview-body">
            <div className="marketing-preview-heading"><div><span className="marketing-eyebrow">Workspace overview</span><h2>내 가상머신</h2></div><span className="marketing-preview-label">Workspace / example</span></div>
            <div className="marketing-preview-resource">
              <div className="marketing-preview-resource-title"><span className="marketing-preview-resource-icon"><CloudServerOutlined /></span><div><strong>my-workspace</strong><span>가상머신 구성 예시</span></div><span className="marketing-preview-policy">Level 1</span></div>
              <dl className="marketing-preview-specs"><div><dt>CPU</dt><dd>2 <span>vCPU</span></dd></div><div><dt>메모리</dt><dd>4 <span>GiB</span></dd></div><div><dt>디스크</dt><dd>25 <span>GiB</span></dd></div></dl>
              <div className="marketing-preview-resource-footer"><span>계정 정책에 맞춘 리소스</span><span>생성 · 시작 · 중지</span></div>
            </div>
            <div className="marketing-preview-connection"><CodeOutlined /><div><strong>작업 공간으로 연결</strong><span>웹 콘솔과 SSH Gateway</span></div><span className="marketing-preview-connection-arrow" aria-hidden="true">↗</span></div>
          </div>
        </div>
      </div>
      <figcaption>Kite console · 화면 구성 예시이며 실제 리소스 상태가 아닙니다.</figcaption>
    </figure>
  );
}
