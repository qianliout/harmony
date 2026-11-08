package test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"harmony/infra/consts"
	"harmony/infra/model/dict"
	"harmony/infra/model/event"
	"harmony/infra/model/record"
)

type fakeWechat struct {
	qrErr error
}

func (*fakeWechat) Code2Session(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("empty code")
	}
	return "openid-" + code, nil
}

func (f *fakeWechat) GetUnlimitedQRCode(ctx context.Context, scene, page, envVersion string) ([]byte, error) {
	if f.qrErr != nil {
		return nil, f.qrErr
	}
	return []byte("fake-png"), nil
}

func TestAuthTokenLifecycle(t *testing.T) {
	h := newHarness(t)
	login, err := h.authSrv.IssueTokenForCLI(h.ctx(), 1, 14)
	requireOK(t, err)
	if login.User.ID != 1 || !login.User.IsAdmin {
		t.Fatalf("unexpected login user %+v", login.User)
	}
	owner, err := h.authSrv.VerifyToken(h.ctx(), login.Token)
	requireOK(t, err)
	if owner != 1 {
		t.Fatalf("expected owner 1, got %d", owner)
	}

	// 把令牌改成 6 天后过期，验证一次请求自动续期到 14 天。
	near := time.Now().Add(6 * 24 * time.Hour)
	if err := h.tx.Model(&struct{}{}).Table("auth_token").Where("user_id = ?", 1).
		Update("expires_at", near).Error; err != nil {
		t.Fatalf("shorten token: %v", err)
	}
	if _, err := h.authSrv.VerifyToken(h.ctx(), login.Token); err != nil {
		t.Fatalf("verify renewed token: %v", err)
	}
	var expires time.Time
	if err := h.tx.Table("auth_token").Select("expires_at").Where("user_id = ?", 1).Scan(&expires).Error; err != nil {
		t.Fatalf("read expires: %v", err)
	}
	if expires.Before(time.Now().Add(13 * 24 * time.Hour)) {
		t.Fatalf("expected renewed expiry, got %v", expires)
	}

	// 过期令牌返回 401。
	past := time.Now().Add(-time.Hour)
	if err := h.tx.Table("auth_token").Where("user_id = ?", 1).Update("expires_at", past).Error; err != nil {
		t.Fatalf("expire token: %v", err)
	}
	if _, err := h.authSrv.VerifyToken(h.ctx(), login.Token); err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestAuthMissingToken(t *testing.T) {
	h := newHarness(t)
	if _, err := h.authSrv.VerifyToken(h.ctx(), ""); err == nil {
		t.Fatal("expected unauthorized error for empty token")
	}
	if _, err := h.authSrv.VerifyToken(h.ctx(), "not-a-token"); err == nil {
		t.Fatal("expected unauthorized error for unknown token")
	}
}

func TestWxLoginClaimsAdminAndInitializesUser(t *testing.T) {
	h := newHarness(t)
	first, err := h.authSrv.WxLogin(h.ctx(), "alice-code")
	requireOK(t, err)
	if first.User.ID != 1 || !h.authSrv.IsAdmin(first.User.ID) {
		t.Fatalf("first login should claim user 1, got %+v", first.User)
	}

	second, err := h.authSrv.WxLogin(h.ctx(), "bob-code")
	requireOK(t, err)
	if second.User.ID == 1 || h.authSrv.IsAdmin(second.User.ID) {
		t.Fatalf("second login should be a new non-admin user, got %+v", second.User)
	}

	var dictCount int64
	if err := h.tx.Model(&dict.Item{}).Where("owner_id = ?", second.User.ID).Count(&dictCount).Error; err != nil {
		t.Fatalf("count dicts: %v", err)
	}
	if dictCount != int64(len(builtinDictCount())) {
		t.Fatalf("expected %d builtin dicts, got %d", len(builtinDictCount()), dictCount)
	}
	var placeholders int64
	if err := h.tx.Model(&event.Event{}).Where("owner_id = ? AND placeholder = 1", second.User.ID).Count(&placeholders).Error; err != nil {
		t.Fatalf("count placeholders: %v", err)
	}
	if placeholders != 1 {
		t.Fatalf("expected one placeholder event, got %d", placeholders)
	}
}

func builtinDictCount() []string {
	names := make([]string, 0, len(consts.BuiltinDictItems))
	for _, item := range consts.BuiltinDictItems {
		names = append(names, item.Code)
	}
	return names
}

func TestWebTicketFlow(t *testing.T) {
	h := newHarness(t)
	ticket, err := h.authSrv.CreateWebTicket(h.ctx(), "127.0.0.1", "test-agent")
	requireOK(t, err)
	if ticket.Ticket == "" || ticket.QRKind != "wxacode" || ticket.QRImage == "" {
		t.Fatalf("unexpected ticket %+v", ticket)
	}
	if err := h.authSrv.ScanWebTicket(h.ctx(), 1, ticket.Ticket); err != nil {
		t.Fatalf("scan: %v", err)
	}
	status, err := h.authSrv.PollWebTicket(h.ctx(), ticket.Ticket)
	requireOK(t, err)
	if status.Status != "scanned" || status.Token != "" {
		t.Fatalf("expected scanned, got %+v", status)
	}
	if err := h.authSrv.ConfirmWebTicket(h.ctx(), 1, ticket.Ticket); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	confirmed, err := h.authSrv.PollWebTicket(h.ctx(), ticket.Ticket)
	requireOK(t, err)
	if confirmed.Status != "confirmed" || confirmed.Token == "" {
		t.Fatalf("expected confirmed token, got %+v", confirmed)
	}
	owner, err := h.authSrv.VerifyToken(h.ctx(), confirmed.Token)
	requireOK(t, err)
	if owner != 1 {
		t.Fatalf("expected h5 token owner 1, got %d", owner)
	}
	again, err := h.authSrv.PollWebTicket(h.ctx(), ticket.Ticket)
	requireOK(t, err)
	if again.Status != "expired" {
		t.Fatalf("ticket should be single use, got %+v", again)
	}
}

func TestImageSignature(t *testing.T) {
	h := newHarness(t)
	expires, sig := imageParts(t, h.authSrv.SignImage(12, 1))
	if !h.authSrv.VerifyImage(12, 1, expires, sig) {
		t.Fatal("expected valid image signature")
	}
	if h.authSrv.VerifyImage(13, 1, expires, sig) {
		t.Fatal("tampered id should fail")
	}
	if h.authSrv.VerifyImage(12, 2, expires, sig) {
		t.Fatal("tampered owner should fail")
	}
	if h.authSrv.VerifyImage(12, 1, expires-1, sig) {
		t.Fatal("tampered expiry should fail")
	}
	if h.authSrv.VerifyImage(12, 1, time.Now().Add(-time.Hour).UnixMilli(), sig) {
		t.Fatal("expired signature should fail")
	}
}

func imageParts(t *testing.T, url string) (int64, string) {
	t.Helper()
	var expires int64
	var sig string
	for _, part := range splitQuery(url) {
		switch {
		case len(part) > 3 && part[:3] == "id=":
		case len(part) > 4 && part[:4] == "exp=":
			value, err := strconv.ParseInt(part[4:], 10, 64)
			if err != nil {
				t.Fatalf("parse exp: %v", err)
			}
			expires = value
		case len(part) > 4 && part[:4] == "sig=":
			sig = part[4:]
		}
	}
	if expires == 0 || sig == "" {
		t.Fatalf("signature parts missing: %s", url)
	}
	return expires, sig
}

func splitQuery(url string) []string {
	out := make([]string, 0)
	start := -1
	for i := 0; i < len(url); i++ {
		if url[i] == '?' {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return out
	}
	current := ""
	for i := start; i < len(url); i++ {
		if url[i] == '&' {
			out = append(out, current)
			current = ""
			continue
		}
		current += string(url[i])
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}

func TestDataIsolation(t *testing.T) {
	h := newHarness(t)
	p := h.createPerson("隔离甲")
	e := h.createEvent("隔离事", "wedding", "2024-01-01", "day")
	b := h.createBook("隔离本", "renqing", 0)
	created, err := h.recordSrv.Create(h.ctx(), record.CreateParam{
		PersonID: p, EventID: e, BookID: b, Direction: "in",
		AmountCent: 10000, OccurredOn: "2024-01-01", DatePrecision: "day", Status: "pending",
	})
	requireOK(t, err)

	other := h.ctxAs(2)
	items, total, err := h.recordSrv.List(other, record.Query{}, nil)
	requireOK(t, err)
	if total != 0 || len(items) != 0 {
		t.Fatalf("user 2 must not see user 1 records, total=%d", total)
	}
	if _, err := h.recordSrv.Detail(other, created.ID); err == nil {
		t.Fatal("user 2 must not read user 1 record detail")
	}
	if _, err := h.recordSrv.Update(other, created.ID, record.UpdateParam{AmountCent: 1, OccurredOn: "2024-01-01"}); err == nil {
		t.Fatal("user 2 must not update user 1 record")
	}
}

func TestAuthAdminOnly(t *testing.T) {
	h := newHarness(t)
	if !h.authSrv.IsAdmin(1) {
		t.Fatal("user 1 should be admin")
	}
	if h.authSrv.IsAdmin(2) {
		t.Fatal("user 2 should not be admin")
	}
}

func TestBuiltinSeedsMatchConstants(t *testing.T) {
	h := newHarness(t)
	var rows []dict.Item
	if err := h.tx.Where("owner_id = ?", 1).Order("dict_type, code").Find(&rows).Error; err != nil {
		t.Fatalf("load dicts: %v", err)
	}
	if len(rows) != len(consts.BuiltinDictItems) {
		t.Fatalf("schema seed has %d dicts, constants have %d", len(rows), len(consts.BuiltinDictItems))
	}
	want := make(map[string]consts.DictSeed, len(consts.BuiltinDictItems))
	for _, item := range consts.BuiltinDictItems {
		want[item.DictType+"/"+item.Code] = item
	}
	for _, row := range rows {
		seed, ok := want[row.DictType+"/"+row.Code]
		if !ok {
			t.Fatalf("schema dict %s/%s missing from constants", row.DictType, row.Code)
		}
		if row.Label != seed.Label || row.Color != seed.Color || row.Sort != seed.Sort {
			t.Fatalf("dict %s/%s mismatch: schema=%+v constants=%+v", row.DictType, row.Code, row, seed)
		}
	}
}
