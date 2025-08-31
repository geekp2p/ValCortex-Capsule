
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	defaultModelName = "gemma2b-local"
	defaultPort      = "8088"
	ollamaPort       = "11434"
	ollamaZipURL     = "https://github.com/ollama/ollama/releases/latest/download/ollama-windows-amd64.zip"
)

type genReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}
type genResp struct {
	Response  string `json:"response"`
	EvalCount int    `json:"eval_count"`
}

func main() {
	workDir, _ := os.Getwd()
	binDir := filepath.Join(workDir, "bin")
	_ = os.MkdirAll(binDir, 0755)

	ollamaExe := filepath.Join(binDir, "ollama.exe")
	modelDir := filepath.Join(workDir, ".ollama")
	_ = os.MkdirAll(modelDir, 0755)

	// auto-download ollama if missing
	if _, err := os.Stat(ollamaExe); err != nil {
		log.Printf("ollama.exe ไม่พบ — กำลังดาวน์โหลดจาก %s ...", ollamaZipURL)
		zipPath := filepath.Join(workDir, "ollama.zip")
		if err := downloadFile(ollamaZipURL, zipPath); err != nil {
			log.Fatalf("ดาวน์โหลด ollama ไม่สำเร็จ: %v", err)
		}
		if err := unzip(zipPath, binDir); err != nil {
			log.Fatalf("แตก zip ไม่สำเร็จ: %v", err)
		}
		_ = os.Remove(zipPath)
		if _, err := os.Stat(ollamaExe); err != nil {
			found, err := findFile(binDir, "ollama.exe")
			if err != nil || found == "" {
				log.Fatalf("ไม่พบ ollama.exe หลังแตก zip")
			}
			if found != ollamaExe {
				_ = os.Rename(found, ollamaExe)
			}
		}
		log.Printf("ดาวน์โหลด Ollama เสร็จแล้ว")
	}

	// ensure Modelfile exists
	modelFile := filepath.Join(workDir, "Modelfile")
	if _, err := os.Stat(modelFile); err != nil {
		content := "FROM G:\\\\models\\\\gemma-2-2b-it-Q4_K_M.gguf\n\n" +
			"TEMPLATE \"\"\"<|user|>\n{{ .Prompt }}\n\n<|assistant|>\"\"\"\n\n" +
			"PARAMETER num_ctx 4096\nPARAMETER temperature 0.7\n"
		if err := os.WriteFile(modelFile, []byte(content), 0644); err != nil {
			log.Fatalf("เขียน Modelfile ไม่สำเร็จ: %v", err)
		}
		log.Println("สร้าง Modelfile เรียบร้อย")
	}

	// start ollama serve
	serveCmd := exec.Command(ollamaExe, "serve")
	serveCmd.Env = append(os.Environ(),
		"OLLAMA_HOST=127.0.0.1:"+ollamaPort,
		"OLLAMA_MODELS="+modelDir,
	)
	serveCmd.Stdout = os.Stdout
	serveCmd.Stderr = os.Stderr
	if err := serveCmd.Start(); err != nil {
		log.Fatalf("สตาร์ท ollama ไม่สำเร็จ: %v", err)
	}
	log.Printf("ollama serve pid=%d", serveCmd.Process.Pid)

	if err := waitPort("127.0.0.1", ollamaPort, 90*time.Second); err != nil {
		killProcess(serveCmd)
		log.Fatalf("ollama ไม่พร้อม: %v", err)
	}

	// create model if needed
	if err := ensureModel(ollamaExe, modelFile, defaultModelName); err != nil {
		killProcess(serveCmd)
		log.Fatalf("สร้างโมเดลไม่สำเร็จ: %v", err)
	}

	// http api
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":true,"model":"`+defaultModelName+`"}`)
	})
	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		var in struct {
			Prompt string `json:"prompt"`
			Model  string `json:"model,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		model := in.Model
		if model == "" {
			model = defaultModelName
		}
		out, err := callOllamaGenerate(model, in.Prompt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(out)
	})

	srv := &http.Server{Addr: ":" + defaultPort, Handler: mux}
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		_ = srv.Shutdown(context.Background())
		killProcess(serveCmd)
	}()

	log.Printf("READY: http://127.0.0.1:%s (POST /chat)", defaultPort)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		killProcess(serveCmd)
		log.Fatal(err)
	}
}

func waitPort(host, port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 1*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}

func ensureModel(ollamaExe, modelfile, name string) error {
	if hasModel() {
		return nil
	}
	cmd := exec.Command(ollamaExe, "create", name, "-f", modelfile)
	cmd.Env = append(os.Environ(), "OLLAMA_HOST=127.0.0.1:"+ollamaPort)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func hasModel() bool {
	res, err := http.Get("http://127.0.0.1:" + ollamaPort + "/api/tags")
	if err != nil { return false }
	defer res.Body.Close()
	var data struct{ Models []struct{ Name string } `json:"models"` }
	_ = json.NewDecoder(res.Body).Decode(&data)
	for _, m := range data.Models {
		if strings.EqualFold(m.Name, defaultModelName) {
			return true
		}
	}
	return false
}

func callOllamaGenerate(model, prompt string) (*genResp, error) {
	body, _ := json.Marshal(genReq{Model: model, Prompt: prompt, Stream: false})
	res, err := http.Post("http://127.0.0.1:"+ollamaPort+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil { return nil, err }
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("status %d: %s", res.StatusCode, string(b))
	}
	var out genResp
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil { return nil, err }
	return &out, nil
}

func downloadFile(url, dst string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil { return err }
	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == 200 { break }
		time.Sleep(2 * time.Second)
	}
	if err != nil { return err }
	if resp.StatusCode != 200 { return fmt.Errorf("status %d", resp.StatusCode) }
	defer resp.Body.Close()
	f, err := os.Create(dst); if err != nil { return err }
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func unzip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil { return err }
	defer r.Close()
	for _, f := range r.File {
		fp := filepath.Join(dst, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fp, 0755); err != nil { return err }
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fp), 0755); err != nil { return err }
		rc, err := f.Open()
		if err != nil { return err }
		out, err := os.Create(fp)
		if err != nil { rc.Close(); return err }
		if _, err := io.Copy(out, rc); err != nil {
			out.Close(); rc.Close(); return err
		}
		out.Close(); rc.Close()
	}
	return nil
}

func findFile(root, name string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil { return err }
		if !d.IsDir() && strings.EqualFold(filepath.Base(p), name) {
			found = p
			return io.EOF
		}
		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) { return "", err }
	return found, nil
}
