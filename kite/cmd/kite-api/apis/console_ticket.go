package apis

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ConsoleTicket struct {
	Subject   string    `json:"sub"`
	Namespace string    `json:"namespace"`
	VMName    string    `json:"vmName"`
	Purpose   string    `json:"purpose"`
	ExpiresAt time.Time `json:"expiresAt"`
}

const consoleTicketPurpose = "vm-console"

// ConsoleTicketService signs and verifies short-lived console tickets.
// ttl controls how long a browser can wait between ticket issue and WebSocket open.
// secret signs tickets so any kite-api replica can verify them without shared memory.
// This type is used by console ticket and console WebSocket handlers.
type ConsoleTicketService struct {
	ttl    time.Duration
	secret []byte
}

// NewConsoleTicketService creates a console ticket signer.
// ttl is the lifetime for each issued ticket.
// secret must match across kite-api replicas and comes from runtime JWT config.
// This function is used by production route defaults and API tests.
func NewConsoleTicketService(ttl time.Duration, secret string) *ConsoleTicketService {
	return &ConsoleTicketService{
		ttl:    ttl,
		secret: []byte(secret),
	}
}

// Issue creates a signed console ticket for one VM.
// subject is the authenticated Kite username.
// namespace and vmName identify the only VM this ticket may open.
// The returned string is safe to place in the WebSocket URL query for a short time.
func (s *ConsoleTicketService) Issue(subject string, namespace string, vmName string, now time.Time) (string, ConsoleTicket, error) {
	if len(s.secret) == 0 {
		return "", ConsoleTicket{}, fmt.Errorf("console ticket secret is not configured")
	}

	ticket := ConsoleTicket{
		Subject:   subject,
		Namespace: namespace,
		VMName:    vmName,
		Purpose:   consoleTicketPurpose,
		ExpiresAt: now.Add(s.ttl).UTC(),
	}

	payload, err := json.Marshal(ticket)
	if err != nil {
		return "", ConsoleTicket{}, fmt.Errorf("marshal console ticket: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign(encodedPayload)
	return encodedPayload + "." + signature, ticket, nil
}

// Consume validates a signed console ticket.
// token is the signed value received on the WebSocket URL.
// namespace is optional; when present it must match the ticket target.
// vmName is the route target that must match the ticket.
// The returned ticket identifies the authenticated user that created the ticket.
func (s *ConsoleTicketService) Consume(token string, namespace string, vmName string, now time.Time) (ConsoleTicket, error) {
	if len(s.secret) == 0 {
		return ConsoleTicket{}, fmt.Errorf("console ticket secret is not configured")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return ConsoleTicket{}, fmt.Errorf("console ticket format is invalid")
	}
	if !hmac.Equal([]byte(parts[1]), []byte(s.sign(parts[0]))) {
		return ConsoleTicket{}, fmt.Errorf("console ticket signature is invalid")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ConsoleTicket{}, fmt.Errorf("decode console ticket: %w", err)
	}

	var ticket ConsoleTicket
	if err := json.Unmarshal(payload, &ticket); err != nil {
		return ConsoleTicket{}, fmt.Errorf("parse console ticket: %w", err)
	}

	if ticket.Purpose != consoleTicketPurpose {
		return ConsoleTicket{}, fmt.Errorf("console ticket purpose is invalid")
	}
	if !ticket.ExpiresAt.After(now.UTC()) {
		return ConsoleTicket{}, fmt.Errorf("console ticket expired")
	}
	if namespace != "" && ticket.Namespace != namespace {
		return ConsoleTicket{}, fmt.Errorf("console ticket target mismatch")
	}
	if ticket.VMName != vmName {
		return ConsoleTicket{}, fmt.Errorf("console ticket target mismatch")
	}

	return ticket, nil
}

// sign creates the HMAC-SHA256 signature for a base64url ticket payload.
// payload is the encoded ticket body that is sent to the browser.
// The returned string is compared during WebSocket ticket validation.
func (s *ConsoleTicketService) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
