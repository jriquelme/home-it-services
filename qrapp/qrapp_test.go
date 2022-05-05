package qrapp

import (
	"context"
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/psanford/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	mfs        *memfs.FS
	ctxMatcher = mock.MatchedBy(func(ctx context.Context) bool { return true })
)

func init() {
	mfs = memfs.New()
	err := filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".json") {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				panic(err)
			}
			err = mfs.WriteFile(filepath.Base(path), b, 0755)
			if err != nil {
				panic(err)
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func testingMsg(name string) (*Message, error) {
	b, err := ioutil.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		return nil, err
	}
	msg := &Message{}
	err = json.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func TestQRApp_HandlerNoAttachment(t *testing.T) {
	t.Parallel()

	// get testing mail notification
	msg, err := testingMsg("snsemail-no-attachment.json")
	require.Nil(t, err)
	expectedEmailKey := msg.Receipt.Action.ObjectKey
	expectedEmailBucket := msg.Receipt.Action.BucketName
	expectedMessageID := msg.Mail.CommonHeaders.MessageID
	expectedReturnPath := msg.Mail.CommonHeaders.ReturnPath
	expectedSubject := msg.Mail.CommonHeaders.Subject
	expectedEmailAddr := msg.Receipt.Recipients[0]

	// mock email downloading
	storage := &MockStorage{}
	emailFile, err := mfs.Open(expectedEmailKey)
	require.Nil(t, err)
	defer emailFile.Close()
	storage.On("DownloadToTmpFile", ctxMatcher, expectedEmailBucket, expectedEmailKey).Return(emailFile, nil)
	storage.On("RemoveTmpFile", ctxMatcher, emailFile).Return(nil)
	// mock email reply
	mailer := &MockMailer{}
	mailer.On("SendReply", ctxMatcher, expectedMessageID, expectedEmailAddr, expectedReturnPath, expectedSubject,
		"olvidaste los adjuntos!", "<p>olvidaste los <b>adjuntos</b>!</p>").Return(nil)

	// SUT
	q := &QRApp{
		Storage:        storage,
		Mailer:         mailer,
		FilesBucket:    "qr.mydomain.com",
		FilesBucketURL: "http://qr.mydomain.com",
	}
	// test
	err = q.ProcessEmail(context.Background(), msg)
	assert.Nil(t, err)

	// check mocks
	mock.AssertExpectationsForObjects(t, storage, mailer)
}

func TestQRApp_HandlerAttachmentWithBkgImage(t *testing.T) {
	t.Parallel()

	// get testing mail notification
	msg, err := testingMsg("snsemail-with-attachment.json")
	require.Nil(t, err)
	expectedEmailKey := msg.Receipt.Action.ObjectKey
	expectedEmailBucket := msg.Receipt.Action.BucketName
	expectedMessageID := msg.Mail.CommonHeaders.MessageID
	expectedReturnPath := msg.Mail.CommonHeaders.ReturnPath
	expectedSubject := msg.Mail.CommonHeaders.Subject
	expectedEmailAddr := msg.Receipt.Recipients[0]

	// mock email downloading
	storage := &MockStorage{}
	emailFile, err := mfs.Open(expectedEmailKey)
	require.Nil(t, err)
	defer emailFile.Close()
	storage.On("DownloadToTmpFile", ctxMatcher, expectedEmailBucket, expectedEmailKey).Return(emailFile, nil)
	storage.On("RemoveTmpFile", ctxMatcher, emailFile).Return(nil)
	// mock attachment and qr uploading to files bucket
	filesBucket := "qr.mydomain.com"
	expectedFileKey := "historia-social-el-circo.pdf"
	// TODO: we should inspect and make some assertions about the uploaded bytes
	storage.On("Upload", ctxMatcher, filesBucket, expectedFileKey, "application/pdf", mock.Anything).Return(nil)
	expectedQRKey := "historia-social-el-circo.pdf.qr.png"
	storage.On("Upload", ctxMatcher, filesBucket, expectedQRKey, "image/png", mock.Anything).Return(nil)
	// mock email reply
	mailer := &MockMailer{}
	expectedTxt := "historia-social-el-circo.pdf quedó en http://qr.mydomain.com/historia-social-el-circo.pdf. El QR está en http://qr.mydomain.com/historia-social-el-circo.pdf.qr.png."
	expectedHtml := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html>
    <head>
        <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
    </head>
    <body>
<p><a href="http://qr.mydomain.com/historia-social-el-circo.pdf">historia-social-el-circo.pdf</a>: <a href="http://qr.mydomain.com/historia-social-el-circo.pdf.qr.png">código QR</a>.</p>
    </body>
</html>`
	mailer.On("SendReply", ctxMatcher, expectedMessageID, expectedEmailAddr, expectedReturnPath, expectedSubject,
		expectedTxt, expectedHtml).Return(nil)

	// SUT
	q := &QRApp{
		Storage:        storage,
		Mailer:         mailer,
		FilesBucket:    filesBucket,
		FilesBucketURL: "http://qr.mydomain.com",
	}
	// test
	err = q.ProcessEmail(context.Background(), msg)
	assert.Nil(t, err)

	// check mocks
	mock.AssertExpectationsForObjects(t, storage, mailer)
}

func TestQRApp_HandlerMultipleAttachmentsNoBkgImage(t *testing.T) {
	t.Parallel()

	// get testing mail notification
	msg, err := testingMsg("snsemail-multiple-attachments-no-bkg.json")
	require.Nil(t, err)
	expectedEmailKey := msg.Receipt.Action.ObjectKey
	expectedEmailBucket := msg.Receipt.Action.BucketName
	expectedMessageID := msg.Mail.CommonHeaders.MessageID
	expectedReturnPath := msg.Mail.CommonHeaders.ReturnPath
	expectedSubject := msg.Mail.CommonHeaders.Subject
	expectedEmailAddr := msg.Receipt.Recipients[0]

	// mock email downloading
	storage := &MockStorage{}
	emailFile, err := mfs.Open(expectedEmailKey)
	require.Nil(t, err)
	defer emailFile.Close()
	storage.On("DownloadToTmpFile", ctxMatcher, expectedEmailBucket, expectedEmailKey).Return(emailFile, nil)
	storage.On("RemoveTmpFile", ctxMatcher, emailFile).Return(nil)
	// mock attachment and qr uploading to files bucket
	filesBucket := "qr.mydomain.com"
	expectedFileKey1 := "toos-leen.jpeg"
	// TODO: we should inspect and make some assertions about the uploaded bytes
	storage.On("Upload", ctxMatcher, filesBucket, expectedFileKey1, "image/jpeg", mock.Anything).Return(nil)
	expectedQRKey1 := "toos-leen.jpeg.qr.png"
	storage.On("Upload", ctxMatcher, filesBucket, expectedQRKey1, "image/png", mock.Anything).Return(nil)
	expectedFileKey2 := "xp-won.jpeg"
	storage.On("Upload", ctxMatcher, filesBucket, expectedFileKey2, "image/jpeg", mock.Anything).Return(nil)
	expectedQRKey2 := "xp-won.jpeg.qr.png"
	storage.On("Upload", ctxMatcher, filesBucket, expectedQRKey2, "image/png", mock.Anything).Return(nil)
	expectedFileKey3 := "a-text-file"
	storage.On("Upload", ctxMatcher, filesBucket, expectedFileKey3, "application/octet-stream", mock.Anything).Return(nil)
	expectedQRKey3 := "a-text-file.qr.png"
	storage.On("Upload", ctxMatcher, filesBucket, expectedQRKey3, "image/png", mock.Anything).Return(nil)
	// mock email reply
	mailer := &MockMailer{}
	expectedTxt := `* a text file quedó en http://qr.mydomain.com/a-text-file. El QR está en http://qr.mydomain.com/a-text-file.qr.png.
* toos-leen.jpeg quedó en http://qr.mydomain.com/toos-leen.jpeg. El QR está en http://qr.mydomain.com/toos-leen.jpeg.qr.png.
* xp-won.jpeg quedó en http://qr.mydomain.com/xp-won.jpeg. El QR está en http://qr.mydomain.com/xp-won.jpeg.qr.png.
`
	expectedHtml := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html>
    <head>
        <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
    </head>
    <body>
<ol>
<li><a href="http://qr.mydomain.com/a-text-file">a text file</a>: <a href="http://qr.mydomain.com/a-text-file.qr.png">código QR</a>.</li>
<li><a href="http://qr.mydomain.com/toos-leen.jpeg">toos-leen.jpeg</a>: <a href="http://qr.mydomain.com/toos-leen.jpeg.qr.png">código QR</a>.</li>
<li><a href="http://qr.mydomain.com/xp-won.jpeg">xp-won.jpeg</a>: <a href="http://qr.mydomain.com/xp-won.jpeg.qr.png">código QR</a>.</li>
</ol>
    </body>
</html>`
	mailer.On("SendReply", ctxMatcher, expectedMessageID, expectedEmailAddr, expectedReturnPath, expectedSubject,
		expectedTxt, expectedHtml).Return(nil)

	// SUT
	q := &QRApp{
		Storage:        storage,
		Mailer:         mailer,
		FilesBucket:    filesBucket,
		FilesBucketURL: "http://qr.mydomain.com",
	}
	// test
	err = q.ProcessEmail(context.Background(), msg)
	assert.Nil(t, err)

	// check mocks
	mock.AssertExpectationsForObjects(t, storage, mailer)
}

func Test_fileNameSlug(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "no ext",
			arg:  "no extension",
			want: "no-extension",
		},
		{
			name: "simple",
			arg:  "simple.txt",
			want: "simple.txt",
		},
		{
			name: "Capitalization and space",
			arg:  "My File.pdf",
			want: "my-file.pdf",
		},
		{
			name: "yo hablo español",
			arg:  "Tú hablas también?.png",
			want: "tu-hablas-tambien.png",
		},
		{
			name: "two dots",
			arg:  "two.dots.pdf",
			want: "two-dots.pdf",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, fileNameSlug(tt.arg), "fileNameSlug(%v)", tt.arg)
		})
	}
}
