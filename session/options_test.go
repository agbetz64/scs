package session

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/mem/engine"
)

func TestCookieOptions(t *testing.T) {
	m := Manage(engine.New())
	h := m(testServeMux)

	_, _, cookie := testRequest(t, h, "/PutString", "")
	if strings.Contains(cookie, "Path=/") == false {
		t.Fatalf("got %q: expected to contain %q", cookie, "Path=/")
	}
	if strings.Contains(cookie, "Domain=") == true {
		t.Fatalf("got %q: expected to not contain %q", cookie, "Domain=")
	}
	if strings.Contains(cookie, "Secure") == true {
		t.Fatalf("got %q: expected to not contain %q", cookie, "Secure")
	}
	if strings.Contains(cookie, "HttpOnly") == false {
		t.Fatalf("got %q: expected to contain %q", cookie, "HttpOnly")
	}

	m = Manage(engine.New(), Path("/foo"), Domain("example.org"), Secure(true), HttpOnly(false), Persist(true), MaxAge(time.Hour))
	h = m(testServeMux)

	_, _, cookie = testRequest(t, h, "/PutString", "")
	if strings.Contains(cookie, "Path=/foo") == false {
		t.Fatalf("got %q: expected to contain %q", cookie, "Path=/foo")
	}
	if strings.Contains(cookie, "Domain=example.org") == false {
		t.Fatalf("got %q: expected to contain %q", cookie, "Domain=example.org")
	}
	if strings.Contains(cookie, "Secure") == false {
		t.Fatalf("got %q: expected to contain %q", cookie, "Secure")
	}
	if strings.Contains(cookie, "HttpOnly") == true {
		t.Fatalf("got %q: expected to not contain %q", cookie, "HttpOnly")
	}
	if strings.Contains(cookie, "Max-Age=3600") == false {
		t.Fatalf("got %q: expected to contain %q:", cookie, "Max-Age=86400")
	}
	if strings.Contains(cookie, "Expires=") == false {
		t.Fatalf("got %q: expected to contain %q:", cookie, "Expires")
	}

	m = Manage(engine.New(), MaxAge(time.Hour))
	h = m(testServeMux)

	_, _, cookie = testRequest(t, h, "/PutString", "")
	if strings.Contains(cookie, "Max-Age=") == true {
		t.Fatalf("got %q: expected not to contain %q:", cookie, "Max-Age=")
	}
	if strings.Contains(cookie, "Expires=") == true {
		t.Fatalf("got %q: expected not to contain %q:", cookie, "Expires")
	}
}

func TestAlwaysSave(t *testing.T) {
	e := engine.New()
	m := Manage(e, AlwaysSave(true))
	h := m(testServeMux)

	_, _, cookie := testRequest(t, h, "/GetString", "")
	if cookie == "" {
		t.Fatalf("got %q: expected session cookie", cookie)
	}
	token := extractTokenFromCookie(cookie)
	_, found, _ := e.FindValues(token)
	if found != true {
		t.Fatalf("got %v: expected %v", found, true)
	}
}

func TestMaxAge(t *testing.T) {
	e := engine.New()
	m := Manage(e, MaxAge(100*time.Millisecond), AlwaysSave(true))
	h := m(testServeMux)

	_, _, cookie := testRequest(t, h, "/PutString", "")
	oldToken := extractTokenFromCookie(cookie)

	_, body, _ := testRequest(t, h, "/GetString", cookie)
	if body != "lorem ipsum" {
		t.Fatalf("got %q: expected %q", body, "lorem ipsum")
	}

	time.Sleep(100 * time.Millisecond)

	_, body, cookie = testRequest(t, h, "/GetString", cookie)
	if body != ErrKeyNotFound.Error() {
		t.Fatalf("got %q: expected %q", body, ErrKeyNotFound.Error())
	}
	newToken := extractTokenFromCookie(cookie)
	if newToken == oldToken {
		t.Fatalf("expected a difference", newToken)
	}
}

func TestErrorFunc(t *testing.T) {
	m := Manage(engine.New())
	man, ok := m(nil).(*manager)
	if ok == false {
		t.Fatal("type assertion failed")
	}

	rr := httptest.NewRecorder()
	man.opts.errorFunc(rr, new(http.Request), errors.New("test error"))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("got %d: expected %d", rr.Code, http.StatusInternalServerError)
	}
	if string(rr.Body.Bytes()) != "test error\n" {
		t.Fatalf("got %q: expected %q", string(rr.Body.Bytes()), "test error\n")
	}

	m = Manage(engine.New(), ErrorFunc(func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(418)
		io.WriteString(w, http.StatusText(418))
	}))
	man, ok = m(nil).(*manager)
	if ok == false {
		t.Fatal("type assertion failed")
	}

	rr = httptest.NewRecorder()
	man.opts.errorFunc(rr, new(http.Request), errors.New("test error"))
	if rr.Code != 418 {
		t.Fatalf("got %d: expected %d", rr.Code, 418)
	}
	if string(rr.Body.Bytes()) != http.StatusText(418) {
		t.Fatalf("got %q: expected %q", string(rr.Body.Bytes()), http.StatusText(418))
	}
}

func TestCookieName(t *testing.T) {
	oldCookieName := CookieName
	CookieName = "custom_cookie_name"

	m := Manage(engine.New())
	h := m(testServeMux)

	_, _, cookie := testRequest(t, h, "/PutString", "")
	if strings.HasPrefix(cookie, "custom_cookie_name=") == false {
		t.Fatalf("got %q: expected prefix %q", "custom_cookie_name=")
	}

	_, body, _ := testRequest(t, h, "/GetString", cookie)
	if body != "lorem ipsum" {
		t.Fatalf("got %q: expected %q", body, "lorem ipsum")
	}

	CookieName = oldCookieName
}

func TestContextDataName(t *testing.T) {
	oldContextDataName := ContextDataName
	ContextDataName = "custom_context_name"

	m := Manage(engine.New())
	h := m(testServeMux)

	_, _, cookie := testRequest(t, h, "/PutString", "")
	_, body, _ := testRequest(t, h, "/DumpContext", cookie)
	if strings.Contains(body, "custom_context_name") == false {
		t.Fatalf("got %q: expected to contain %q", body, "custom_context_name")
	}
	_, body, _ = testRequest(t, h, "/GetString", cookie)
	if body != "lorem ipsum" {
		t.Fatalf("got %q: expected %q", body, "lorem ipsum")
	}

	ContextDataName = oldContextDataName
}
