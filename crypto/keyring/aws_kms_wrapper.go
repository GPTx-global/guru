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
	"os"

	"github.com/GPTx-global/guru/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// KMSKeyringWrapper Cosmos SDK keyring 인터페이스 구현
type KMSKeyringWrapper struct {
	kmsBackend *KMSKeyring
}

// NewKMSKeyringWrapper 새로운 KMS wrapper 인스턴스 생성
func NewKMSKeyringWrapper(region string) (*KMSKeyringWrapper, error) {
	// 환경변수에서 region 읽기
	if region == "" {
		region = os.Getenv(EnvAWSRegion)
		if region == "" {
			region = "us-east-1" // 기본값
		}
	}

	kmsBackend, err := NewKMSKeyring(region)
	if err != nil {
		return nil, err
	}

	return &KMSKeyringWrapper{
		kmsBackend: kmsBackend,
	}, nil
}

// Backend keyring backend 타입 반환
func (w *KMSKeyringWrapper) Backend() string {
	return BackendKMS
}

// List 모든 키 레코드 목록 조회
func (w *KMSKeyringWrapper) List() ([]*keyring.Record, error) {
	kmsKeys, err := w.kmsBackend.ListKMSKeys()
	if err != nil {
		return nil, err
	}

	records := make([]*keyring.Record, 0, len(kmsKeys))
	for _, kmsKey := range kmsKeys {
		record, err := keyring.NewLocalRecord(kmsKey.UID, nil, kmsKey.PublicKey)
		if err != nil {
			continue // 오류가 있는 키는 건너뛰기
		}
		records = append(records, record)
	}

	return records, nil
}

// SupportedAlgorithms 지원되는 서명 알고리즘 목록 반환
func (w *KMSKeyringWrapper) SupportedAlgorithms() (keyring.SigningAlgoList, keyring.SigningAlgoList) {
	// KMS는 eth_secp256k1만 지원
	algos := keyring.SigningAlgoList{hd.EthSecp256k1}
	return algos, algos
}

// Key UID로 키 레코드 조회
func (w *KMSKeyringWrapper) Key(uid string) (*keyring.Record, error) {
	kmsKey, err := w.kmsBackend.GetKMSKey(uid)
	if err != nil {
		return nil, err
	}

	return keyring.NewLocalRecord(kmsKey.UID, nil, kmsKey.PublicKey)
}

// KeyByAddress 주소로 키 레코드 조회
func (w *KMSKeyringWrapper) KeyByAddress(address sdk.Address) (*keyring.Record, error) {
	accAddr, ok := address.(sdk.AccAddress)
	if !ok {
		return nil, fmt.Errorf("주소 타입 변환 실패: %T", address)
	}
	kmsKey, err := w.kmsBackend.GetKMSKeyByAddress(accAddr)
	if err != nil {
		return nil, err
	}

	return keyring.NewLocalRecord(kmsKey.UID, nil, kmsKey.PublicKey)
}

// Delete 키 삭제
func (w *KMSKeyringWrapper) Delete(uid string) error {
	return w.kmsBackend.Remove(uid)
}

// DeleteByAddress 주소로 키 삭제
func (w *KMSKeyringWrapper) DeleteByAddress(address sdk.Address) error {
	accAddr, ok := address.(sdk.AccAddress)
	if !ok {
		return fmt.Errorf("주소 타입 변환 실패: %T", address)
	}
	kmsKey, err := w.kmsBackend.GetKMSKeyByAddress(accAddr)
	if err != nil {
		return err
	}
	return w.kmsBackend.Remove(kmsKey.UID)
}

// Rename 키 이름 변경 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) Rename(oldname, newname string) error {
	return fmt.Errorf("KMS 백엔드는 키 이름 변경을 지원하지 않습니다")
}

// NewMnemonic 새로운 KMS 키 생성 (mnemonic 대신 KMS 키 생성)
func (w *KMSKeyringWrapper) NewMnemonic(uid string, language keyring.Language, hdPath, bip39Passphrase string, algo keyring.SignatureAlgo) (*keyring.Record, string, error) {
	// KMS에서는 mnemonic을 생성하지 않고 KMS 키를 생성
	if algo.Name() != hd.EthSecp256k1Type {
		return nil, "", fmt.Errorf("KMS 백엔드는 %s 알고리즘만 지원합니다", hd.EthSecp256k1Type)
	}

	kmsKey, err := w.kmsBackend.CreateKMSKey(uid)
	if err != nil {
		return nil, "", err
	}

	record, err := keyring.NewLocalRecord(kmsKey.UID, nil, kmsKey.PublicKey)
	if err != nil {
		return nil, "", err
	}

	// KMS는 mnemonic이 없으므로 빈 문자열 반환
	return record, "", nil
}

// NewAccount NewMnemonic의 별칭
func (w *KMSKeyringWrapper) NewAccount(uid string, mnemonic, bip39Passphrase, hdPath string, algo keyring.SignatureAlgo) (*keyring.Record, error) {
	// KMS에서는 mnemonic을 무시하고 새 키 생성
	record, _, err := w.NewMnemonic(uid, keyring.English, hdPath, bip39Passphrase, algo)
	return record, err
}

// SaveLedgerKey Ledger 키 저장 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) SaveLedgerKey(uid string, algo keyring.SignatureAlgo, hrp string, coinType, account, index uint32) (*keyring.Record, error) {
	return nil, fmt.Errorf("KMS 백엔드는 Ledger 키를 지원하지 않습니다")
}

// SaveOfflineKey 오프라인 키 저장 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) SaveOfflineKey(uid string, pubkey cryptotypes.PubKey) (*keyring.Record, error) {
	return nil, fmt.Errorf("KMS 백엔드는 오프라인 키를 지원하지 않습니다")
}

// SaveMultisig 멀티시그 키 저장 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) SaveMultisig(uid string, pubkey cryptotypes.PubKey) (*keyring.Record, error) {
	return nil, fmt.Errorf("KMS 백엔드는 멀티시그 키를 지원하지 않습니다")
}

// Sign 메시지 서명
func (w *KMSKeyringWrapper) Sign(uid string, msg []byte) ([]byte, cryptotypes.PubKey, error) {
	kmsKey, err := w.kmsBackend.GetKMSKey(uid)
	if err != nil {
		return nil, nil, err
	}

	signature, err := w.kmsBackend.Sign(uid, msg)
	if err != nil {
		return nil, nil, err
	}

	return signature, kmsKey.PublicKey, nil
}

// SignByAddress 주소로 메시지 서명
func (w *KMSKeyringWrapper) SignByAddress(address sdk.Address, msg []byte) ([]byte, cryptotypes.PubKey, error) {
	accAddr, ok := address.(sdk.AccAddress)
	if !ok {
		return nil, nil, fmt.Errorf("주소 타입 변환 실패: %T", address)
	}
	kmsKey, err := w.kmsBackend.GetKMSKeyByAddress(accAddr)
	if err != nil {
		return nil, nil, err
	}

	signature, err := w.kmsBackend.Sign(kmsKey.UID, msg)
	if err != nil {
		return nil, nil, err
	}

	return signature, kmsKey.PublicKey, nil
}

// ImportPrivKey 개인 키 가져오기 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) ImportPrivKey(uid, armor, passphrase string) error {
	return fmt.Errorf("KMS 백엔드는 개인 키 가져오기를 지원하지 않습니다")
}

// ImportPubKey 공개 키 가져오기 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) ImportPubKey(uid string, armor string) error {
	return fmt.Errorf("KMS 백엔드는 공개 키 가져오기를 지원하지 않습니다")
}

// ExportPubKeyArmor 공개 키를 Armor 형식으로 내보내기
func (w *KMSKeyringWrapper) ExportPubKeyArmor(uid string) (string, error) {
	kmsKey, err := w.kmsBackend.GetKMSKey(uid)
	if err != nil {
		return "", err
	}

	// Armor 형식으로 변환 (예시)
	return fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----",
		string(kmsKey.PublicKey.Bytes())), nil
}

// ExportPubKeyArmorByAddress 주소로 공개 키를 Armor 형식으로 내보내기
func (w *KMSKeyringWrapper) ExportPubKeyArmorByAddress(address sdk.Address) (string, error) {
	accAddr, ok := address.(sdk.AccAddress)
	if !ok {
		return "", fmt.Errorf("주소 타입 변환 실패: %T", address)
	}
	kmsKey, err := w.kmsBackend.GetKMSKeyByAddress(accAddr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----",
		string(kmsKey.PublicKey.Bytes())), nil
}

// ExportPrivKeyArmor 개인 키를 Armor 형식으로 내보내기 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) ExportPrivKeyArmor(uid, encryptPassphrase string) (armor string, err error) {
	return "", fmt.Errorf("KMS 백엔드는 개인 키 내보내기를 지원하지 않습니다")
}

// ExportPrivKeyArmorByAddress 주소로 개인 키를 Armor 형식으로 내보내기 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) ExportPrivKeyArmorByAddress(address sdk.Address, encryptPassphrase string) (armor string, err error) {
	return "", fmt.Errorf("KMS 백엔드는 개인 키 내보내기를 지원하지 않습니다")
}

// MigrateAll 모든 키 마이그레이션 (KMS에서는 지원하지 않음)
func (w *KMSKeyringWrapper) MigrateAll() ([]*keyring.Record, error) {
	return nil, fmt.Errorf("KMS 백엔드는 키 마이그레이션을 지원하지 않습니다")
}
