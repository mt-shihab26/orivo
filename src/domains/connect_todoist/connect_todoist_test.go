package connect_todoist

import (
	"net"
	"net/http"
	"testing"
)

func TestWrongStateDoesNotEndTheSignIn(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + listener.Addr().String() + "/callback"

	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		code, err := waitForCode(listener, "right")
		done <- result{code, err}
	}()

	resp, err := http.Get(base + "?state=wrong&code=stolen")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("wrong state answered %s, want 400", resp.Status)
	}

	resp, err = http.Get(base + "?state=right&code=real")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := <-done; got.err != nil || got.code != "real" {
		t.Fatalf("got code %q, err %v; want the real code", got.code, got.err)
	}
}
