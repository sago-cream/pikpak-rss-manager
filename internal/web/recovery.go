package web

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

func (s *Server) recoverPassword(w http.ResponseWriter, r *http.Request) {
	if !s.validCSRF(r) {
		JSON(w, 403, map[string]string{"error": "請求驗證失敗"})
		return
	}
	if !s.initialized() {
		JSON(w, 409, map[string]string{"error": "請先完成初始化"})
		return
	}
	var in struct {
		Code            string `json:"code"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := decode(w, r, &in); err != nil {
		failure(w, err)
		return
	}
	in.Code = strings.TrimSpace(in.Code)
	if in.Code == "" {
		failure(w, errors.New("請輸入復原碼"))
		return
	}
	if in.NewPassword == "" {
		failure(w, errors.New("請輸入新密碼"))
		return
	}
	if in.NewPassword != in.ConfirmPassword {
		failure(w, errors.New("兩次輸入的密碼不一致"))
		return
	}
	if !s.beginAuth(w, r) {
		return
	}
	defer func() { <-s.authGate }()
	if len(in.Code) != 43 {
		failure(w, store.ErrRecoveryCode)
		return
	}
	verifier, err := hashPassword(in.NewPassword)
	in.NewPassword, in.ConfirmPassword = "", ""
	if err != nil {
		failure(w, err)
		return
	}
	err = s.DB.RecoverAdministratorPassword(r.Context(), in.Code, verifier, time.Now())
	in.Code = ""
	if errors.Is(err, store.ErrRecoveryCode) {
		failure(w, store.ErrRecoveryCode)
		return
	}
	if err != nil {
		JSON(w, 500, map[string]string{"error": "無法重設密碼，請重試"})
		return
	}
	s.mu.Lock()
	s.hash = verifier
	clear(s.sessions)
	s.mu.Unlock()
	s.cookie(w, r, "pp_session", "", true, -1)
	s.cookie(w, r, "pp_csrf", nonce(), false, 43200)
	JSON(w, 200, map[string]bool{"ok": true})
}
