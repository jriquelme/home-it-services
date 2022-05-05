package qrapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	htmltpl "html/template"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	txttpl "text/template"

	"github.com/gosimple/slug"
	"github.com/jhillyerd/enmime"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type Storage interface {
	// DownloadToTmpFile downloads an object from a bucket to a temporary file.
	DownloadToTmpFile(ctx context.Context, bucket, key string) (fs.File, error)

	// RemoveTmpFile removes a temporary file created by DownloadToTmpFile.
	RemoveTmpFile(ctx context.Context, tmpFile fs.File) error

	// Upload uploads an object to a bucket, using the contents from the reader, setting the given content type.
	Upload(ctx context.Context, bucket, key, contentType string, r io.Reader) error

	// Delete deletes an object from a bucket.
	Delete(ctx context.Context, bucket, key string) error
}

//go:generate mockery --name=Storage --testonly --inpackage --disable-version-string --quiet

type Mailer interface {
	// SendReply sends a reply email.
	SendReply(ctx context.Context, messageID, from, to, subject, text, html string) error
}

//go:generate mockery --name=Mailer --testonly --inpackage --disable-version-string --quiet

type QRApp struct {
	Storage        Storage
	Mailer         Mailer
	FilesBucket    string
	FilesBucketURL string
}

type Message struct {
	NotificationType string `json:"notificationType"`
	Mail             struct {
		CommonHeaders struct {
			ReturnPath string   `json:"returnPath"`
			From       []string `json:"from"`
			MessageID  string   `json:"messageId"`
			Subject    string   `json:"subject"`
		} `json:"commonHeaders"`
	} `json:"mail"`
	Receipt struct {
		Recipients []string `json:"recipients"`
		Action     struct {
			Type       string `json:"type"`
			BucketName string `json:"bucketName"`
			ObjectKey  string `json:"objectKey"`
		} `json:"action"`
	} `json:"receipt"`
}

func (q *QRApp) ProcessEmail(ctx context.Context, msg *Message) error {
	// get email from S3
	bucket := msg.Receipt.Action.BucketName
	key := msg.Receipt.Action.ObjectKey
	tmpFile, err := q.Storage.DownloadToTmpFile(ctx, bucket, key)
	if err != nil {
		return err
	}
	defer q.Storage.RemoveTmpFile(ctx, tmpFile)

	// extract attachments from email
	envelope, err := enmime.ReadEnvelope(tmpFile)
	if err != nil {
		return fmt.Errorf("couldn't read email: %s", err)
	}
	// no attachments, send an email reply with an error message
	if len(envelope.Attachments) == 0 {
		text := "olvidaste los adjuntos!"
		html := "<p>olvidaste los <b>adjuntos</b>!</p>"
		ch := msg.Mail.CommonHeaders
		if len(msg.Receipt.Recipients) == 0 {
			return errors.New("missing receipt.recipients from message")
		}
		err := q.Mailer.SendReply(ctx, ch.MessageID, msg.Receipt.Recipients[0], ch.ReturnPath, ch.Subject, text, html)
		if err != nil {
			return err
		}
		return nil
	}
	// separate images from other file types
	imgAttachments := make([]*enmime.Part, 0, len(envelope.Attachments))
	docAttachments := make([]*enmime.Part, 0, len(envelope.Attachments))
	for _, attch := range envelope.Attachments {
		log.Printf("attachment %s %s", attch.FileName, attch.ContentType)
		switch attch.ContentType {
		case "image/jpeg", "image/png":
			imgAttachments = append(imgAttachments, attch)
		default:
			docAttachments = append(docAttachments, attch)
		}
	}
	// analyze attachments and check if we have to use a background image
	var attachments []*enmime.Part
	var bkgImg string
	switch {
	case len(imgAttachments) == 1 && len(docAttachments) > 0:
		// a single image detected with additional files (use the image as background)
		attachments = docAttachments
		imgAttch := imgAttachments[0]
		bkgImg = filepath.Join(os.TempDir(), fileNameSlug(imgAttch.FileName))
		err := os.WriteFile(bkgImg, imgAttch.Content, 0666)
		if err != nil {
			return err
		}
	default:
		// for all the remaining cases generate a QR for every attachment regardless of the file type
		attachments = docAttachments
		attachments = append(attachments, imgAttachments...)
	}
	// generate QR code for all attachments
	results := make(chan ProcessingResult, len(attachments))
	wg := &sync.WaitGroup{}
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	for _, attachment := range attachments {
		attachment := attachment
		wg.Add(1)
		go func() {
			defer wg.Done()
			attachmentURL, qrImageURL, err := q.processAttachment(ctx, attachment, bkgImg)
			results <- ProcessingResult{
				AttachmentName: attachment.FileName,
				AttachmentURL:  attachmentURL,
				QRImageURL:     qrImageURL,
				Error:          err,
			}
		}()
	}
	wg.Wait()
	if err != nil {
		return err
	}
	close(results)
	// send response email
	err = q.sendReply(ctx, results, msg)
	if err != nil {
		return err
	}
	return nil
}

type ProcessingResult struct {
	AttachmentName string
	AttachmentURL  string
	QRImageURL     string
	Error          error
}

func (q *QRApp) processAttachment(ctx context.Context, attachment *enmime.Part, bkgImg string) (attachmentURL string, qrImgURL string, err error) {
	// upload attachment to FilesBucket
	attachmentKey := fileNameSlug(attachment.FileName)
	err = q.Storage.Upload(ctx, q.FilesBucket, attachmentKey, attachment.ContentType, bytes.NewReader(attachment.Content))
	if err != nil {
		return
	}
	defer func() {
		// remove attachment from bucket if the whole operation isn't successful
		if err != nil {
			deleteErr := q.Storage.Delete(context.Background(), q.FilesBucket, attachmentKey)
			if deleteErr != nil {
				log.Printf("couldn't delete %s from %s: %s", attachmentKey, q.FilesBucket, deleteErr)
			}
		}
	}()
	attachmentURL, err = q.filesStaticWebsiteURL(attachmentKey)
	if err != nil {
		return
	}
	// generate QR code
	qrImgKey := attachmentKey + ".qr.png"
	outputImg := filepath.Join(os.TempDir(), qrImgKey)
	err = q.generateQR(attachmentURL, bkgImg, outputImg)
	if err != nil {
		return
	}
	defer os.Remove(outputImg)
	// upload QR code to FilesBucket
	r, err := os.Open(outputImg)
	if err != nil {
		return
	}
	defer r.Close()
	qrImgURL, err = q.filesStaticWebsiteURL(qrImgKey)
	if err != nil {
		return
	}
	err = q.Storage.Upload(ctx, q.FilesBucket, qrImgKey, "image/png", r)
	return
}

func (q *QRApp) generateQR(url, bkgImg, outputImg string) error {
	qrCode, err := qrcode.New(url)
	if err != nil {
		return err
	}
	options := make([]standard.ImageOption, 0, 2)
	options = append(options, standard.WithQRWidth(21))
	if bkgImg != "" {
		options = append(options, standard.WithHalftone(bkgImg))
	}
	w, err := standard.New(outputImg, options...)
	if err != nil {
		return err
	}
	err = qrCode.Save(w)
	if err != nil {
		return err
	}
	return nil
}

var (
	txtReplyTpl  *txttpl.Template
	htmlReplyTpl *htmltpl.Template
)

func init() {
	txtReplyTpl = txttpl.Must(txttpl.New("txtReply").Parse(`
{{- define "result" -}}
	{{- if .Error -}}
No pude generar el código QR de {{.AttachmentName}}: {{.Error}}
	{{- else -}}
{{.AttachmentName}} quedó en {{.AttachmentURL}}. El QR está en {{.QRImageURL}}.
	{{- end -}}
{{- end -}}

{{if eq (len .) 1}}
{{- template "result" (index . 0) -}}
{{else}}
{{- range $v := . -}}
* {{template "result" $v}}
{{ end -}}
{{end}}`))
	htmlReplyTpl = htmltpl.Must(htmltpl.New("htmlReply").Parse(`
{{- define "result" -}}
	{{- if .Error -}}
No pude generar el código QR de {{.AttachmentName}}: {{.Error}}
	{{- else -}}
<a href="{{.AttachmentURL}}">{{.AttachmentName}}</a>: <a href="{{.QRImageURL}}">código QR</a>.
	{{- end -}}
{{- end -}}
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html>
    <head>
        <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
    </head>
    <body>
{{if eq (len .) 1 -}}
<p>{{- template "result" (index . 0) -}}</p>
{{- else -}}
<ol>
{{range $v := . -}}
<li>{{template "result" $v}}</li>
{{ end -}}
</ol>
{{- end}}
    </body>
</html>`))
}

func (q *QRApp) sendReply(ctx context.Context, results <-chan ProcessingResult, msg *Message) error {
	// collect results in a slice
	var resultsSlice []ProcessingResult
	for result := range results {
		resultsSlice = append(resultsSlice, result)
	}
	// sort results to get a consistent output
	sort.Slice(resultsSlice, func(i, j int) bool {
		return resultsSlice[i].AttachmentName < resultsSlice[j].AttachmentName
	})
	// evaluate txt/html message templates
	text := &bytes.Buffer{}
	err := txtReplyTpl.Execute(text, resultsSlice)
	if err != nil {
		return err
	}
	html := &bytes.Buffer{}
	err = htmlReplyTpl.Execute(html, resultsSlice)
	if err != nil {
		return err
	}
	// send email
	ch := msg.Mail.CommonHeaders
	if len(msg.Receipt.Recipients) == 0 {
		return errors.New("missing receipt.recipients from message")
	}
	err = q.Mailer.SendReply(ctx, ch.MessageID, msg.Receipt.Recipients[0], ch.ReturnPath, ch.Subject, text.String(), html.String())
	if err != nil {
		return err
	}
	return nil
}

func (q *QRApp) filesStaticWebsiteURL(key string) (string, error) {
	// from https://stackoverflow.com/questions/34668012/combine-url-paths-with-path-join
	keyURL, err := url.Parse(q.FilesBucketURL)
	if err != nil {
		return "", err
	}
	keyURL.Path = path.Join(keyURL.Path, key)
	return keyURL.String(), nil
}

func fileNameSlug(name string) string {
	ext := filepath.Ext(name)
	withoutExt := name[:len(name)-len(ext)]
	return slug.Make(withoutExt) + ext
}
