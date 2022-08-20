package email

import (
	"fmt"
	"sync"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

var _ ISender = &SendGridSender{}

const (
	errWrapSendGridSender = "email.SendGridSender Error : "
)

// SendGridSender : send grid email sender
type SendGridSender struct {
	emailCache map[string]*mail.Email // cache
	mu         sync.Mutex
	client     *sendgrid.Client // http api client
}

// NewSendGridSender : email sender that implements i sender (via send_grid api)
func NewSendGridSender(apiKey string) ISender {
	return &SendGridSender{
		emailCache: make(map[string]*mail.Email),
		mu:         sync.Mutex{},
		client:     sendgrid.NewSendClient(apiKey),
	}
}

// SendEmail : Sends a plain text email to the client
func (s *SendGridSender) SendEmail(c *Content) error {
	fromEmailCacheKey := s.toEmailKey(c.FromUserFriendlyName, c.From)
	toEmailCacheKey := s.toEmailKey(c.ToFriendlyName, c.To)

	from := s.emailCache[fromEmailCacheKey]

	if from == nil {
		// create a new one
		from = mail.NewEmail(c.FromUserFriendlyName, c.From)
		s.mu.Lock()
		s.emailCache[fromEmailCacheKey] = from
		s.mu.Unlock()
	}

	to := s.emailCache[toEmailCacheKey]

	if to == nil {
		// create a new one
		to = mail.NewEmail(c.ToFriendlyName, c.To)
		s.mu.Lock()
		s.emailCache[toEmailCacheKey] = to
		s.mu.Unlock()
	}
	message := mail.NewSingleEmail(from, c.Subject, to, c.Body, c.Body)
	res, err := s.client.Send(message)
	_ = res
	if err != nil {
		return err
	}

	return nil
}

func (s *SendGridSender) toEmailKey(name string, email string) string {
	return fmt.Sprintf("%s::%s", name, email)
}

// SendHTMLEmail : Sends an email that is formatted with HTML
func (s *SendGridSender) SendHTMLEmail(c *Content) error {
	return s.SendEmail(c)
}
