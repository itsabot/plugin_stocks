package stocks

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/itsabot/abot/core"
	"github.com/julienschmidt/httprouter"
)

var r *httprouter.Router

func TestMain(m *testing.M) {
	err := os.Setenv("ABOT_ENV", "test")
	if err != nil {
		log.Fatal("failed to set ABOT_ENV.", err)
	}
	r, err = core.NewServer()
	if err != nil {
		log.Fatal("failed to start abot server.", err)
	}
	cleanup()
	exitVal := m.Run()
	cleanup()
	os.Exit(exitVal)
}

func TestKWGetStockDetails(t *testing.T) {
	testReq(t, "How's the AAPL stock?", "is trading at")
	testReq(t, "What's the share price for GOOG?", "is trading at")
}

func request(method, path string, data []byte) (int, string) {
	req, err := http.NewRequest(method, path, bytes.NewBuffer(data))
	if err != nil {
		return 0, "err completing request: " + err.Error()
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, string(w.Body.Bytes())
}

func testReq(t *testing.T, in, exp string) {
	data := struct {
		FlexIDType int
		FlexID     string
		CMD        string
	}{
		FlexIDType: 3,
		FlexID:     "0",
		CMD:        in,
	}
	byt, err := json.Marshal(data)
	if err != nil {
		t.Fatal("failed to marshal req.", err)
	}
	c, b := request("POST", os.Getenv("ABOT_URL")+"/", byt)
	if c != http.StatusOK {
		t.Fatal("exp", http.StatusOK, "got", c, b)
	}
	if !strings.Contains(b, exp) {
		t.Fatalf("exp %q, got %q for %q\n", exp, b, in)
	}
}

func cleanup() {
	q := `DELETE FROM messages`
	_, err := p.DB.Exec(q)
	if err != nil {
		p.Log.Info("failed to delete messages.", err)
	}
	q = `DELETE FROM states`
	_, err = p.DB.Exec(q)
	if err != nil {
		p.Log.Info("failed to delete messages.", err)
	}
}
