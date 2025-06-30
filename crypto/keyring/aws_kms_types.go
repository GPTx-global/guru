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
	"errors"

	"github.com/99designs/keyring"
)

const (
	// BackendKMS는 AWS KMS를 사용하는 keyring backend 이름
	BackendKMS = "kms-aws"

	// KMS 키 설정 상수
	KMSKeySpec       = "ECC_SECG_P256K1" // secp256k1 곡선
	KMSKeyUsage      = "SIGN_VERIFY"     // 서명 및 검증용
	SigningAlgorithm = "ECDSA_SHA_256"   // ECDSA SHA-256 서명

	// KMS 태그
	KMSTagApplication = "Application"
	KMSTagKeyType     = "KeyType"
	KMSTagCreatedBy   = "CreatedBy"

	// 환경 변수
	EnvAWSRegion    = "AWS_REGION"
	EnvVaultAddress = "VAULT_ADDR"
	EnvVaultToken   = "VAULT_TOKEN"
)

// KMSKeyInfo AWS KMS 키 정보를 담는 구조체
type KMSKeyInfo struct {
	UID       string `json:"uid"`
	KeyID     string `json:"key_id"`
	Alias     string `json:"alias,omitempty"`
	Region    string `json:"region"`
	CreatedAt string `json:"created_at"`
}

// Item 99designs/keyring.Item 인터페이스 구현
type Item struct {
	Key         string
	Data        []byte
	Label       string
	Description string

	// KMS 특화 필드
	KMSKeyInfo *KMSKeyInfo `json:"kms_key_info,omitempty"`
}

// 에러 정의
var (
	ErrKeyNotFound     = keyring.ErrKeyNotFound
	ErrKMSNotAvailable = errors.New("AWS KMS 서비스를 사용할 수 없습니다")
	ErrInvalidKeyID    = errors.New("유효하지 않은 KMS 키 ID입니다")
	ErrSigningFailed   = errors.New("KMS 서명에 실패했습니다")
	ErrKeyCreation     = errors.New("KMS 키 생성에 실패했습니다")
	ErrRegionNotSet    = errors.New("AWS 리전이 설정되지 않았습니다")
	ErrInvalidConfig   = errors.New("유효하지 않은 KMS 설정입니다")
)
