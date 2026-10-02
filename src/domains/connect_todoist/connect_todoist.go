package connect_todoist

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"orivo/src/systems/config"
	"orivo/src/systems/todoist"
)

const loginTimeout = 5 * time.Minute

func Run(pasteToken bool) error {
	var creds todoist.Credentials
	var err error

	if pasteToken {
		creds, err = readToken()
	} else {
		creds, err = browserLogin()
	}
	if err != nil {
		return err
	}

	name, err := todoist.NewClient(creds.AccessToken).User()
	if errors.Is(err, todoist.ErrTokenRejected) {
		return errors.New("Todoist rejected the token; nothing was saved")
	}
	if err != nil {
		return fmt.Errorf("could not reach Todoist: %w", err)
	}

	if err := todoist.SaveCredentials(config.TodoistAuth(), creds); err != nil {
		return err
	}

	fmt.Printf("Connected as %s.\n", name)
	fmt.Println("Run `orivo sync-todoist` to fetch your todos.")
	return nil
}

func browserLogin() (todoist.Credentials, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return todoist.Credentials{}, err
	}
	defer listener.Close()

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", listener.Addr().(*net.TCPAddr).Port)
	oauth := todoist.NewOAuth()

	clientID, err := oauth.Register(redirectURI)
	if err != nil {
		return todoist.Credentials{}, fmt.Errorf("could not reach Todoist: %w", err)
	}

	state := todoist.NewState()
	verifier, challenge := todoist.NewPKCE()
	loginURL := oauth.LoginURL(clientID, redirectURI, state, challenge)

	fmt.Println("Opening Todoist in your browser to sign in.")
	fmt.Println("If it does not open, visit this address yourself:")
	fmt.Println()
	fmt.Println("  " + loginURL)
	fmt.Println()
	_ = openBrowser(loginURL)

	fmt.Println("Waiting for you to approve access...")
	code, err := waitForCode(listener, state)
	if err != nil {
		return todoist.Credentials{}, err
	}

	creds, err := oauth.Exchange(clientID, redirectURI, code, verifier)
	if errors.Is(err, todoist.ErrTokenRejected) {
		return todoist.Credentials{}, errors.New("Todoist did not accept the sign-in; try again")
	}
	return creds, err
}

func waitForCode(listener net.Listener, state string) (string, error) {
	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		outcome := result{code: query.Get("code")}

		switch {
		case query.Get("state") != state:
			outcome = result{err: errors.New("the sign-in reply did not match this request; try again")}
		case query.Get("error") != "":
			outcome = result{err: fmt.Errorf("Todoist sign-in was not approved: %s", query.Get("error"))}
		case outcome.code == "":
			outcome = result{err: errors.New("Todoist sent no sign-in code; try again")}
		}

		message := "You can close this tab and go back to the terminal."
		if outcome.err != nil {
			w.WriteHeader(http.StatusBadRequest)
			message = "Sign-in failed. Go back to the terminal and try again."
		}
		fmt.Fprintf(w, "<!doctype html><meta charset=\"utf-8\"><title>orivo</title><p style=\"font-family:sans-serif\">%s</p>", message)

		select {
		case done <- outcome:
		default:
		}
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go server.Serve(listener)
	defer server.Close()

	select {
	case outcome := <-done:
		time.Sleep(200 * time.Millisecond)
		return outcome.code, outcome.err
	case <-time.After(loginTimeout):
		return "", errors.New("timed out waiting for the browser sign-in")
	}
}

func readToken() (todoist.Credentials, error) {
	fmt.Println("Copy your API token from Todoist: Settings > Integrations > Developer.")
	fmt.Print("Paste it here: ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	token := strings.TrimSpace(line)
	if token == "" {
		if err != nil {
			return todoist.Credentials{}, errors.New("no token given")
		}
		return todoist.Credentials{}, errors.New("the token is empty")
	}
	return todoist.Credentials{AccessToken: token}, nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
