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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type KMSTestSuite struct {
	suite.Suite
}

func TestKMSTestSuite(t *testing.T) {
	suite.Run(t, new(KMSTestSuite))
}

// TestBackendConstant KMS backend 상수 테스트
func (suite *KMSTestSuite) TestBackendConstant() {
	require.Equal(suite.T(), "kms-aws", BackendKMS)
}

// TestIsKMSBackend backend 확인 함수 테스트
func (suite *KMSTestSuite) TestIsKMSBackend() {
	require.True(suite.T(), IsKMSBackend(BackendKMS))
	require.False(suite.T(), IsKMSBackend("os"))
	require.False(suite.T(), IsKMSBackend("file"))
	require.False(suite.T(), IsKMSBackend("test"))
}

// TestValidateKMSConfig KMS 설정 검증 테스트
func (suite *KMSTestSuite) TestValidateKMSConfig() {
	// 환경변수 백업
	originalRegion := os.Getenv(EnvAWSRegion)
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalProfile := os.Getenv("AWS_PROFILE")

	defer func() {
		// 환경변수 복원
		os.Setenv(EnvAWSRegion, originalRegion)
		os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		os.Setenv("AWS_PROFILE", originalProfile)
	}()

	// 모든 환경변수 제거
	os.Unsetenv(EnvAWSRegion)
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_PROFILE")

	// 리전이 없으면 실패해야 함
	err := ValidateKMSConfig()
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "AWS_REGION")

	// 리전은 있지만 자격증명이 없으면 실패해야 함
	os.Setenv(EnvAWSRegion, "us-east-1")
	err = ValidateKMSConfig()
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "AWS 자격증명")

	// ACCESS_KEY_ID가 있으면 성공해야 함
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	err = ValidateKMSConfig()
	require.NoError(suite.T(), err)

	// PROFILE이 있어도 성공해야 함
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Setenv("AWS_PROFILE", "test-profile")
	err = ValidateKMSConfig()
	require.NoError(suite.T(), err)
}

// TestKMSKeyInfo KMS 키 정보 구조체 테스트
func (suite *KMSTestSuite) TestKMSKeyInfo() {
	keyInfo := &KMSKeyInfo{
		UID:       "test-key",
		KeyID:     "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
		Alias:     "alias/test-key",
		Region:    "us-east-1",
		CreatedAt: "2023-01-01T00:00:00Z",
	}

	require.Equal(suite.T(), "test-key", keyInfo.UID)
	require.Equal(suite.T(), "us-east-1", keyInfo.Region)
	require.NotEmpty(suite.T(), keyInfo.KeyID)
}

// TestItem Item 구조체 테스트
func (suite *KMSTestSuite) TestItem() {
	keyInfo := &KMSKeyInfo{
		UID:    "test-key",
		KeyID:  "test-key-id",
		Region: "us-east-1",
	}

	item := Item{
		Key:         "test-key",
		Data:        []byte("test-data"),
		Label:       "Test Key",
		Description: "Test KMS Key",
		KMSKeyInfo:  keyInfo,
	}

	require.Equal(suite.T(), "test-key", item.Key)
	require.Equal(suite.T(), "Test Key", item.Label)
	require.NotNil(suite.T(), item.KMSKeyInfo)
	require.Equal(suite.T(), "test-key", item.KMSKeyInfo.UID)
}

// TestErrorDefinitions 에러 정의 테스트
func (suite *KMSTestSuite) TestErrorDefinitions() {
	require.NotNil(suite.T(), ErrKeyNotFound)
	require.NotNil(suite.T(), ErrKMSNotAvailable)
	require.NotNil(suite.T(), ErrInvalidKeyID)
	require.NotNil(suite.T(), ErrSigningFailed)
	require.NotNil(suite.T(), ErrKeyCreation)
	require.NotNil(suite.T(), ErrRegionNotSet)
	require.NotNil(suite.T(), ErrInvalidConfig)
}

// TestNewKMSKeyringWrapper wrapper 생성 테스트 (환경변수 의존성으로 실제 KMS 연결은 스킵)
func (suite *KMSTestSuite) TestNewKMSKeyringWrapper() {
	// 실제 AWS KMS 연결 없이 wrapper 생성 로직만 테스트
	// 환경변수가 설정되지 않은 상태에서는 기본값 사용 확인

	originalRegion := os.Getenv(EnvAWSRegion)
	defer os.Setenv(EnvAWSRegion, originalRegion)

	os.Unsetenv(EnvAWSRegion)

	// 실제 KMS 연결은 실패하지만 region 기본값 설정 로직은 확인 가능
	_, err := NewKMSKeyringWrapper("")
	require.Error(suite.T(), err)                   // 실제 AWS 연결 실패 예상
	require.Contains(suite.T(), err.Error(), "KMS") // KMS 관련 오류 메시지 확인
}
