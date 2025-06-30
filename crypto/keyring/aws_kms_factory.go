// Copyright 2022 Evmos Foundation
// This file is part of the Evmos Network packages.
//
// Evmos is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Evmos packages are distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Evmos packages. If not, see https://github.com/evmos/evmos/blob/main/LICENSE

package keyring

import (
	"fmt"
	"io"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// NewKMSKeyringFactory AWS KMS keyring 팩토리 함수
// Cosmos SDK의 keyring.New()와 호환되도록 구현
func NewKMSKeyringFactory(
	serviceName string,
	backend string,
	rootDir string,
	input io.Reader,
	cdc codec.Codec,
	opts ...keyring.Option,
) (keyring.Keyring, error) {
	if backend != BackendKMS {
		return nil, fmt.Errorf("잘못된 backend: %s, 기대값: %s", backend, BackendKMS)
	}

	// AWS 리전 환경변수에서 읽기
	region := os.Getenv(EnvAWSRegion)
	if region == "" {
		region = "us-east-1" // 기본값
	}

	// KMS wrapper 인스턴스 생성
	wrapper, err := NewKMSKeyringWrapper(region)
	if err != nil {
		return nil, fmt.Errorf("KMS keyring 초기화 실패: %w", err)
	}

	return wrapper, nil
}

// RegisterKMSBackend AWS KMS backend를 Cosmos SDK keyring에 등록
// 이 함수는 앱 초기화 시 호출되어야 함
func RegisterKMSBackend() {
	// Cosmos SDK v0.46.13에서는 custom backend 등록이 제한적이므로
	// keyring.New() 함수를 직접 확장하여 구현
	// 실제 등록은 앱 레벨에서 처리됨
}

// IsKMSBackend 주어진 backend가 KMS인지 확인
func IsKMSBackend(backend string) bool {
	return backend == BackendKMS
}

// ValidateKMSConfig KMS 설정 검증
func ValidateKMSConfig() error {
	region := os.Getenv(EnvAWSRegion)
	if region == "" {
		return fmt.Errorf("AWS_REGION 환경변수가 설정되지 않았습니다")
	}

	// 기본적인 AWS 자격증명 확인
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_PROFILE") == "" {
		return fmt.Errorf("AWS 자격증명이 설정되지 않았습니다 (AWS_ACCESS_KEY_ID 또는 AWS_PROFILE 필요)")
	}

	return nil
}
