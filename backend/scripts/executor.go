package scripts

import (
	"bytes"
	"digistore/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type OrderCtx struct {
	InvoiceNo   string
	ProductName string
	BuyerName   string
	BuyerEmail  string
	Total       string
	Qty         string
}

type ActionResult struct {
	Provider string `json:"provider"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
	Status   string `json:"status"` // ok | skipped | failed
	Output   string `json:"output"`
}

type Result struct {
	Status  string         // success | partial | failed
	Actions []ActionResult // hasil per action
	Outputs []string       // ringkasan teks (untuk log)
}

// Execute menjalankan semua ProviderAction dalam script JSON.
// Action dengan Enabled=false di-skip — tidak dieksekusi, tapi tetap dicatat di log.
func Execute(scriptJSON string, order *models.Order, emailFn func(to, subj, body string) error) Result {
	if strings.TrimSpace(scriptJSON) == "" {
		return Result{Status: "success", Outputs: []string{"[skip] tidak ada script"}}
	}

	var actions []models.ProviderAction
	if err := json.Unmarshal([]byte(scriptJSON), &actions); err != nil {
		return Result{
			Status:  "failed",
			Outputs: []string{"[error] script JSON tidak valid: " + err.Error()},
		}
	}

	ctx := OrderCtx{
		InvoiceNo:   order.InvoiceNo,
		ProductName: order.ProductName,
		BuyerName:   order.BuyerName,
		BuyerEmail:  order.BuyerEmail,
		Total:       fmt.Sprintf("Rp %d", order.Total),
		Qty:         fmt.Sprintf("%d", order.Qty),
	}

	var results []ActionResult
	okCount, failCount := 0, 0

	for i, action := range actions {
		label := action.Label
		if label == "" {
			label = fmt.Sprintf("%s #%d", action.Provider, i+1)
		}
		prefix := fmt.Sprintf("[%d/%d][%s][%s]", i+1, len(actions), action.Provider, label)

		// ── SKIP jika action dinonaktifkan ───────────────────────────────────
		if !action.Enabled {
			res := ActionResult{
				Provider: action.Provider, Label: label,
				Enabled: false, Status: "skipped",
				Output: fmt.Sprintf("%s SKIP — action dinonaktifkan", prefix),
			}
			results = append(results, res)
			log.Printf("%s", res.Output)
			continue
		}

		// ── Eksekusi berdasarkan provider ────────────────────────────────────
		var out string
		var execErr error

		switch action.Provider {

		case "email":
			to := render(action.To, ctx)
			subj := render(action.Subject, ctx)
			body := render(action.Body, ctx)
			if execErr = emailFn(to, subj, body); execErr != nil {
				out = fmt.Sprintf("%s GAGAL kirim email ke %s: %s", prefix, to, execErr)
			} else {
				out = fmt.Sprintf("%s email terkirim ke %s", prefix, to)
			}

		case "slack":
			wh := action.WebhookURL
			if wh == "" {
				wh = action.URL
			}
			msg := render(action.Message, ctx)
			payload, _ := json.Marshal(map[string]string{"text": msg})
			if execErr = postJSON(wh, payload, nil); execErr != nil {
				out = fmt.Sprintf("%s GAGAL kirim Slack: %s", prefix, execErr)
			} else {
				out = fmt.Sprintf("%s pesan Slack terkirim", prefix)
			}

		case "discord":
			wh := action.WebhookURL
			if wh == "" {
				wh = action.URL
			}
			msg := render(action.Message, ctx)
			payload, _ := json.Marshal(map[string]string{"content": msg})
			if execErr = postJSON(wh, payload, nil); execErr != nil {
				out = fmt.Sprintf("%s GAGAL kirim Discord: %s", prefix, execErr)
			} else {
				out = fmt.Sprintf("%s pesan Discord terkirim", prefix)
			}

		case "webhook", "curl":
			method := action.Method
			if method == "" {
				method = "POST"
			}
			url := render(action.URL, ctx)
			payload, _ := json.Marshal(map[string]interface{}{
				"invoice_no":   ctx.InvoiceNo,
				"product_name": ctx.ProductName,
				"buyer_name":   ctx.BuyerName,
				"buyer_email":  ctx.BuyerEmail,
				"total":        ctx.Total,
				"qty":          ctx.Qty,
				"timestamp":    time.Now().Format(time.RFC3339),
			})
			httpReq, e := http.NewRequest(method, url, bytes.NewBuffer(payload))
			if e != nil {
				execErr = e
				out = fmt.Sprintf("%s GAGAL buat request: %s", prefix, e)
				break
			}
			httpReq.Header.Set("Content-Type", "application/json")
			for k, v := range action.Headers {
				httpReq.Header.Set(k, v)
			}
			client := &http.Client{Timeout: 10 * time.Second}
			resp, e2 := client.Do(httpReq)
			if e2 != nil {
				execErr = e2
				out = fmt.Sprintf("%s GAGAL %s %s: %s", prefix, method, url, e2)
			} else {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				out = fmt.Sprintf("%s %s %s → %d: %s", prefix, method, url, resp.StatusCode, string(body))
			}

		case "log":
			msg := render(action.Message, ctx)
			out = fmt.Sprintf("%s %s", prefix, msg)
			log.Printf("[SCRIPT LOG] %s", msg)

		default:
			out = fmt.Sprintf("%s provider '%s' tidak dikenali", prefix, action.Provider)
			execErr = fmt.Errorf("unknown provider: %s", action.Provider)
		}

		status := "ok"
		if execErr != nil {
			status = "failed"
			failCount++
		} else {
			okCount++
		}
		log.Printf("%s", out)
		results = append(results, ActionResult{
			Provider: action.Provider, Label: label,
			Enabled: true, Status: status, Output: out,
		})
	}

	// Tentukan status keseluruhan
	overallStatus := "success"
	if okCount == 0 && failCount > 0 {
		overallStatus = "failed"
	} else if failCount > 0 {
		overallStatus = "partial"
	}

	var outputs []string
	for _, r := range results {
		outputs = append(outputs, r.Output)
	}

	return Result{Status: overallStatus, Actions: results, Outputs: outputs}
}

func postJSON(url string, payload []byte, headers map[string]string) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func render(tpl string, ctx OrderCtx) string {
	return strings.NewReplacer(
		"{{invoice_no}}", ctx.InvoiceNo,
		"{{product_name}}", ctx.ProductName,
		"{{buyer_name}}", ctx.BuyerName,
		"{{buyer_email}}", ctx.BuyerEmail,
		"{{total}}", ctx.Total,
		"{{qty}}", ctx.Qty,
	).Replace(tpl)
}
