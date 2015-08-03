package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	. "github.com/nazoking/hookserver"
)

func main() {
	c := NewHookServer()

	flag.StringVar(&c.Addr, "addr", c.Addr, "address and port of hook server")
	parser := flag.String("parser", "", "json of payload parse configuration")
	flag.StringVar(&c.DefaultBin, "default", "_all", "default script name")
	flag.StringVar(&c.Secret, "secret", "", "secret of webhook Signature")
	flag.StringVar(&c.ScriptRoot, "scripts", c.ScriptRoot, "script root")

	flag.Parse()

	var file []byte
	if *parser != "" {
		var e error
		file, e = ioutil.ReadFile(*parser)
		if e != nil {
			fmt.Printf("File error: %v\n", e)
			os.Exit(1)
		}
	} else {
		file = []byte(`
{
  "pull_request":{
    "Path": "{{.BASE_OWNER}}/{{.BASE_REPO}}/pull_request/{{.ACTION}}",
    "Values":{
      "HEAD_OWNER":"/pull_request/head/repo/owner/login",
      "HEAD_REPO":"/pull_request/head/repo/name",
      "HEAD_BRANCH":"/pull_request/head/ref",
      "HEAD_SHA":"/pull_request/head/sha",
      "BASE_OWNER":"/pull_request/base/repo/owner/login",
      "BASE_REPO":"/pull_request/base/repo/name",
      "BASE_BRANCH":"/pull_request/head/ref",
      "BASE_SHA":"/pull_request/head/sha",
      "ACTION":"/action"
    }
  },
  "push":{
    "Path": "{{.OWNER}}/{{.REPO}}/push/{{.BRANCH}}",
    "Values":{
      "OWNER":"/repository/owner/name",
      "REPO":"/repository/name",
      "BRANCH":"/ref"
    }
  }
}`)
	}
	if keys, err := c.SetEventMapJson(&file); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	} else {
		c.Logger.Println("Event Knonw[" + strings.Join(keys, ", ") + "]")
	}
	if len(c.EventMap) == 0 {
		fmt.Println("no event map. please set -parser=config.json")
		os.Exit(1)
	}

	c.ScriptRoot, _ = filepath.Abs(c.ScriptRoot)
	if s, err := os.Stat(c.ScriptRoot); err != nil || !s.IsDir() {
		fmt.Print("no scripts directory " + c.ScriptRoot)
		os.Exit(1)
	}
	c.Logger.Println("ScriptRoot " + c.ScriptRoot)

	c.Logger.Println("HookServer start " + c.Addr)
	err := http.ListenAndServe(c.Addr, c)
	if err != nil {
		c.Logger.Fatal(err)
		os.Exit(1)
	}
	c.Logger.Println("HookServer stop " + c.Addr)
}
