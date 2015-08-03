package hookserver

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	// "net/http/cgi"

	"os/exec"
	"strings"
	"syscall"
)

type Processor struct {
	*HookServer
	Response http.ResponseWriter
	Request  *http.Request
}

func (p *Processor) validateSecret(body *[]byte, secret string) error {
	headers := p.Request.Header
	hmacHex := headers.Get("X-Hub-Signature")

	if !strings.HasPrefix(hmacHex, "sha1=") {
		return fmt.Errorf("Unknown hash type: %s", hmacHex)
	}

	hmacSig, err := hex.DecodeString(hmacHex[5:])
	if err != nil {
		return err
	}

	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(*body)
	expectedMAC := mac.Sum(nil)
	if !hmac.Equal(hmacSig, expectedMAC) {
		return errors.New("Invalid Signature")
	}

	return nil
}

func (p *Processor) InternalServerError(err error) {
	p.Logger.Println(err)
	p.Response.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(p.Response, err.Error())
}
func (p *Processor) NotFound() {
	p.Response.WriteHeader(http.StatusNotFound)
}
func (p *Processor) BadRequest(err error) {
	p.Logger.Println(err)
	p.Response.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(p.Response, err.Error())
}

func upperCaseAndUnderscore(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - ('a' - 'A')
	case r == '-':
		return '_'
	case r == '=':
		return '_'
	}
	// TODO: other transformations in spec or practice?
	return r
}

func (h *HookServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := Processor{h, w, r}
	eventType := r.Header.Get("X-Github-Event")

	event, ok := h.EventMap[eventType]
	if !ok {
		p.BadRequest(fmt.Errorf("unknown event %s", eventType))
		return
	}
	body, err := getBody(r)
	if err != nil {
		p.InternalServerError(err)
		return
	}
	if secret := p.Secret; secret != "" {
		if err := p.validateSecret(&body, secret); err != nil {
			p.BadRequest(err)
			return
		}
	}

	payload, err := getPayloadString(r, &body)
	if err != nil {
		p.BadRequest(err)
		return
	}
	val := map[string]string{
		"SERVER_NAME":    r.Host,
		"HTTP_HOST":      r.Host,
		"REQUEST_METHOD": r.Method,
		"QUERY_STRING":   r.URL.RawQuery,
		"REQUEST_URI":    r.URL.RequestURI(),
		"REMOTE_ADDR":    r.RemoteAddr,
		"REMOTE_HOST":    r.RemoteAddr,
	}
	for k, v := range r.Header {
		k = strings.Map(upperCaseAndUnderscore, k)
		joinStr := ", "
		if k == "COOKIE" {
			joinStr = "; "
		}
		val["HTTP_"+k] = strings.Join(v, joinStr)
	}
	if err := parsePayload(val, payload, event.Values); err != nil {
		p.InternalServerError(err)
		return
	}

	bin := event.GetBinPath(val)
	binPath, err := findBinPath(p.ScriptRoot, bin, p.DefaultBin)
	if err != nil {
		p.Logger.Printf("%s => not found", bin)
		p.NotFound()
	} else {
		val["PATH_INFO"] = binPath
		cmd := exec.Command(binPath)
		//cmd.Stdin = r.Body
		//cmd.Stdin = payload
		cmd.Stdin = bytes.NewReader(*payload)
		for k, v := range val {
			cmd.Env = append(cmd.Env, k+"="+v)
		}

		var errorExit bool
		if err := cmd.Run(); err != nil {
			if e2, ok := err.(*exec.ExitError); ok {
				if s, ok := e2.Sys().(syscall.WaitStatus); ok {
					errorExit = s.ExitStatus() != 0
				} else {
				}
			}
		}
		if errorExit {
			p.Logger.Printf("ng %s => %v", binPath, err)
			p.InternalServerError(fmt.Errorf("%s ERROR", bin))
		} else {
			p.Logger.Printf("ok %s => %s", bin, binPath)
			p.Response.WriteHeader(http.StatusOK)
			fmt.Fprintf(p.Response, "ok")
		}

	}
}
