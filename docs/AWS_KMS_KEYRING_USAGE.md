# AWS KMS Keyring Backend 사용 가이드

## 개요

Guru 블록체인은 기업급 키 관리를 위해 AWS Key Management Service (KMS)를 keyring backend로 지원합니다. 이 기능은 높은 보안성과 감사 기능을 제공하며, 프로덕션 환경에서 안전한 키 관리를 가능하게 합니다.

## 특징

- ✅ **기업급 보안**: AWS KMS의 FIPS 140-2 Level 2 검증된 HSM 사용
- ✅ **감사 및 로깅**: AWS CloudTrail을 통한 모든 키 작업 로깅
- ✅ **권한 관리**: IAM을 통한 세밀한 접근 제어
- ✅ **고가용성**: AWS의 다중 AZ 인프라 활용
- ✅ **암호화**: secp256k1 곡선 지원 (Ethereum 호환)
- ✅ **비용 효율성**: 사용한 만큼만 지불하는 요금제

## 전제 조건

### 1. AWS 계정 및 자격증명 설정

```bash
# AWS CLI 설치 (macOS)
brew install awscli

# AWS 자격증명 설정
aws configure
# 또는 환경변수 설정
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

### 2. IAM 권한 설정

KMS backend 사용을 위해 다음 IAM 권한이 필요합니다:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "kms:CreateKey",
                "kms:GetPublicKey",
                "kms:Sign",
                "kms:ScheduleKeyDeletion",
                "kms:ListKeys",
                "kms:DescribeKey",
                "kms:TagResource"
            ],
            "Resource": "*"
        }
    ]
}
```

### 3. 환경변수 설정

```bash
# 필수 환경변수
export AWS_REGION="us-east-1"  # 또는 원하는 리전

# 선택적 환경변수 (AWS 자격증명이 다른 방법으로 설정되지 않은 경우)
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"

# 또는 AWS Profile 사용
export AWS_PROFILE="your-profile"
```

## 사용법

### 1. 키 생성

```bash
# AWS KMS backend를 사용하여 새 키 생성
gurud keys add my-validator-key --keyring-backend kms-aws

# 출력 예시:
# - name: my-validator-key
#   type: local
#   address: guru1abc123...
#   pubkey: '{"@type":"/ethermint.crypto.v1.ethsecp256k1.PubKey","key":"A1B2C3..."}'
```

### 2. 키 목록 조회

```bash
# 모든 키 목록 보기
gurud keys list --keyring-backend kms-aws

# 특정 키 정보 보기
gurud keys show my-validator-key --keyring-backend kms-aws
```

### 3. 트랜잭션 서명

```bash
# 트랜잭션 전송
gurud tx bank send my-validator-key guru1recipient... 1000uguru \
  --keyring-backend kms-aws \
  --chain-id guru-testnet \
  --gas auto \
  --gas-adjustment 1.2
```

### 4. 키 삭제

```bash
# 키 삭제 (AWS KMS에서는 7일 후 완전 삭제)
gurud keys delete my-validator-key --keyring-backend kms-aws
```

## 고급 사용법

### 1. 프로그래매틱 사용

```go
package main

import (
    "github.com/GPTx-global/guru/crypto/keyring"
)

func main() {
    // KMS keyring wrapper 생성
    kr, err := keyring.NewKMSKeyringWrapper("us-east-1")
    if err != nil {
        panic(err)
    }

    // 새 키 생성
    record, _, err := kr.NewMnemonic(
        "my-key",
        keyring.English,
        "m/44'/60'/0'/0/0",
        "",
        hd.EthSecp256k1,
    )
    if err != nil {
        panic(err)
    }

    // 키 정보 출력
    fmt.Printf("Key Name: %s\n", record.Name)
    fmt.Printf("Address: %s\n", record.GetAddress())
}
```

### 2. 다중 리전 설정

```bash
# 리전별로 다른 키 사용
export AWS_REGION="us-west-2"
gurud keys add west-validator --keyring-backend kms-aws

export AWS_REGION="eu-west-1"
gurud keys add eu-validator --keyring-backend kms-aws
```

### 3. 배치 스크립트 예시

```bash
#!/bin/bash
# setup-validators.sh

REGIONS=("us-east-1" "us-west-2" "eu-west-1")
VALIDATORS=("validator-1" "validator-2" "validator-3")

for i in "${!REGIONS[@]}"; do
    export AWS_REGION="${REGIONS[$i]}"
    gurud keys add "${VALIDATORS[$i]}" --keyring-backend kms-aws
    echo "Created ${VALIDATORS[$i]} in ${REGIONS[$i]}"
done
```

## 보안 고려사항

### 1. IAM 권한 최소화

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "kms:Sign",
                "kms:GetPublicKey"
            ],
            "Resource": "arn:aws:kms:us-east-1:123456789012:key/12345678-*",
            "Condition": {
                "StringEquals": {
                    "kms:ViaService": "guru-blockchain.us-east-1.amazonaws.com"
                }
            }
        }
    ]
}
```

### 2. 키 회전 정책

```bash
# 정기적인 키 회전 (예: 매년)
# 1. 새 키 생성
gurud keys add validator-2024 --keyring-backend kms-aws

# 2. 밸리데이터 키 업데이트
gurud tx staking edit-validator \
  --new-moniker "My Validator" \
  --from validator-2024 \
  --keyring-backend kms-aws

# 3. 이전 키 삭제
gurud keys delete validator-2023 --keyring-backend kms-aws
```

### 3. 모니터링 및 알림

```bash
# CloudTrail 로그 모니터링을 위한 CloudWatch 알림 설정
aws logs create-log-group --log-group-name /aws/kms/guru-blockchain

# KMS 키 사용량 모니터링
aws cloudwatch put-metric-alarm \
  --alarm-name "KMS-Key-Usage-High" \
  --alarm-description "High KMS key usage" \
  --metric-name NumberOfRequestsSucceeded \
  --namespace AWS/KMS \
  --statistic Sum \
  --period 300 \
  --threshold 1000 \
  --comparison-operator GreaterThanThreshold
```

## 문제 해결

### 1. 일반적인 오류

#### "AWS 리전이 설정되지 않았습니다"
```bash
export AWS_REGION="us-east-1"
```

#### "AWS 자격증명이 설정되지 않았습니다"
```bash
aws configure
# 또는
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
```

#### "KMS 서비스 연결 실패"
```bash
# AWS KMS 서비스 상태 확인
aws kms list-keys --region us-east-1

# 네트워크 연결 확인
ping kms.us-east-1.amazonaws.com
```

### 2. 디버깅

```bash
# 상세 로깅 활성화
export GURU_LOG_LEVEL=debug
gurud keys list --keyring-backend kms-aws

# AWS CLI 디버그 모드
aws kms list-keys --debug
```

### 3. 성능 최적화

```bash
# 리전 지연시간 최적화
# 가장 가까운 AWS 리전 사용
export AWS_REGION="ap-northeast-2"  # 한국의 경우

# 연결 풀링 설정 (프로그래매틱 사용 시)
export AWS_MAX_ATTEMPTS=3
export AWS_RETRY_MODE=adaptive
```

## 비용 관리

### 1. KMS 요금 구조

- **키 저장**: $1/월 per key
- **API 요청**: $0.03 per 10,000 requests
- **교차 리전 복제**: 추가 요금

### 2. 비용 최적화 팁

```bash
# 사용하지 않는 키 정리
aws kms list-keys --query 'Keys[?KeyState==`PendingDeletion`]'

# 키 사용량 모니터링
aws logs filter-log-events \
  --log-group-name CloudTrail \
  --filter-pattern "{ $.eventName = Sign || $.eventName = GetPublicKey }"
```

## 마이그레이션 가이드

### 기존 keyring에서 KMS로 마이그레이션

```bash
# 1. 기존 키 백업
gurud keys export my-key --keyring-backend file

# 2. KMS에 새 키 생성
gurud keys add my-key-kms --keyring-backend kms-aws

# 3. 밸리데이터 키 업데이트
gurud tx staking edit-validator \
  --from my-key-kms \
  --keyring-backend kms-aws

# 4. 기존 키 삭제 (선택사항)
gurud keys delete my-key --keyring-backend file
```

## 추가 리소스

- [AWS KMS 개발자 가이드](https://docs.aws.amazon.com/kms/latest/developerguide/)
- [Cosmos SDK Keyring 문서](https://docs.cosmos.network/main/run-node/keyring)
- [Guru 블록체인 문서](https://docs.guru.network)

## 지원

문제가 발생하거나 질문이 있는 경우:

1. [GitHub Issues](https://github.com/GPTx-global/guru/issues)
2. [Discord 커뮤니티](https://discord.gg/guru)
3. [Telegram 채널](https://t.me/guru_blockchain)

---

**주의**: AWS KMS는 실제 비용이 발생하는 서비스입니다. 테스트 환경에서는 `test` backend를 사용하시기 바랍니다. 