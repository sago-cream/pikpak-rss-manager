package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

const PasswordRecoveryTTL = 15 * time.Minute
const recoverySetting = "administrator_recovery"

var ErrRecoveryCode = errors.New("復原碼無效或已過期，請在伺服器建立新碼")

type passwordRecovery struct {
	CodeHash     string `json:"code_hash"`
	PasswordHash string `json:"password_hash"`
	ExpiresAt    int64  `json:"expires_at"`
}

// IssuePasswordRecovery is called only by the local server command. Persist a
// digest bound to the current verifier, so a later password change revokes it.
func (s *Store) IssuePasswordRecovery(ctx context.Context, now time.Time) (string, error) {
	verifier, _, err := s.Administrator(ctx)
	if err != nil {
		return "", err
	}
	if verifier == "" {
		return "", errors.New("請先完成初始化")
	}
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(value)
	digest := sha256.Sum256([]byte(code))
	encoded, err := json.Marshal(passwordRecovery{hex.EncodeToString(digest[:]), verifier, now.Add(PasswordRecoveryTTL).Unix()})
	if err != nil {
		return "", err
	}
	if err := s.SetSetting(ctx, recoverySetting, string(encoded)); err != nil {
		return "", err
	}
	return code, nil
}

// Consume the code and replace only its bound verifier in one transaction.
// Public settings, cloud credentials and download records remain unchanged.
func (s *Store) RecoverAdministratorPassword(ctx context.Context, code, verifier string, now time.Time) error {
	if verifier == "" {
		return errors.New("密碼驗證值不可為空")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var encrypted string
	if err := tx.QueryRowContext(ctx, "SELECT value FROM settings WHERE key=?", recoverySetting).Scan(&encrypted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecoveryCode
		}
		return err
	}
	decoded, err := s.open(encrypted)
	if err != nil {
		return err
	}
	var recovery passwordRecovery
	if json.Unmarshal([]byte(decoded), &recovery) != nil {
		return ErrRecoveryCode
	}
	digest := sha256.Sum256([]byte(code))
	if recovery.ExpiresAt <= now.Unix() || subtle.ConstantTimeCompare([]byte(recovery.CodeHash), []byte(hex.EncodeToString(digest[:]))) != 1 {
		return ErrRecoveryCode
	}
	result, err := tx.ExecContext(ctx, "UPDATE administrator SET password_hash=? WHERE id=1 AND password_hash=?", verifier, recovery.PasswordHash)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrRecoveryCode
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM settings WHERE key=?", recoverySetting); err != nil {
		return err
	}
	return tx.Commit()
}
