# Oracle Daemon - 고가용성 에러 처리 시스템

Oracle Daemon은 Guru Network에서 외부 데이터를 안전하고 신뢰성 있게 수집하는 시스템입니다. 이 문서는 강화된 에러 처리 시스템과 고가용성 기능에 대해 설명합니다.

## 🚀 주요 기능

### 1. 포괄적인 에러 처리
- **재시도 로직**: 지수 백오프를 사용한 스마트 재시도
- **회로 차단기**: 연속된 실패 시 시스템 보호
- **실패 복구**: 자동 복구 및 상태 관리

### 2. 고가용성 시스템
- **자동 재시작**: 프로세스 다운 시 자동 재시작
- **헬스 체크**: 실시간 시스템 상태 모니터링
- **그레이스풀 종료**: 안전한 종료 프로세스

### 3. 모니터링 및 로깅
- **구조화된 로깅**: 표준화된 로그 형식
- **상태 추적**: 세부적인 상태 정보 제공
- **성능 모니터링**: 시스템 성능 지표 수집

## 📁 프로젝트 구조

```
oracled/
├── retry/          # 재시도 및 회로 차단기 모듈
│   └── retry.go
├── health/         # 헬스 체크 시스템
│   └── health.go
├── client/         # RPC 클라이언트 및 트랜잭션 처리
│   ├── client.go   # 강화된 연결 관리
│   ├── tx.go       # 트랜잭션 재시도 로직
│   └── config.go   # 설정 관리
├── scheduler/      # 이벤트 처리 및 작업 스케줄링
│   ├── scheduler.go # 이벤트 재처리 시스템
│   └── executor.go  # HTTP 요청 재시도
├── cmd/
│   └── main.go     # 메인 애플리케이션
├── monitor.sh      # 프로세스 모니터링 스크립트
├── oracled.service # Systemd 서비스 파일
└── README.md
```

## 🔨 빌드 방법

### 1. 기본 빌드

```bash
cd oracled/cmd
go build -o oracled .
```

### 2. 프로덕션 빌드 (권장)

```bash
cd oracled/cmd
go build -ldflags="-s -w" -trimpath -o oracled .
```

**프로덕션 빌드 옵션 설명:**
- `-ldflags="-s -w"`: 디버그 정보 제거로 바이너리 크기 최소화
- `-trimpath`: 빌드 경로 정보 제거로 보안 강화

### 3. 크로스 컴파일

```bash
# Linux 64비트용
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o oracled-linux .

# ARM64 Linux용 (AWS Graviton 등)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o oracled-arm64 .

# Windows용
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o oracled.exe .
```

### 4. 빌드 확인

```bash
# 바이너리 정보 확인
ls -la oracled
file oracled

# 실행 가능성 확인
./oracled --version  # (현재는 version 옵션 없음)
```

## 🚀 실행 방식 및 권장사항

### **방식 1: 직접 실행 (개발/테스트용)**

```bash
cd oracled/cmd
./oracled
```

**✅ 장점:**
- 즉시 실행 가능
- 디버깅 용이 (직접 로그 확인)
- 개발 및 테스트에 적합
- 설정 변경 후 빠른 재시작

**❌ 단점:**
- 터미널 종료 시 프로세스 종료
- 자동 재시작 없음
- 시스템 부팅 시 자동 시작 불가
- 프로세스 감시 기능 없음

**📍 사용 시기:** 개발, 디버깅, 1회성 테스트

---

### **방식 2: 모니터 스크립트 실행 (권장 - 수동 관리)**

```bash
cd oracled
./monitor.sh monitor
```

**✅ 장점:**
- 자동 재시작 기능
- 프로세스 감시 및 복구
- 상세한 헬스 체크
- 로그 분석 및 알림
- 수동 제어 가능 (start/stop/restart)

**❌ 단점:**
- 시스템 재부팅 시 수동 시작 필요
- 터미널 세션 종료 시 모니터 중단
- 사용자 로그아웃 시 영향받을 수 있음

**📍 사용 시기:** 개발 서버, 임시 운영, 수동 관리 환경

---

### **방식 3: Systemd 서비스 (권장 - 프로덕션)** ⭐

```bash
# 서비스 설치
sudo cp oracled.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable oracled
sudo systemctl start oracled
```

**✅ 장점:**
- 시스템 부팅 시 자동 시작
- OS 레벨 프로세스 관리
- 강력한 재시작 정책
- 리소스 제한 및 보안 설정
- 로그 중앙화 (journald)
- 사용자 독립적 실행

**❌ 단점:**
- 초기 설정 복잡
- 권한 관리 필요
- 디버깅이 상대적으로 어려움

**📍 사용 시기:** 프로덕션 환경, 24/7 운영

## 🎯 시나리오별 상세 실행 방법

### **A. 개발자 로컬 테스트**

```bash
# 방법 1: 빌드 후 실행
cd oracled/cmd
go build -o oracled .
./oracled

# 방법 2: 즉시 실행
cd oracled/cmd
go run .

# 방법 3: 환경 변수와 함께 실행
export GURU_CHAIN_ID="guru_3110-1"
export GURU_RPC_ENDPOINT="http://localhost:26657"
./oracled
```

### **B. 개발 서버에서 지속 실행**

```bash
# 1. 백그라운드로 모니터 실행
cd oracled
nohup ./monitor.sh monitor > /tmp/oracle_monitor.log 2>&1 &

# 2. 상태 확인
./monitor.sh status

# 3. 로그 실시간 확인
./monitor.sh logs          # 데몬 로그
./monitor.sh monitor-logs  # 모니터 로그

# 4. 제어 명령어
./monitor.sh start    # 시작
./monitor.sh stop     # 정지
./monitor.sh restart  # 재시작
```

### **C. 프로덕션 서버 (완전 자동화)**

#### **단계 1: 환경 준비**

```bash
# 1. 설치 디렉토리 생성
sudo mkdir -p /opt/guru/oracled
sudo mkdir -p /opt/guru/.guru

# 2. 전용 사용자 생성 (보안)
sudo useradd -r -s /bin/false -d /opt/guru guru

# 3. 바이너리 및 파일 복사
sudo cp -r oracled/* /opt/guru/oracled/
sudo chown -R guru:guru /opt/guru/

# 4. 실행 권한 설정
sudo chmod +x /opt/guru/oracled/cmd/oracled
sudo chmod +x /opt/guru/oracled/monitor.sh
```

#### **단계 2: 키 설정**

```bash
# guru 사용자로 키 생성
sudo -u guru gurud keys add oracled --keyring-backend os --home /opt/guru/.guru

# 또는 기존 키 복사
sudo cp -r ~/.guru/keyring-os /opt/guru/.guru/
sudo chown -R guru:guru /opt/guru/.guru
```

#### **단계 3: Systemd 서비스 설치**

```bash
# 1. 서비스 파일 복사
sudo cp /opt/guru/oracled/oracled.service /etc/systemd/system/

# 2. 서비스 경로 수정 (필요시)
sudo nano /etc/systemd/system/oracled.service

# 3. Systemd 재로드
sudo systemctl daemon-reload

# 4. 서비스 활성화 및 시작
sudo systemctl enable oracled
sudo systemctl start oracled

# 5. 상태 확인
sudo systemctl status oracled
```

#### **단계 4: 모니터링 설정**

```bash
# 로그 확인
sudo journalctl -u oracled -f

# 서비스 상태 확인
sudo systemctl is-active oracled
sudo systemctl is-enabled oracled

# 재시작 테스트
sudo systemctl restart oracled
```

## 🔧 설정 및 환경 변수

### **필수 환경 변수**

```bash
# 블록체인 설정
export GURU_HOME="/opt/guru/.guru"          # 키 저장 위치
export GURU_CHAIN_ID="guru_3110-1"          # 체인 ID
export GURU_RPC_ENDPOINT="http://localhost:26657"  # RPC 엔드포인트

# 키 설정
export GURU_KEY_NAME="oracled"              # 키 이름
export GURU_KEY_BACKEND="os"                # 키링 백엔드

# 가스 설정
export GURU_GAS_LIMIT="200000"              # 가스 한도
export GURU_GAS_PRICE="20000000agurutestnet"  # 가스 가격
```

### **환경 변수 영구 설정**

```bash
# ~/.bashrc 또는 ~/.profile에 추가
echo 'export GURU_HOME="$HOME/.guru"' >> ~/.bashrc
echo 'export GURU_CHAIN_ID="guru_3110-1"' >> ~/.bashrc
echo 'export GURU_RPC_ENDPOINT="http://localhost:26657"' >> ~/.bashrc
source ~/.bashrc
```

### **Systemd 환경 변수 설정**

`oracled.service` 파일에서 수정:

```ini
[Service]
Environment=HOME=/opt/guru
Environment=GURU_HOME=/opt/guru/.guru
Environment=GURU_CHAIN_ID=guru_3110-1
Environment=GURU_RPC_ENDPOINT=http://localhost:26657
Environment=GURU_KEY_NAME=oracled
Environment=GURU_KEY_BACKEND=os
```

## ⚠️ 사전 요구사항 및 주의사항

### **1. 사전 준비사항**

```bash
# Go 버전 확인 (1.19+ 필요)
go version

# Guru 노드 실행 상태 확인
curl http://localhost:26657/status

# 키 존재 확인
gurud keys list --keyring-backend os

# 네트워크 연결 확인
ping 8.8.8.8
```

### **2. 키 관리**

```bash
# 새 키 생성
gurud keys add oracled --keyring-backend os

# 기존 키 가져오기
gurud keys import oracled keyfile.json --keyring-backend os

# 키 백업
gurud keys export oracled --keyring-backend os > oracled-backup.json

# 키 권한 설정
chmod 600 ~/.guru/keyring-os/*
```

### **3. 포트 및 방화벽**

```bash
# RPC 포트 확인
netstat -tlnp | grep 26657

# 방화벽 설정 (필요시)
sudo ufw allow 26657/tcp

# 프로세스 확인
ps aux | grep gurud
ps aux | grep oracled
```

### **4. 로그 위치 및 관리**

| 실행 방식 | 로그 위치 | 확인 명령어 |
|-----------|-----------|-------------|
| 직접 실행 | 터미널 출력 | - |
| 모니터 스크립트 | `/tmp/oracled.out`<br>`/tmp/oracled_monitor.log` | `./monitor.sh logs`<br>`./monitor.sh monitor-logs` |
| Systemd | journald | `sudo journalctl -u oracled -f` |

## 🎯 최종 권장사항

### **🥇 프로덕션 환경: Systemd 서비스**
```bash
# 완전 자동화된 운영
sudo systemctl enable oracled
sudo systemctl start oracled
sudo journalctl -u oracled -f
```

**이유:**
- 24/7 안정 운영 보장
- 시스템 재부팅 시 자동 시작
- OS 레벨 보안 및 리소스 관리
- 중앙화된 로그 관리

### **🥈 개발/테스트 환경: 모니터 스크립트**
```bash
# 유연성과 안정성 확보
./monitor.sh monitor
```

**이유:**
- 개발 중 유연성 + 안정성 확보
- 쉬운 디버깅 및 로그 접근
- 필요 시 즉시 중단/재시작 가능

### **🥉 임시 테스트: 직접 실행**
```bash
# 빠른 기능 검증
./oracled
```

**이유:**
- 빠른 기능 검증
- 즉시 디버깅 가능
- 단순 테스트용

## 🛡️ 에러 처리 시스템

### 재시도 전략

#### 네트워크 재시도
- **최대 시도 횟수**: 10회
- **기본 딜레이**: 2초
- **최대 딜레이**: 60초
- **승수**: 1.5배

#### 트랜잭션 재시도
- **최대 시도 횟수**: 3회
- **기본 딜레이**: 500ms
- **최대 딜레이**: 10초
- **승수**: 2.0배

### 회로 차단기

- **실패 임계값**: 5회 연속 실패
- **리셋 타임아웃**: 2-5분 (컴포넌트별 상이)
- **상태**: Closed → Open → Half-Open → Closed

### 재시도 가능한 에러

```go
// 네트워크 에러
- "connection refused"
- "timeout"
- "temporary failure"
- "network is unreachable"
- "no such host"

// 트랜잭션 에러
- "account sequence mismatch"
- "invalid sequence"
- "tx already exists in cache"

// HTTP 에러
- 5xx 서버 에러 (재시도 가능)
- 4xx 클라이언트 에러 (재시도 불가)
```

## 📊 헬스 체크 시스템

### 헬스 체크 항목

1. **RPC 연결**: 블록체인 노드 연결 상태
2. **키링 접근**: 서명 키 접근 가능성
3. **스케줄러**: 이벤트 처리 상태
4. **HTTP 클라이언트**: 외부 API 연결 상태

### 헬스 체크 주기

- **기본 간격**: 30초
- **실패 임계값**: 연속 5회 실패 시 재시작
- **최대 재시작**: 10회

## 🔍 모니터링 스크립트

### 사용법

```bash
# 기본 모니터링 시작
./monitor.sh monitor

# 서비스 제어
./monitor.sh start|stop|restart|status

# 로그 확인
./monitor.sh logs          # 데몬 로그
./monitor.sh monitor-logs  # 모니터 로그
```

### 모니터링 기능

- **프로세스 감시**: PID 기반 프로세스 상태 확인
- **로그 분석**: 에러 패턴 탐지
- **자동 재시작**: 실패 시 자동 복구
- **알림 시스템**: 색상 코딩된 상태 표시

### 모니터 스크립트 설정

```bash
# monitor.sh 파일 상단 설정 변경
MAX_RESTART_ATTEMPTS=5      # 최대 재시작 횟수
RESTART_DELAY=10           # 재시작 간 대기 시간
HEALTH_CHECK_INTERVAL=30   # 헬스 체크 주기
CONSECUTIVE_FAILURES_THRESHOLD=3  # 연속 실패 임계값
```

## 📝 로깅 시스템

### 로그 형식

```
[카테고리] 함수명: 메시지
```

### 로그 카테고리

- `[ START ]`: 함수 시작
- `[  END  ]`: 함수 완료
- `[SUCCESS]`: 성공적인 작업
- `[  WARN ]`: 경고 메시지
- `[ERROR  ]`: 에러 메시지
- `[ RETRY ]`: 재시도 시도
- `[ DELAY ]`: 딜레이 적용
- `[ HEALTH]`: 헬스 체크 결과

### 로그 파일 위치

- **데몬 로그**: `/tmp/oracled.out`
- **모니터 로그**: `/tmp/oracled_monitor.log`
- **시스템 로그**: `journalctl -u oracled` (systemd 사용 시)

## 🚨 문제 해결

### 일반적인 문제

#### 1. 빌드 실패
```bash
# Go 모듈 정리
go mod tidy
go mod download

# 캐시 정리
go clean -modcache

# 의존성 업데이트
go get -u ./...
```

#### 2. RPC 연결 실패
```bash
# 노드 상태 확인
curl http://localhost:26657/status

# 방화벽 확인
sudo ufw status

# 포트 사용 확인
netstat -tlnp | grep 26657
```

#### 3. 키링 접근 실패
```bash
# 키 존재 확인
gurud keys list --keyring-backend os

# 권한 확인
ls -la ~/.guru/

# 키 권한 수정
chmod 600 ~/.guru/keyring-os/*
```

#### 4. 시퀀스 에러
```bash
# 계정 정보 확인
gurud query auth account $(gurud keys show oracled -a)

# 수동 시퀀스 리셋 (필요시)
# 이는 자동으로 처리되므로 일반적으로 불필요
```

#### 5. 서비스 시작 실패
```bash
# 서비스 상태 확인
sudo systemctl status oracled

# 서비스 로그 확인
sudo journalctl -u oracled --no-pager

# 설정 파일 검증
sudo systemctl cat oracled

# 권한 확인
sudo -u guru /opt/guru/oracled/cmd/oracled --help
```

### 로그 분석

#### 에러 패턴 찾기
```bash
# 에러 로그 필터링
grep "ERROR" /tmp/oracled.out

# 재시도 횟수 확인
grep "RETRY" /tmp/oracled.out | wc -l

# 헬스 체크 실패 확인
grep "Health check failed" /tmp/oracled_monitor.log

# 최근 에러 확인 (systemd)
sudo journalctl -u oracled --since "1 hour ago" | grep ERROR
```

#### 성능 모니터링
```bash
# 프로세스 리소스 사용량
top -p $(pgrep oracled)

# 메모리 사용량
ps -o pid,rss,vsz,command -p $(pgrep oracled)

# 파일 디스크립터 사용량
lsof -p $(pgrep oracled) | wc -l
```

## 🔄 운영 절차

### 정상 운영

1. **시작**: `systemctl start oracled` 또는 `./monitor.sh start`
2. **모니터링**: `./monitor.sh monitor-logs`
3. **상태 확인**: `./monitor.sh status`

### 비상 상황

1. **수동 재시작**: `./monitor.sh restart`
2. **완전 정지**: `./monitor.sh stop`
3. **로그 확인**: `./monitor.sh logs`

### 업그레이드

1. **서비스 정지**: `systemctl stop oracled`
2. **바이너리 백업**: `cp /opt/guru/oracled/cmd/oracled /opt/guru/oracled/cmd/oracled.backup`
3. **바이너리 교체**: 새 버전으로 교체
4. **서비스 시작**: `systemctl start oracled`
5. **상태 확인**: 로그 및 헬스 체크 확인

## 📈 성능 튜닝

### 시스템 리소스

- **파일 디스크립터**: 65536
- **프로세스 수**: 32768
- **메모리**: 시스템 메모리의 10% 이하 사용 권장

### 네트워크 최적화

- **연결 풀링**: HTTP 클라이언트 연결 재사용
- **타임아웃**: 30초 (조정 가능)
- **재시도 간격**: 점진적 증가

## 🛡️ 보안 고려사항

### Systemd 보안 설정

- **사용자 격리**: 전용 `guru` 사용자로 실행
- **파일시스템 보호**: 읽기 전용 시스템 파티션
- **네트워크 제한**: 필요한 주소 패밀리만 허용
- **권한 제한**: 최소 권한 원칙

### 키 관리

- **키링 백엔드**: OS 키링 사용 권장
- **파일 권한**: 600 (소유자만 읽기/쓰기)
- **백업**: 안전한 위치에 키 백업

## 📞 지원

문제가 발생하거나 질문이 있는 경우:

1. **로그 확인**: 먼저 에러 로그를 확인하세요
2. **문서 참조**: 이 README의 문제 해결 섹션을 참조하세요
3. **커뮤니티**: Guru Network 개발자 커뮤니티에 문의하세요

---

**참고**: 이 시스템은 24/7 무중단 운영을 위해 설계되었지만, 정기적인 모니터링과 유지보수가 필요합니다. 