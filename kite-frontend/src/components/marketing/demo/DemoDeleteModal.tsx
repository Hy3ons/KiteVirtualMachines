import { Modal } from 'antd';
import type { DemoSnapshot } from './demoStory';
import { DEMO_VM, demoIgnore } from './demoData';

// DemoDeleteModal displays the actual deletion wording in a story-controlled local overlay.
// snapshot controls visibility/loading; no focus is captured and no delete callback reaches the API.
export function DemoDeleteModal({ snapshot }: { readonly snapshot: DemoSnapshot }) {
  return <Modal className="demo-delete-modal" open={snapshot.deleteConfirm} title={`${DEMO_VM.name}을(를) 삭제하시겠습니까?`} centered closable={false} getContainer={false} focusable={{ trap: false, focusTriggerAfterClose: false }} okText="삭제" cancelText="취소" okButtonProps={{ danger: true, loading: snapshot.busy }} onOk={demoIgnore} onCancel={demoIgnore}>
    <p>가상머신과 연결된 데이터가 영구적으로 삭제됩니다.</p>
  </Modal>;
}
