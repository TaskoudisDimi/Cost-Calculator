package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	apiKey  string
	from    string
	httpCli *http.Client
}

func NewClient(apiKey, from string) *Client {
	return &Client{apiKey: apiKey, from: from, httpCli: &http.Client{}}
}

type sendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (c *Client) Send(to, subject, htmlBody string) error {
	payload := sendRequest{
		From:    c.from,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend returned %d", resp.StatusCode)
	}
	return nil
}

func ReminderHTML(bills []BillItem) string {
	rows := ""
	for _, b := range bills {
		rows += fmt.Sprintf(
			`<tr><td style="padding:8px 12px;border-bottom:1px solid #2d2d2d;">%s</td>`+
				`<td style="padding:8px 12px;border-bottom:1px solid #2d2d2d;text-align:right;">€%.2f</td>`+
				`<td style="padding:8px 12px;border-bottom:1px solid #2d2d2d;text-align:center;">%s</td></tr>`,
			b.Name, b.Amount, b.DueDate,
		)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="background:#0f0f0f;color:#e5e7eb;font-family:sans-serif;margin:0;padding:32px;">
  <div style="max-width:520px;margin:0 auto;">
    <h2 style="color:#6366f1;margin-bottom:8px;">BillTracker</h2>
    <p style="color:#9ca3af;margin-bottom:24px;">Έχεις λογαριασμούς που λήγουν σύντομα:</p>
    <table style="width:100%%;border-collapse:collapse;background:#1a1a1a;border-radius:8px;overflow:hidden;">
      <thead>
        <tr style="background:#1e1e2e;">
          <th style="padding:10px 12px;text-align:left;color:#9ca3af;font-weight:500;">Πάροχος</th>
          <th style="padding:10px 12px;text-align:right;color:#9ca3af;font-weight:500;">Ποσό</th>
          <th style="padding:10px 12px;text-align:center;color:#9ca3af;font-weight:500;">Λήξη</th>
        </tr>
      </thead>
      <tbody>%s</tbody>
    </table>
    <p style="color:#6b7280;font-size:12px;margin-top:24px;">
      Αυτό το email στάλθηκε αυτόματα από το BillTracker.
    </p>
  </div>
</body>
</html>`, rows)
}

type BillItem struct {
	Name    string
	Amount  float64
	DueDate string
}
