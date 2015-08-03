package hookserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	// "net/http/cgi"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	scan "github.com/mattn/go-scan"
)

type EventConfig struct {
	Path         string
	Values       map[string]string
	PathTemplate *template.Template
}

func (event *EventConfig) Compile(name string) {
	event.PathTemplate = template.Must(template.New(name).Parse(event.Path))
}
func (event *EventConfig) CalcPath(val map[string]string) string {
	var doc bytes.Buffer
	event.PathTemplate.Execute(&doc, val)
	return doc.String()
}
func (event *EventConfig) GetBinPath(val map[string]string) string {
	bin := event.CalcPath(val)
	if !path.IsAbs(bin) {
		bin = "/" + bin
	}
	bin = path.Clean(bin)
	return bin
}

type HookServer struct {
	Addr       string
	EventMap   map[string]*EventConfig
	Logger     *log.Logger
	DefaultBin string
	Secret     string
	ScriptRoot string
}

func NewHookServer() *HookServer {
	p, _ := filepath.Abs(".")
	logger := log.New(os.Stdout, "* ", log.LstdFlags)
	return &HookServer{":8999", make(map[string]*EventConfig), logger, "_all", "", p + "/scripts"}
}

func getPayloadString(r *http.Request, body *[]byte) (*[]byte, error) {
	contentType := r.Header.Get("content-type")
	if contentType == "application/x-www-form-urlencoded" {
		values, err := url.ParseQuery(string(*body))
		if err != nil {
			return nil, err
		}
		pValues := values["payload"]
		if len(pValues) == 0 {
			return nil, errors.New("No payload found")
		}
		ret := []byte(pValues[0])
		return &ret, nil
	} else if contentType == "application/json" {
		return body, nil
	} else {
		return nil, errors.New("Unknown content type")
	}
}
func parsePayload(val map[string]string, payloadS *[]byte, values map[string]string) error {
	var payload interface{}
	if err := json.Unmarshal(*payloadS, &payload); err != nil {
		return err
	}

	for name, jpath := range values {
		var value string
		if scan.ScanTree(payload, jpath, &value) == nil {
			val[name] = value
		}
	}
	return nil
}

func getBody(p *http.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(p.Body)
	return body, err
}
func findBinPath(root string, bin string, defaultBin string) (string, error) {
	if bin == "/" {
		return "", errors.New("not found")
	}
	if strings.HasSuffix(bin, "/") {
		bin = bin[:len(bin)-1]
	}
	if b, err := exec.LookPath(root + bin + "/" + defaultBin); err == nil && b != "" {
		return b, nil
	}
	if b, err := exec.LookPath(root + bin); err == nil && b != "" {
		return b, nil
	}
	pare, _ := path.Split(bin)
	return findBinPath(root, pare, defaultBin)
}

func (h *HookServer) SetEventMapJson(file *[]byte) ([]string, error) {
	if err := json.Unmarshal(*file, &h.EventMap); err != nil {
		return nil, err
	}
	m := make([]string, 0, len(h.EventMap))
	for k, _ := range h.EventMap {
		h.EventMap[k].Compile(k)
		m = append(m, k)
	}
	return m, nil
}
