package types

import "time"

// Job represents a single Oracle data collection task
type Job struct {
	ID    string        // Job 고유 ID
	URL   string        // 외부 API URL
	Path  string        // JSON 데이터 추출 경로
	Nonce uint64        // Oracle 실행 횟수 (0부터 시작)
	Delay time.Duration // Oracle 완료 후 대기 시간
}

// OracleData represents Oracle data to be sent to the blockchain
type OracleData struct {
	RequestID string // 요청 ID (Job.ID 그대로, 예: "BTC-PRICE")
	Data      string // 실제 수집된 데이터 (외부 API에서 파싱한 결과)
	Nonce     uint64 // 현재 Oracle 실행 횟수
}
