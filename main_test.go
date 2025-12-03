package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIndexHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	index(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("ожидался статус 200, получили %d", resp.StatusCode)
	}
}

func TestShowPostHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/post/9999", nil) // статьи 9999 нет
	w := httptest.NewRecorder()

	show_post(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("ожидался статус 404, получили %d", resp.StatusCode)
	}
}

func TestLoginHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()

	login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("ожидался статус 200, получили %d", resp.StatusCode)
	}
}

func TestRegisterPostHandler_EmptyFields(t *testing.T) {
	form := strings.NewReader("username=&email=&password=")
	req := httptest.NewRequest("POST", "/register", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	registerPost(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("ожидался статус 200, получили %d", resp.StatusCode)
	}
}
