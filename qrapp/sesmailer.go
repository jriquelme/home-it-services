package qrapp

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/jhillyerd/enmime"
)

type SESMailer struct {
	SESClient *ses.Client
}

func (sm *SESMailer) SendReply(ctx context.Context, messageID, from, to, subject, text, html string) error {
	mailBuilder := enmime.Builder().Subject(subject).From("QR App", from).ReplyTo("", to).To("", to).
		Header("In-Reply-To", messageID).Header("References", messageID).Text([]byte(text)).HTML([]byte(html))
	part, err := mailBuilder.Build()
	if err != nil {
		return fmt.Errorf("error building email: %s", err)
	}
	mailBytes := &bytes.Buffer{}
	err = part.Encode(mailBytes)
	if err != nil {
		return fmt.Errorf("error building email: %s", err)
	}
	email, err := sm.SESClient.SendRawEmail(ctx, &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: mailBytes.Bytes(),
		},
	})
	if err != nil {
		return fmt.Errorf("couldn't send email with SES: %s", err)
	}
	log.Printf("response email send: %s", *email.MessageId)
	return nil
}
