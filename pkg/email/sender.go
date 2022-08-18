package email

// Content : email information
type Content struct {
	FromUserFriendlyName string
	From                 string
	To                   string
	ToFriendlyName       string
	CC                   []string
	Subject              string
	Body                 string
}

// ISender : Email Sender Generic interface that basically can be implemented for many email providers
//           Forexample => SendGrid , SES , SNS ...etc
type ISender interface {
	// SendEmail : Sends a plain text email to the client
	SendEmail(c *Content) error
	// SendHTMLEmail : Sends an email that is formatted with HTML
	SendHTMLEmail(c *Content) error
}
