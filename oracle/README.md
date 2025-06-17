# GURU Oracle Daemon

GURU Oracle Daemon은 GURU 블록체인 네트워크에서 외부 데이터를 안전하고 신뢰성 있게 수집하여 온체인으로 전송하는 분산형 오라클 시스템입니다. 이 시스템은 실시간 이벤트 처리와 고성능 워커 풀을 통해 확장 가능한 오라클 서비스를 제공합니다.

## 🏗️ 시스템 아키텍처

### Core Components

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Oracle Daemon                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │
│  │  Daemon Core    │  │ Subscribe Mgr   │  │      Job Manager            │  │
│  │                 │  │                 │  │                             │  │
│  │ • System Init   │  │ • Event Sub     │  │ • Worker Pool Management    │  │
│  │ • Component     │  │ • Real-time     │  │ • Job Scheduling            │  │
│  │   Coordination  │  │   Monitoring    │  │ • CPU-based Scaling         │  │
│  │ • Lifecycle     │  │ • Event Filter  │  │                             │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────────────────────────────────────┐   │
│  │   Executor      │  │              Tx Manager                         │   │
│  │                 │  │                                                 │   │
│  │ • HTTP Request  │  │ • Transaction Creation                          │   │
│  │ • JSON Parsing  │  │ • Sequence Management                           │   │
│  │ • Data Extract  │  │ • Broadcasting                                  │   │
│  └─────────────────┘  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GURU Blockchain Network                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │
│  │  Oracle Module  │  │   Event System  │  │       State Machine         │  │
│  │                 │  │                 │  │                             │  │
│  │ • Request Reg   │  │ • Event Emit    │  │ • State Update              │  │
│  │ • Data Verify   │  │ • Subscriber    │  │ • Consensus Process         │  │
│  │ • Result Store  │  │   Notification  │  │ • Finality Guarantee        │  │
│  │                 │  │ • Log Generate  │  │                             │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
External API ──┐
               │
               ▼
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│    HTTP Request     │───▶│    JSON Parsing     │───▶│   Data Extraction   │
│                     │    │                     │    │                     │
│ • URL Call          │    │ • Response Parse    │    │ • Path-based        │
│ • Header Setup      │    │ • Structure Verify  │    │ • Value Extract     │
│ • Timeout Manage    │    │ • Error Handling    │    │ • Type Convert      │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
                                                                      │
                                                                      ▼
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│ Blockchain Submit   │◀───│ Transaction Build   │◀───│   Result Queue      │
│                     │    │                     │    │                     │
│ • Broadcasting      │    │ • Message Compose   │    │ • Result Store      │
│ • Response Handle   │    │ • Signature Process │    │ • Queue Management  │
│ • Error Recovery    │    │ • Gas Calculation   │    │ • Order Guarantee   │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
```

## 🔄 이벤트 처리 시스템

### 이벤트 타입 및 처리 방식

#### 1. Register Event (오라클 요청 등록)
```go
// 이벤트 구독 쿼리
registerQuery := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgRegisterOracleRequestDoc'"

// 처리 과정
1. 블록체인에서 새로운 오라클 요청 등록 감지
2. 요청 문서에서 계정 목록 확인
3. 현재 데몬이 할당된 계정인지 검증
4. 엔드포인트 URL과 파싱 규칙 추출
5. Job 객체 생성 및 워커 풀에 전달
```

#### 2. Update Event (오라클 요청 업데이트)
```go
// 이벤트 구독 쿼리
updateQuery := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgUpdateOracleRequestDoc'"

// 처리 과정
1. 기존 오라클 요청의 업데이트 감지
2. 변경된 설정 정보 추출
3. 활성 작업 목록에서 해당 작업 찾기
4. 작업 설정 업데이트 (URL, 파싱 규칙, 주기 등)
```

#### 3. Complete Event (데이터 수집 완료)
```go
// 이벤트 구독 쿼리
completeQuery := "tm.event='NewBlock' AND complete_oracle_data_set.request_id EXISTS"

// 처리 과정
1. 새 블록에서 완료 이벤트 감지
2. 요청 ID와 논스 정보 추출
3. 활성 작업 목록에서 해당 작업 확인
4. 논스 동기화 및 다음 수집 주기 준비
```

### 이벤트 필터링 및 라우팅

```go
func (sm *SubscribeManager) Subscribe() []*types.Job {
    select {
    case event := <-sm.subscriptions[registerMsg]:
        // 계정 필터링: 현재 데몬에 할당된 작업만 처리
        if !sm.filterAccount(event, oracletypes.EventTypeRegisterOracleRequestDoc) {
            return nil
        }
        return types.MakeJobs(event)
    
    case event := <-sm.subscriptions[updateMsg]:
        return types.MakeJobs(event)
    
    case event := <-sm.subscriptions[completeMsg]:
        return types.MakeJobs(event)
    
    case <-sm.ctx.Done():
        return nil
    }
}
```

## ⚙️ 워커 풀 시스템

### CPU 기반 동적 스케일링

```go
// CPU 코어 수에 따른 워커 생성
func (jm *JobManager) Start(ctx context.Context, resultQueue chan<- *types.JobResult) {
    for i := 0; i < runtime.NumCPU(); i++ {
        jm.wg.Add(1)
        go jm.worker(ctx, resultQueue)
    }
}
```

### Job Processing Pipeline

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│     Job Queue       │───▶│    Worker Pool      │───▶│   Result Queue      │
│                     │    │                     │    │                     │
│ • Job Waiting       │    │ • CPU Core Based    │    │ • Result Collection │
│ • Buffer Management │    │ • Parallel Process  │    │ • Order Guarantee   │
│ • Backpressure Ctrl │    │ • Error Handling    │    │ • Transaction Ready │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
```

### 작업 상태 관리

```go
type JobType byte

const (
    Register JobType = iota  // 새 작업 등록
    Update                   // 기존 작업 업데이트  
    Complete                 // 데이터 수집 완료
)

// 작업 라이프사이클
Register → Update (선택적) → Complete → Register (반복)
```

## 🔧 초기 설정

### 1. 환경 준비

#### 시스템 요구사항
```bash
# Go 버전 확인 (1.21 이상 권장)
go version

# 필수 의존성 설치
sudo apt-get update
sudo apt-get install -y build-essential git curl

# 포트 확인 (기본값)
# - RPC: 26657
# - gRPC: 9090
# - REST: 1317
```

#### 디렉토리 구조 생성
```bash
# 기본 데몬 디렉토리 (자동 생성됨)
mkdir -p ~/.oracled
mkdir -p ~/.oracled/keyring-test

# 로그 디렉토리 권한 설정
chmod 755 ~/.oracled
```

### 2. 키링 설정

#### 새 키 생성
```bash
# 테스트 키링에 새 키 생성
gurud keys add oracle-node --keyring-backend test --keyring-dir ~/.oracled/keyring-test

# 키 정보 확인
gurud keys show oracle-node --keyring-backend test --keyring-dir ~/.oracled/keyring-test
```

#### 기존 키 가져오기
```bash
# 니모닉으로 키 복구
gurud keys add oracle-node --recover --keyring-backend test --keyring-dir ~/.oracled/keyring-test

# 키스토어 파일에서 가져오기
gurud keys import oracle-node keyfile.json --keyring-backend test --keyring-dir ~/.oracled/keyring-test
```

### 3. 설정 파일 구성

#### 자동 설정 파일 생성
```bash
# 첫 실행 시 기본 설정 파일 자동 생성
./oracled

# 생성된 설정 파일 위치
~/.oracled/config.toml
```

#### 수동 설정 파일 편집
```toml
# ~/.oracled/config.toml

# RPC endpoint for connecting to the blockchain node  
rpc_endpoint = "http://localhost:26657"

# Chain ID for the blockchain network
chain_id = "guru_3110-1"

# Directory path for keyring storage (absolute path recommended)
keyring_dir = "/home/user/.oracled/keyring-test"

# Key name for signing transactions
key_name = "oracle-node"

# Gas price for transactions (format: amount + denomination)
gas_price = "630000000000aguru"

# Gas limit for transactions
gas_limit = 30000
```

### 4. 네트워크 연결 확인

#### RPC 연결 테스트
```bash
# 노드 상태 확인
curl -s http://localhost:26657/status | jq .

# 블록 높이 확인
curl -s http://localhost:26657/status | jq .result.sync_info.latest_block_height

# 네트워크 정보 확인
curl -s http://localhost:26657/net_info | jq .result.n_peers
```

#### 계정 잔액 확인
```bash
# 계정 잔액 조회
gurud query bank balances $(gurud keys show oracle-node -a --keyring-backend test --keyring-dir ~/.oracled/keyring-test)

# 가스비 지불을 위한 최소 잔액 필요 (예: 1000000aguru)
```

## 🚀 실행 방법


#### 기본 실행
```bash
./oracled
```

#### 커스텀 설정으로 실행
```bash
./oracled \
  --daemon-dir /custom/path \
  --keyring-backend test \
  --rpc-endpoint http://validator.guru.network:26657 \
  --chain-id guru_3110-1 \
  --key-name my-oracle-key \
  --gas-price 630000000000aguru \
  --gas-limit 50000
```
