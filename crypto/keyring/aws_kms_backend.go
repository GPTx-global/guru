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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/pkg/errors"

	"github.com/GPTx-global/guru/crypto/ethsecp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// KMSKeyring AWS KMS를 사용하는 키링 백엔드
type KMSKeyring struct {
	kmsClient *kms.KMS
	keyCache  map[string]*KMSKey // UID -> KMS Key 매핑
	addrCache map[string]string  // Address -> UID 매핑
	mutex     sync.RWMutex
	region    string
}

// KMSKey AWS KMS 키 정보
type KMSKey struct {
	UID       string
	KeyID     string
	PublicKey cryptotypes.PubKey
	Address   sdk.AccAddress
}

// NewKMSKeyring 새로운 AWS KMS 키링 인스턴스를 생성합니다
func NewKMSKeyring(region string) (*KMSKeyring, error) {
	if region == "" {
		return nil, ErrRegionNotSet
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("AWS 세션 생성 실패: %w", err)
	}

	kmsClient := kms.New(sess)

	// KMS 서비스 가용성 테스트
	_, err = kmsClient.ListKeys(&kms.ListKeysInput{Limit: aws.Int64(1)})
	if err != nil {
		return nil, errors.Wrap(err, "KMS 서비스 연결 실패")
	}

	return &KMSKeyring{
		kmsClient: kmsClient,
		keyCache:  make(map[string]*KMSKey),
		addrCache: make(map[string]string),
		region:    region,
	}, nil
}

// Backend 백엔드 타입 반환
func (k *KMSKeyring) Backend() string {
	return BackendKMS
}

// Get 키 조회 (99designs/keyring 인터페이스)
func (k *KMSKeyring) Get(key string) (Item, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	if kmsKey, exists := k.keyCache[key]; exists {
		// 메타데이터를 JSON으로 직렬화
		keyInfo := &KMSKeyInfo{
			UID:       kmsKey.UID,
			KeyID:     kmsKey.KeyID,
			Region:    k.region,
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		data, err := json.Marshal(keyInfo)
		if err != nil {
			return Item{}, fmt.Errorf("키 정보 직렬화 실패: %w", err)
		}

		return Item{
			Key:         key,
			Data:        data,
			Label:       fmt.Sprintf("KMS Key: %s", kmsKey.UID),
			Description: fmt.Sprintf("AWS KMS Key in region %s", k.region),
			KMSKeyInfo:  keyInfo,
		}, nil
	}

	return Item{}, ErrKeyNotFound
}

// Set 키 저장 (99designs/keyring 인터페이스)
func (k *KMSKeyring) Set(item Item) error {
	// KMS는 키를 외부에서 저장하지 않음
	// 키 생성은 CreateKMSKey를 통해 수행
	return fmt.Errorf("KMS 백엔드는 외부 키 저장을 지원하지 않습니다. CreateKMSKey를 사용하세요")
}

// Remove 키 삭제 (99designs/keyring 인터페이스)
func (k *KMSKeyring) Remove(key string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	kmsKey, exists := k.keyCache[key]
	if !exists {
		return ErrKeyNotFound
	}

	// AWS KMS에서 키 삭제 스케줄링 (즉시 삭제는 불가능)
	_, err := k.kmsClient.ScheduleKeyDeletion(&kms.ScheduleKeyDeletionInput{
		KeyId:               aws.String(kmsKey.KeyID),
		PendingWindowInDays: aws.Int64(7), // 최소 7일
	})
	if err != nil {
		return fmt.Errorf("KMS 키 삭제 스케줄링 실패: %w", err)
	}

	// 로컬 캐시에서 제거
	delete(k.keyCache, key)
	delete(k.addrCache, kmsKey.Address.String())

	return nil
}

// Keys 모든 키 목록 조회 (99designs/keyring 인터페이스)
func (k *KMSKeyring) Keys() ([]string, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	keys := make([]string, 0, len(k.keyCache))
	for uid := range k.keyCache {
		keys = append(keys, uid)
	}

	return keys, nil
}

// CreateKMSKey AWS KMS에서 새로운 키를 생성하고 등록합니다
func (k *KMSKeyring) CreateKMSKey(uid string) (*KMSKey, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	// 이미 존재하는 키인지 확인
	if _, exists := k.keyCache[uid]; exists {
		return nil, fmt.Errorf("키 '%s'가 이미 존재합니다", uid)
	}

	// KMS 키 생성
	keyOutput, err := k.kmsClient.CreateKey(&kms.CreateKeyInput{
		KeySpec:     aws.String(KMSKeySpec),
		KeyUsage:    aws.String(KMSKeyUsage),
		Description: aws.String(fmt.Sprintf("Guru blockchain key for %s", uid)),
		Tags: []*kms.Tag{
			{
				TagKey:   aws.String(KMSTagApplication),
				TagValue: aws.String("guru-blockchain"),
			},
			{
				TagKey:   aws.String(KMSTagKeyType),
				TagValue: aws.String("secp256k1"),
			},
			{
				TagKey:   aws.String(KMSTagCreatedBy),
				TagValue: aws.String(uid),
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "KMS 키 생성 실패")
	}

	keyID := *keyOutput.KeyMetadata.KeyId

	// 공개 키 조회
	pubKeyOutput, err := k.kmsClient.GetPublicKey(&kms.GetPublicKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return nil, errors.Wrap(err, "KMS 공개 키 조회 실패")
	}

	// secp256k1 공개 키로 변환
	pubKey := &ethsecp256k1.PubKey{Key: pubKeyOutput.PublicKey}
	address := sdk.AccAddress(pubKey.Address().Bytes())

	kmsKey := &KMSKey{
		UID:       uid,
		KeyID:     keyID,
		PublicKey: pubKey,
		Address:   address,
	}

	// 캐시에 저장
	k.keyCache[uid] = kmsKey
	k.addrCache[address.String()] = uid

	return kmsKey, nil
}

// GetKMSKey UID로 KMS 키 조회
func (k *KMSKeyring) GetKMSKey(uid string) (*KMSKey, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	if kmsKey, exists := k.keyCache[uid]; exists {
		return kmsKey, nil
	}

	return nil, ErrKeyNotFound
}

// GetKMSKeyByAddress 주소로 KMS 키 조회
func (k *KMSKeyring) GetKMSKeyByAddress(address sdk.AccAddress) (*KMSKey, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	if uid, exists := k.addrCache[address.String()]; exists {
		if kmsKey, exists := k.keyCache[uid]; exists {
			return kmsKey, nil
		}
	}

	return nil, ErrKeyNotFound
}

// Sign KMS를 사용하여 메시지 서명
func (k *KMSKeyring) Sign(uid string, message []byte) ([]byte, error) {
	kmsKey, err := k.GetKMSKey(uid)
	if err != nil {
		return nil, err
	}

	// SHA-256 해시 계산
	hash := sha256.Sum256(message)

	// KMS 서명 요청
	signOutput, err := k.kmsClient.Sign(&kms.SignInput{
		KeyId:            aws.String(kmsKey.KeyID),
		Message:          hash[:],
		MessageType:      aws.String("DIGEST"),
		SigningAlgorithm: aws.String(SigningAlgorithm),
	})
	if err != nil {
		return nil, errors.Wrap(err, "KMS 서명 실패")
	}

	return signOutput.Signature, nil
}

// ListKMSKeys 모든 KMS 키 목록 조회
func (k *KMSKeyring) ListKMSKeys() ([]*KMSKey, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	keys := make([]*KMSKey, 0, len(k.keyCache))
	for _, kmsKey := range k.keyCache {
		keys = append(keys, kmsKey)
	}

	return keys, nil
}
