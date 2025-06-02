# Guru Oracle Daemon

Guru 체인을 위한 완전 자동화된 Oracle 서비스 데몬입니다.

## 개요

Oracle Daemon은 다음과 같은 완전 자동화된 워크플로우를 제공합니다:

1. **이벤트 모니터링**: Guru 네트워크에서 발생하는 블록체인 이벤트를 실시간으로 감지
2. **이벤트 필터링**: Oracle 관련 이벤트 타입을 확인하여 Scheduler로 전달
3. **이벤트 파싱**: Scheduler에서 이벤트를 파싱하여 외부 데이터 수집 작업(Job) 생성
4. **Job 관리**: 생성된 Job들의 스케줄링 및 중복 방지 관리
5. **Job 실행**: Executor가 외부 API에서 데이터를 수집하고 정제
6. **트랜잭션 생성**: 수집된 데이터를 포함한 트랜잭션 생성 및 서명
7. **네트워크 전송**: 생성된 트랜잭션을 Guru 네트워크에 전송

## 아키텍처

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Guru Chain    │───▶│     Client      │───▶│   Scheduler     │
│   (Blockchain)  │    │ (Event Monitor) │    │ (Event Parser)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                                              │
         │                                              ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Tx Builder    │◀───│  Oracle Data    │◀───│    Executor     │
│                 │    │   Processor     │    │ (Job Processor) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │ External APIs   │
                                               │                 │
                                               └─────────────────┘
```

### 주요 컴포넌트 및 책임 분리

#### 1. Client (`client/`)
**책임**: 이벤트 수집 및 필터링
- **RPC 연결 관리**: Guru 체인과의 WebSocket 연결
- **이벤트 모니터링**: 블록 및 트랜잭션 이벤트 실시간 감지
- **이벤트 필터링**: Oracle 관련 이벤트 타입만 확인하여 Scheduler로 전달
- **트랜잭션 처리**: Oracle 데이터를 포함한 트랜잭션 생성 및 전송

#### 2. Scheduler (`scheduler/`)
**책임**: 이벤트 파싱 및 Job 관리
- **이벤트 파싱**: 수신된 이벤트에서 Oracle 요청 추출
- **Job 생성**: 이벤트 데이터를 외부 데이터 수집 작업으로 변환
- **Job 스케줄링**: Job 큐 관리 및 실행 순서 제어
- **중복 방지**: 동일한 Job의 중복 실행 방지
- **Job 상태 관리**: 활성 Job 추적 및 모니터링

#### 3. Executor (`scheduler/executor.go`)
**책임**: Job 실행 및 데이터 처리
- **외부 API 호출**: Job에 정의된 외부 데이터 소스에서 데이터 수집
- **데이터 정제**: 수집된 데이터의 형식 변환 및 검증
- **결과 처리**: Oracle 데이터 생성 및 결과 채널로 전송
- **에러 처리**: 네트워크 오류, 데이터 오류 등의 예외 상황 처리

#### 4. Event Parser (`scheduler/events.go`)
**책임**: 이벤트 분석 및 Job 변환
- **이벤트 분석**: Oracle 관련 이벤트인지 판단
- **데이터 추출**: 이벤트에서 필요한 Oracle 요청 정보 추출
- **Job 변환**: 추출된 정보를 외부 데이터 수집 작업으로 변환

#### 5. Transaction Builder (`client/tx.go`)
**책임**: 트랜잭션 생성 및 전송
- **트랜잭션 생성**: Cosmos SDK 기반 트랜잭션 구성
- **서명 처리**: 키링을 사용한 트랜잭션 서명
- **브로드캐스팅**: 네트워크로 트랜잭션 전송

## 설정

### 환경 설정
Oracle Daemon은 다음 설정을 사용합니다:

```go
type Config struct {
    rpcEndpoint  string   // RPC 엔드포인트 (기본값: "http://localhost:26657")
    chainID      string   // 체인 ID (기본값: "guru_7001-1")
    keyringDir   string   // 키링 디렉토리 (기본값: "~/.guru")
    keyName      string   // 사용할 키 이름 (기본값: "oracle")
    gasPrice     string   // 가스 가격 (기본값: "20000000000aguru")
    gasLimit     uint64   // 가스 한도 (기본값: 200000)
    oracleEvents []string // 모니터링할 Oracle 이벤트
}
```

### 키 설정
Oracle Daemon을 실행하기 전에 키링에 Oracle 키가 설정되어 있어야 합니다:

```bash
# Oracle 키 생성 (guru 바이너리 필요)
gurud keys add oracle

# 또는 기존 키 가져오기
gurud keys import oracle oracle.key
```

## 사용법

### 빌드
```bash
cd oracled
go build -o oracled ./cmd
```

### 실행
```bash
./oracled
```

### 종료
`Ctrl+C`를 눌러 안전하게 종료할 수 있습니다.

## 로그 출력 예시

```
=== Guru Oracle Daemon 시작 ===
rpc client created
rpc client started
tx builder created
Executor started
Scheduler started with event, job, and delayed job processors
Event processor started, waiting for events...
Job processor started, waiting for immediate jobs...
Delayed job processor started, waiting for delayed jobs...
Started oracle transaction processor...
Oracle Daemon이 실행 중입니다...
- Client: 2가지 이벤트 모니터링 (작업 요청 tx + Oracle 완료)
- Scheduler: 즉시 실행 vs 지연 실행 분리
- Executor: Nonce 기반 반복 실행
종료하려면 Ctrl+C를 누르세요
Started monitoring blockchain events...
새로운 블록 생성됨: 높이=12345, 해시=ABC123...
Job request event forwarded to scheduler
Job request detected: BTC-PRICE (immediate execution)
Job BTC-PRICE scheduled for immediate execution
Handling immediate job: BTC-PRICE (nonce: 0)
Executing job: BTC-PRICE (URL: https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT, nonce: 0)
Oracle data sent for job BTC-PRICE (nonce: 0): 45123.45000000
Job BTC-PRICE completed successfully (nonce: 0)
Processing oracle result: BTC-PRICE
Oracle transaction sent successfully: DEF456...

Oracle complete event detected
Oracle complete: Job BTC-PRICE nonce 0 completed, next nonce: 1
Oracle complete detected: BTC-PRICE (delayed execution after 30s)
Using next nonce from complete event: 1
Job BTC-PRICE scheduled for delayed execution (nonce: 1, delay: 30s)
Waiting 30s before executing job BTC-PRICE...
Handling delayed job: BTC-PRICE (nonce: 1)
Executing job: BTC-PRICE (URL: https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT, nonce: 1)
Oracle data sent for job BTC-PRICE (nonce: 1): 45250.30000000
Job BTC-PRICE completed successfully (nonce: 1)
Processing oracle result: BTC-PRICE
Oracle transaction sent successfully: GHI789...

Oracle complete: Job BTC-PRICE nonce 1 completed, next nonce: 2
Oracle complete detected: BTC-PRICE (delayed execution after 30s)
Using next nonce from complete event: 2
Job BTC-PRICE scheduled for delayed execution (nonce: 2, delay: 30s)
Waiting 30s before executing job BTC-PRICE...
Handling delayed job: BTC-PRICE (nonce: 2)
Executing job: BTC-PRICE (URL: https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT, nonce: 2)
Oracle data sent for job BTC-PRICE (nonce: 2): 45387.85000000
Job BTC-PRICE completed successfully (nonce: 2)
```

## 개발 상태

### 완료된 기능 ✅
- RPC 클라이언트 연결 및 이벤트 모니터링
- 이벤트 타입 필터링 및 전달 시스템
- 이벤트 파싱 및 Job 생성 시스템
- Job 스케줄링 및 큐 관리 시스템
- 외부 API 데이터 수집 및 정제
- 트랜잭션 빌더 및 서명 시스템
- 채널 기반 비동기 처리
- 안전한 종료 처리
- **개선된 책임 분리 아키텍처**

### 향후 개발 예정 🚧
- 실제 Oracle 모듈과의 통합
- 다양한 데이터 소스 지원
- 에러 복구 및 재시도 로직
- 메트릭 및 모니터링
- 설정 파일 지원

## 아키텍처 개선사항

### v3.0 개선사항 🆕
- **Scheduler와 Executor 분리**: 더욱 명확한 책임 분리
- **Job 큐 시스템**: 스케줄링과 실행의 완전 분리
- **가독성 향상**: 각 컴포넌트의 역할이 명확하게 구분
- **유지보수성 개선**: 각 컴포넌트를 독립적으로 수정 가능
- **확장성 개선**: 새로운 기능 추가 시 영향 범위 최소화

### 아키텍처 변화
**이전**: Client → Event → Scheduler(파싱+실행) → Result  
**현재**: Client → Event → Scheduler(파싱+관리) → Executor(실행) → Result

**책임 분리**:
- **Client**: 이벤트 수집 및 필터링만
- **Scheduler**: 이벤트 파싱 및 Job 관리만
- **Executor**: Job 실행 및 데이터 처리만

## 주의사항

- 현재 버전은 테스트용으로 실제 Oracle 메시지 대신 로그 출력을 사용합니다
- 실제 운영 환경에서 사용하기 전에 Oracle 모듈이 구현되어야 합니다
- 키링 보안에 주의하여 운영 환경에서는 적절한 권한 설정이 필요합니다

## 라이센스

이 프로젝트는 Guru 프로젝트의 일부로 동일한 라이센스를 따릅니다. 