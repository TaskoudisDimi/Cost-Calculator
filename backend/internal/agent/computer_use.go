// Package agent implements the AI payment automation using chromedp + Claude Computer Use.
package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	claudeModel   = "claude-opus-4-5"
	maxIterations = 50
	viewportW     = 1280
	viewportH     = 800
)

// Event is a progress update streamed back to the frontend.
type Event struct {
	Type    string `json:"type"`
	JobID   string `json:"job_id,omitempty"`
	Message string `json:"message,omitempty"`
	Action  string `json:"action,omitempty"`
	Ref     string `json:"reference,omitempty"`
}

// JobConfig holds everything the agent needs for a payment.
type JobConfig struct {
	JobID        string
	APIKey       string
	ProviderName string
	Amount       float64
	PaymentCode  string
	PaymentURL   string
	StartURL     string
	CredLabel    string
	CredUser     string
	CredPass     string
	CardNumber   string
	CardExpiry   string
	CardCVV      string
	CardHolder   string
	OTPCh        chan string
}

// RunLoop runs the browser automation + Claude Computer Use loop.
func RunLoop(ctx context.Context, cfg JobConfig, out chan<- Event) {
	emit := func(t, msg string) {
		select {
		case out <- Event{Type: t, JobID: cfg.JobID, Message: msg}:
		default:
		}
	}

	emit("started", fmt.Sprintf("Ξεκινώ πληρωμή %s — %.2f€", cfg.ProviderName, cfg.Amount))

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.WindowSize(viewportW, viewportH),
	)
	// Use system Chromium if CHROME_BIN is set (Docker/Cloud Run)
	if chromeBin := os.Getenv("CHROME_BIN"); chromeBin != "" {
		opts = append(opts, chromedp.ExecPath(chromeBin))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	defer cancelChrome()

	startURL := cfg.StartURL
	if startURL == "" {
		startURL = InferURL(cfg.ProviderName)
	}

	emit("navigating", "Ανοίγω "+startURL)
	_, err := chromedp.Run(chromeCtx, chromedp.Navigate(startURL))
	if err != nil {
		emit("error", "Αδύνατη η πλοήγηση: "+err.Error())
		return
	}
	time.Sleep(2 * time.Second)

	systemPrompt := buildSystemPrompt(cfg)
	messages := []claudeMessage{}

	ss, err := takeScreenshot(chromeCtx)
	if err != nil {
		emit("error", "Screenshot απέτυχε: "+err.Error())
		return
	}
	messages = append(messages, claudeMessage{
		Role: "user",
		Content: []claudeContent{
			{Type: "text", Text: "Ξεκίνα την πληρωμή. Τρέχουσα σελίδα:"},
			imageContent(ss),
		},
	})

	for i := 0; i < maxIterations; i++ {
		select {
		case <-ctx.Done():
			emit("error", "Ακυρώθηκε.")
			return
		default:
		}

		resp, err := callClaude(cfg.APIKey, systemPrompt, messages)
		if err != nil {
			emit("error", "Claude API: "+err.Error())
			return
		}

		pauseType := ""  // "otp" or "user_input"
		pausePrompt := ""
		for _, blk := range resp.Content {
			if blk.Type == "text" && blk.Text != "" {
				emit("agent_message", blk.Text)
				if strings.Contains(blk.Text, "OTP_REQUIRED") {
					pauseType = "otp"
					pausePrompt = "Εισάγετε τον OTP που λάβατε στο κινητό"
				}
				if idx := strings.Index(blk.Text, "USER_INPUT_REQUIRED:"); idx >= 0 {
					pauseType = "user_input"
					rest := strings.SplitN(blk.Text[idx:], "USER_INPUT_REQUIRED:", 2)[1]
					pausePrompt = strings.TrimSpace(strings.SplitN(rest, "\n", 2)[0])
				}
				if strings.Contains(blk.Text, "PAYMENT_COMPLETE") {
					ref := strings.TrimSpace(strings.SplitN(blk.Text, "PAYMENT_COMPLETE", 2)[1])
					out <- Event{Type: "success", JobID: cfg.JobID, Message: "Η πληρωμή ολοκληρώθηκε! ✓", Ref: ref}
					return
				}
				if strings.Contains(blk.Text, "CAPTCHA_BLOCKED") {
					emit("error", "CAPTCHA εντοπίστηκε — αδύνατη η αυτόματη πληρωμή.")
					return
				}
				if strings.Contains(blk.Text, "AMOUNT_MISMATCH") {
					emit("error", blk.Text)
					return
				}
			}
		}

		if pauseType != "" {
			evtType := "otp_required"
			if pauseType == "user_input" {
				evtType = "user_input_required"
			}
			out <- Event{Type: evtType, JobID: cfg.JobID, Message: pausePrompt}
			select {
			case answer := <-cfg.OTPCh:
				emit("otp_received", "Απάντηση λήφθηκε, συνεχίζω...")
				fillScript := fmt.Sprintf(
					`(function(){var el=document.querySelector('input[type="number"],input[type="text"],input[type="tel"]');if(el){el.value=%q;el.dispatchEvent(new Event('input',{bubbles:true}));}})()`,
					answer,
				)
				_, _ = chromedp.Run(chromeCtx, chromedp.Evaluate[any](fillScript))
				_, _ = chromedp.Run(chromeCtx, chromedp.KeyEvent("\r"))
				time.Sleep(2 * time.Second)
			case <-time.After(3 * time.Minute):
				emit("error", "Timeout: δεν ελήφθη απάντηση εντός 3 λεπτών.")
				return
			case <-ctx.Done():
				return
			}
		}

		toolUses := []claudeContentBlock{}
		for _, blk := range resp.Content {
			if blk.Type == "tool_use" {
				toolUses = append(toolUses, blk)
			}
		}

		if len(toolUses) == 0 {
			if resp.StopReason == "end_turn" {
				emit("info", "Agent ολοκλήρωσε χωρίς επιβεβαίωση.")
			}
			break
		}

		messages = append(messages, claudeMessage{Role: "assistant", Content: contentFromBlocks(resp.Content)})

		toolResults := []claudeContent{}
		for _, tu := range toolUses {
			emit("action", describeAction(tu.Input))
			ss, err := executeAction(chromeCtx, tu.Input)
			if err != nil {
				emit("info", "Action error: "+err.Error())
			}
			toolResults = append(toolResults, claudeContent{
				Type:      "tool_result",
				ToolUseID: tu.ID,
				Content:   []claudeContent{imageContent(ss)},
			})
		}
		messages = append(messages, claudeMessage{Role: "user", Content: toolResults})
	}

	emit("error", "Agent έφτασε τον μέγιστο αριθμό ενεργειών χωρίς επιβεβαίωση.")
}

// InferURL maps a provider name to its payment/login URL.
func InferURL(name string) string {
	n := strings.ToLower(name)
	mapping := map[string]string{
		"taxisnet": "https://www.aade.gr/myaade",
		"aade":     "https://www.aade.gr/myaade",
		"εφορία":   "https://www.aade.gr/myaade",
		"εφορια":   "https://www.aade.gr/myaade",
		"deh":      "https://www.dei.gr/el/idiotikes-katikies/logariasmos-kai-pliromi/ilektroniki-pliromi/",
		"δεη":      "https://www.dei.gr/el/idiotikes-katikies/logariasmos-kai-pliromi/ilektroniki-pliromi/",
		"vodafone": "https://www.vodafone.gr/my-vodafone/",
		"cosmote":  "https://www.cosmote.gr/cs/mycosmote/",
		"telekom":  "https://go.telekom.gr/openBills",
		"nova":     "https://www.nova.gr/my-nova/",
		"wind":     "https://www.wind.gr/my-wind/",
		"eydap":    "https://www.eydap.gr/CustomerService/ElectronicServices/",
		"ευδαπ":    "https://www.eydap.gr/CustomerService/ElectronicServices/",
		"elta":     "https://www.elta.gr/",
	}
	for k, url := range mapping {
		if strings.Contains(n, k) {
			return url
		}
	}
	return "https://www.google.com/search?q=" + name + "+πληρωμή+λογαριασμού+online"
}

func buildSystemPrompt(cfg JobConfig) string {
	credPart := "Δεν υπάρχουν αποθηκευμένοι κωδικοί — ψάξε για πληρωμή επισκέπτη.\n"
	if cfg.CredUser != "" {
		credPart = fmt.Sprintf("Username/email: %s\nPassword: %s\n", cfg.CredUser, cfg.CredPass)
	}
	cardPart := "Δεν υπάρχει κάρτα — χρησιμοποίησε RF/IRIS αν υπάρχει.\n"
	if cfg.CardNumber != "" {
		cardPart = fmt.Sprintf("Αριθμός κάρτας: %s\nExpiry: %s\nCVV: %s\nΌνομα: %s\n",
			cfg.CardNumber, cfg.CardExpiry, cfg.CardCVV, cfg.CardHolder)
	}
	rfPart := ""
	if cfg.PaymentCode != "" {
		rfPart = fmt.Sprintf("RF/Κωδικός πληρωμής: %s\n", cfg.PaymentCode)
	}

	return fmt.Sprintf(`Είσαι AI agent πληρωμών για την εφαρμογή BillTracker. Πρέπει να πληρώσεις:

Πάροχος: %s
Ποσό: %.2f EUR
%s
Κωδικοί σύνδεσης:
%s
Στοιχεία κάρτας:
%s
ΟΔΗΓΙΕΣ:
1. Πλοήγησε στη σελίδα πληρωμής του παρόχου.
2. Συνδέσου αν χρειάζεται με τους παρεχόμενους κωδικούς.
3. Βρες την ενότητα πληρωμής λογαριασμού. Ψάξε για το ποσό %.2f€ ή τον κωδικό RF.
4. Αν χρειαστεί OTP via SMS για login ή πληρωμή, πες ακριβώς: "OTP_REQUIRED" και περίμενε.
5. Αν χρειάζεσαι οποιαδήποτε άλλη πληροφορία από τον χρήστη (π.χ. αριθμός κινητού, PIN, κωδικός), πες:
   "USER_INPUT_REQUIRED: [ερώτηση προς τον χρήστη]"
   π.χ. "USER_INPUT_REQUIRED: Ποιος είναι ο αριθμός κινητού σας (10 ψηφία);"
   Μετά περίμενε — ο χρήστης θα απαντήσει και θα λάβεις την απάντηση στο επόμενο μήνυμα.
6. Εισάγαγε τα στοιχεία κάρτας όταν ζητηθούν.
7. Ολοκλήρωσε. Μόλις τελειώσει, πες: "PAYMENT_COMPLETE" και τον αριθμό επιβεβαίωσης.

ΚΑΝΟΝΕΣ:
- Πλήρωσε ΜΟΝΟ το ακριβές ποσό: %.2f EUR
- ΜΗΝ αποδεχτείς επιπλέον χρεώσεις
- Αν δεις CAPTCHA αδύνατο να λύσεις, πες: "CAPTCHA_BLOCKED"
- Αν ποσό ΔΕΝ ταιριάζει, πες: "AMOUNT_MISMATCH: [ποσό]"`,
		cfg.ProviderName, cfg.Amount, rfPart, credPart, cardPart,
		cfg.Amount, cfg.Amount, cfg.Amount)
}

func takeScreenshot(ctx context.Context) (string, error) {
	buf, err := chromedp.Run(ctx, chromedp.CaptureScreenshot())
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

func executeAction(ctx context.Context, input map[string]any) (string, error) {
	action, _ := input["action"].(string)

	var runErr error
	switch action {
	case "left_click":
		coord, ok := input["coordinate"].([]any)
		if ok && len(coord) >= 2 {
			x, y := toFloat(coord[0]), toFloat(coord[1])
			_, runErr = chromedp.Run(ctx, chromedp.MouseClickXY(x, y))
			time.Sleep(500 * time.Millisecond)
		}

	case "double_click":
		coord, ok := input["coordinate"].([]any)
		if ok && len(coord) >= 2 {
			x, y := toFloat(coord[0]), toFloat(coord[1])
			_, runErr = chromedp.Run(ctx, chromedp.MouseClickXY(x, y))
			if runErr == nil {
				_, runErr = chromedp.Run(ctx, chromedp.MouseClickXY(x, y))
			}
			time.Sleep(500 * time.Millisecond)
		}

	case "type":
		text, _ := input["text"].(string)
		_, runErr = chromedp.Run(ctx, chromedp.KeyEvent(text))
		time.Sleep(300 * time.Millisecond)

	case "key":
		key, _ := input["key"].(string)
		_, runErr = chromedp.Run(ctx, chromedp.KeyEvent(key))
		time.Sleep(500 * time.Millisecond)

	case "scroll":
		coord, ok := input["coordinate"].([]any)
		if ok && len(coord) >= 2 {
			y := toFloat(coord[1])
			dir, _ := input["direction"].(string)
			delta := 300.0
			if dir == "up" {
				delta = -300
			}
			script := fmt.Sprintf("window.scrollBy(0, %v)", delta)
			_ = y
			_, runErr = chromedp.Run(ctx, chromedp.Evaluate[any](script))
		}
		time.Sleep(300 * time.Millisecond)
	}

	ss, ssErr := takeScreenshot(ctx)
	if runErr != nil && ssErr != nil {
		return "", ssErr
	}
	return ss, nil
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}

func describeAction(input map[string]any) string {
	action, _ := input["action"].(string)
	switch action {
	case "left_click", "right_click", "double_click":
		coord, _ := input["coordinate"].([]any)
		return fmt.Sprintf("Click %v", coord)
	case "type":
		text, _ := input["text"].(string)
		if len(text) >= 13 {
			digits := 0
			for _, c := range text {
				if c >= '0' && c <= '9' {
					digits++
				}
			}
			if digits >= 13 {
				return "Type: [card number]"
			}
		}
		if len(text) > 30 {
			return "Type: " + text[:30] + "..."
		}
		return "Type: " + text
	case "key":
		return "Key: " + fmt.Sprint(input["key"])
	case "scroll":
		return fmt.Sprintf("Scroll %v", input["direction"])
	}
	return action
}

// ── Claude API types ──────────────────────────────────────────────────────────

type claudeMessage struct {
	Role    string         `json:"role"`
	Content []claudeContent `json:"content"`
}

type claudeContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Source    *imageSource    `json:"source,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   []claudeContent `json:"content,omitempty"`
}

type imageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type claudeContentBlock struct {
	Type  string         `json:"type"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Text  string         `json:"text,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

type claudeResponse struct {
	Content    []claudeContentBlock `json:"content"`
	StopReason string               `json:"stop_reason"`
}

func imageContent(b64 string) claudeContent {
	return claudeContent{
		Type: "image",
		Source: &imageSource{
			Type:      "base64",
			MediaType: "image/png",
			Data:      b64,
		},
	}
}

func contentFromBlocks(blocks []claudeContentBlock) []claudeContent {
	out := make([]claudeContent, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "text":
			out = append(out, claudeContent{Type: "text", Text: b.Text})
		case "tool_use":
			inp, _ := json.Marshal(b.Input)
			out = append(out, claudeContent{Type: "tool_use", Text: string(inp)})
		}
	}
	return out
}

func callClaude(apiKey, system string, messages []claudeMessage) (*claudeResponse, error) {
	body := map[string]any{
		"model":      claudeModel,
		"max_tokens": 4096,
		"system":     system,
		"messages":   messages,
		"tools": []map[string]any{
			{
				"type":              "computer_20241022",
				"name":              "computer",
				"display_width_px":  viewportW,
				"display_height_px": viewportH,
				"display_number":    1,
			},
		},
		"betas": []string{"computer-use-2024-10-22"},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "computer-use-2024-10-22")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Claude API %d: %s", resp.StatusCode, string(b))
	}

	var result claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
