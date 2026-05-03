// Package queue menyediakan factory untuk membuat asynq client dan server,
// serta format standar TaskEnvelope untuk komunikasi antar service via Redis queue.
package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Daftar error yang mungkin dikembalikan oleh fungsi queue.
var (
	ErrEmptyEventType = errors.New("event_type tidak boleh kosong")
	ErrEmptyTenantID  = errors.New("tenant_id tidak boleh kosong")
	ErrInvalidPayload = errors.New("payload tidak valid")
)

// TaskEnvelope adalah format standar untuk semua task yang dikirim via queue.
// Setiap event antar service menggunakan format ini untuk konsistensi.
type TaskEnvelope struct {
	// EventType menentukan jenis event, contoh: "customer.created"
	EventType string `json:"event_type"`

	// TenantID adalah UUID tenant pemilik event
	TenantID string `json:"tenant_id"`

	// Timestamp adalah waktu event dibuat dalam format ISO 8601
	Timestamp time.Time `json:"timestamp"`

	// CorrelationID adalah UUID v4 untuk tracing lintas service.
	// Jika kosong saat enqueue, akan di-generate otomatis.
	CorrelationID string `json:"correlation_id"`

	// Payload berisi data spesifik event dalam format JSON mentah
	Payload json.RawMessage `json:"payload"`
}

// ClientConfig berisi konfigurasi koneksi Redis untuk asynq client dan server.
type ClientConfig struct {
	// Host adalah alamat host Redis
	Host string

	// Port adalah nomor port Redis
	Port int

	// Password adalah password Redis (kosong jika tidak ada)
	Password string

	// DB adalah nomor database Redis yang digunakan
	DB int
}

// redisAddr membangun string alamat Redis dari host dan port.
func (c ClientConfig) redisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// toRedisClientOpt mengkonversi ClientConfig ke asynq.RedisClientOpt.
func (c ClientConfig) toRedisClientOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     c.redisAddr(),
		Password: c.Password,
		DB:       c.DB,
	}
}

// NewClient membuat asynq client baru untuk mengirim task ke queue.
// Client harus di-close setelah selesai digunakan.
func NewClient(cfg ClientConfig) (*asynq.Client, error) {
	client := asynq.NewClient(cfg.toRedisClientOpt())
	return client, nil
}

// NewServer membuat asynq server baru untuk memproses task dari queue.
// Parameter concurrency menentukan jumlah worker goroutine.
// Parameter queues menentukan prioritas queue, contoh: {"critical": 6, "default": 3, "low": 1}.
func NewServer(cfg ClientConfig, concurrency int, queues map[string]int) (*asynq.Server, error) {
	srv := asynq.NewServer(
		cfg.toRedisClientOpt(),
		asynq.Config{
			Concurrency: concurrency,
			Queues:      queues,
		},
	)
	return srv, nil
}

// EnqueueTask membuat asynq.Task dari TaskEnvelope dan mengirimnya ke queue.
// EventType digunakan sebagai tipe task di asynq.
// Jika CorrelationID kosong, akan di-generate UUID v4 baru.
// Jika Timestamp kosong (zero value), akan diisi dengan waktu saat ini.
func EnqueueTask(client *asynq.Client, envelope TaskEnvelope) error {
	if envelope.EventType == "" {
		return ErrEmptyEventType
	}
	if envelope.TenantID == "" {
		return ErrEmptyTenantID
	}

	// Generate correlation ID jika belum diisi
	if envelope.CorrelationID == "" {
		envelope.CorrelationID = uuid.New().String()
	}

	// Isi timestamp jika belum diisi
	if envelope.Timestamp.IsZero() {
		envelope.Timestamp = time.Now()
	}

	// Serialisasi envelope ke JSON sebagai payload task
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("gagal serialisasi envelope: %w", err)
	}

	// Buat task dengan EventType sebagai tipe task
	task := asynq.NewTask(envelope.EventType, payload)

	// Kirim task ke queue
	_, err = client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("gagal mengirim task ke queue: %w", err)
	}

	return nil
}

// DecodeEnvelope mendekode payload asynq.Task menjadi TaskEnvelope.
// Digunakan oleh worker untuk membaca data dari task yang diterima.
func DecodeEnvelope(task *asynq.Task) (*TaskEnvelope, error) {
	var envelope TaskEnvelope
	if err := json.Unmarshal(task.Payload(), &envelope); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPayload, err.Error())
	}
	return &envelope, nil
}
