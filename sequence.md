# Kite Controller Sequence Diagrams

## 1. KiteUser Reconciler (`user-reconcile.go`)

```mermaid
sequenceDiagram
    participant Informer as KiteUser Informer
    participant Reconciler as ReconcileKiteUser
    participant API as Kubernetes API

    Informer->>Reconciler: Add/Update Event (KiteUser)
    Reconciler->>Reconciler: Validate Spec (namespace 확인)
    Reconciler->>API: Get Namespace (존재 여부 및 Kite 관리 여부 확인)
    API-->>Reconciler: Namespace Object
    Reconciler->>Reconciler: Render Resources (Namespace, NetworkPolicy, ResourceQuota)
    
    loop For each rendered resource
        Reconciler->>API: Server-Side Apply (Patch)
        API-->>Reconciler: Success
    end
    
    Reconciler->>API: Update KiteUser Status (Phase: Ready)
    API-->>Reconciler: Success
```

```mermaid
sequenceDiagram
    participant Informer as KiteUser Informer
    participant Reconciler as DeleteKiteUserResources
    participant API as Kubernetes API

    Informer->>Reconciler: Delete Event (KiteUser)
    Reconciler->>API: List KiteUsers (동일 네임스페이스 참조 여부 확인)
    API-->>Reconciler: KiteUser List
    
    alt 다른 KiteUser가 네임스페이스를 참조함
        Reconciler->>Reconciler: 삭제 중단 (유지)
    else 참조하는 다른 KiteUser가 없음
        loop For each resource EXCEPT Namespace
            Reconciler->>API: Delete Resource (NetworkPolicy, ResourceQuota)
            API-->>Reconciler: Success
        end
    end
```

## 2. Kite Namespace Reconciler (`namespace-reconcile.go`)

```mermaid
sequenceDiagram
    participant Informer as Namespace Informer
    participant Reconciler as ReconcileKiteNamespace
    participant API as Kubernetes API

    Informer->>Reconciler: Add/Update Event (Namespace)
    Reconciler->>Reconciler: Check Labels (managed-by=kite-controller 인지 확인)
    
    alt Kite가 관리하는 Namespace일 경우
        Reconciler->>API: List KiteUsers
        API-->>Reconciler: KiteUser List
        
        alt 이 Namespace를 참조하는 KiteUser가 없음 (Orphan)
            Reconciler->>API: Delete Namespace
            API-->>Reconciler: Success
        else 누군가 사용 중
            Reconciler->>Reconciler: 삭제 안 함
        end
    end
```

## 3. Kite Virtual Machine Reconciler (`machine-reconcile.go`)

```mermaid
sequenceDiagram
    participant Informer as KiteVM Informer
    participant Reconciler as ReconcileKiteVirtualMachine
    participant API as Kubernetes API

    Informer->>Reconciler: Add/Update Event
    Reconciler->>API: Add Finalizer (kite-vm-cleanup)
    
    alt spec.delete == true (사용자 삭제 요청)
        Reconciler->>API: Update Status (Phase: Deleting)
        Reconciler->>API: Delete KubeVirt VM & Owned Resources
        Reconciler->>API: Remove Finalizer & Delete CRD
    else DeletionTimestamp 존재 (Kubernetes 레벨 삭제)
        Reconciler->>API: Delete KubeVirt VM & Owned Resources
        Reconciler->>API: Remove Finalizer
    else 정상 구동 로직
        Reconciler->>Reconciler: Validate Spec
        Reconciler->>API: Get Global Config (domainPrefix 확인)
        Reconciler->>Reconciler: Render (DataVolume, CloudInit, VM, Service, Ingress)
        
        loop For each rendered resource
            Reconciler->>API: Server-Side Apply (Patch)
        end
        
        Reconciler->>API: Update KiteVirtualMachine Status (Provisioning)
    end
```

## 4. KubeVirt Virtual Machine Status Reconciler (`kubevirt-status-reconcile.go`)

```mermaid
sequenceDiagram
    participant Informer as KubeVirt VM Informer
    participant Reconciler as ReconcileKubeVirtVirtualMachineStatus
    participant API as Kubernetes API

    Informer->>Reconciler: Add/Update/Delete Event (KubeVirt VM)
    Reconciler->>Reconciler: Check Labels (kite-controller 소유 여부 확인)
    Reconciler->>API: Get Owning KiteVirtualMachine
    API-->>Reconciler: KiteVM Object
    
    Reconciler->>Reconciler: Map KubeVirt Status -> Kite Status (Running, Stopped, Starting 등)
    Reconciler->>API: Update KiteVM Status (Phase, PowerState 등)
    
    alt 의도된 상태(Spec)와 실제 상태(Status)가 다를 경우
        Reconciler->>API: Trigger ReconcileKiteVirtualMachine (상태 맞추기)
    end
```

## 5. Kite VM Service Reconciler (`vm-service-reconcile.go`)

```mermaid
sequenceDiagram
    participant Informer as Service Informer
    participant Reconciler as ReconcileKiteVirtualMachineService
    participant API as Kubernetes API

    Informer->>Reconciler: Add/Update/Delete Event (Service)
    Reconciler->>Reconciler: Check Labels (kite-controller 소유 여부 확인)
    Reconciler->>API: Get Owning KiteVirtualMachine
    API-->>Reconciler: KiteVM Object
    
    Reconciler->>Reconciler: VM-owned Service drift 확인
    Reconciler->>API: Update KiteVM Status 또는 VM reconcile 유도
```

## 6. Kite VM DataVolume Reconciler (`data-volume-reconcile.go`)

```mermaid
sequenceDiagram
    participant Informer as DataVolume Informer
    participant Reconciler as ReconcileKiteVirtualMachineDataVolume
    participant API as Kubernetes API

    Informer->>Reconciler: Add/Update/Delete Event (DataVolume)
    Reconciler->>Reconciler: Check Labels (kite-controller 소유 여부 확인)
    Reconciler->>API: Get Owning KiteVirtualMachine
    API-->>Reconciler: KiteVM Object
    
    Reconciler->>Reconciler: DataVolume Phase 및 Progress(진행률) 확인
    Reconciler->>API: Update KiteVM Status (status.dataVolumePhase, progress, message)
```
