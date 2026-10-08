package gomaxpages

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	maxClient "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	template "github.com/max-messenger/max-message-template-go"
	"github.com/max-messenger/maxbot"
)

var safePayload = regexp.MustCompile(`^[a-zA-Z0-9_\-/]+$`)

const (
	startPagePayload = "start"
	fileExt          = ".md"
)

type UploadAPI interface {
	Upload(ctx context.Context, uploadType model.UploadType, r io.Reader, name string, size int64) (token string, err error)
}

type API interface {
	Send(ctx context.Context, msg *maxClient.Message) (model.SendMessageResult, error)
	EditMessage(ctx context.Context, messageID string, body model.NewMessageBody) (model.SimpleQueryResult, error)
}

var (
	ErrContextDirEmpty  = fmt.Errorf("context directory is empty")
	ErrTraversalAttempt = fmt.Errorf("path traversal attempt")
	ErrNotFound         = fmt.Errorf("page not found")
)

type Pages struct {
	contextDir string
	values     template.Values
}

func New(contentDir string, uploader UploadAPI) (*Pages, error) {
	if contentDir == "" {
		return nil, ErrContextDirEmpty
	}

	uploadContext, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	values, err := files(uploadContext, uploader, contentDir)
	if err != nil {
		return nil, err
	}

	return &Pages{
		contextDir: contentDir,
		values:     values,
	}, nil
}

func (p *Pages) Handle(ctx maxbot.Context) error {
	handle := ctx.Edit
	payload := ctx.Update().Payload
	if payload == "" {
		payload = startPagePayload
		handle = ctx.Send
	}

	body, attachments, err := p.sendOrEdit(payload)
	if err != nil {
		return err
	}

	return handle(body, maxbot.WithFormat(model.FormatMarkdown), maxbot.WithAttachments(attachments))
}

func (p *Pages) HandleApi(ctx context.Context, api API, update model.Update) error {
	var isSend bool
	payload := update.Payload
	if payload == "" {
		payload = startPagePayload
		isSend = true
	}

	body, attachments, err := p.sendOrEdit(payload)
	if err != nil {
		return err
	}

	msg := maxClient.NewMessage()
	msg.SetChat(update.ChatID)
	msg.SetText(body)
	msg.SetFormat(model.FormatMarkdown)
	msg.AddAttachments(attachments)

	if isSend {
		_, err = api.Send(ctx, msg)

		return err
	}

	messageBody := msg.MessageBody()
	if attachments != nil {
		messageBody.Attachments = attachments
	}

	_, err = api.EditMessage(ctx, update.MessageID, messageBody)

	return err
}

func (p *Pages) sendOrEdit(payload string) (body string, attachments []model.Attachment, err error) {
	sanitized := p.sanitize(payload)
	if sanitized == "" {
		return
	}

	pagePath := p.filePath(sanitized)

	absPage, err := filepath.Abs(pagePath)
	if err != nil {
		return
	}
	absRoot, err := filepath.Abs(p.contextDir)
	if err != nil {
		return
	}
	if !strings.HasPrefix(absPage, absRoot+string(os.PathSeparator)) {
		err = ErrTraversalAttempt

		return
	}

	content, err := os.ReadFile(pagePath)
	if err != nil {
		err = ErrNotFound

		return
	}

	values := maps.Clone(p.values)

	body, attachments = template.ParseTemplate(string(content), values)

	return
}

func (p *Pages) sanitize(payload string) string {
	if !safePayload.MatchString(payload) {
		return ""
	}
	// отсекаем попытки выхода вверх
	for _, part := range strings.Split(payload, "/") {
		if part == ".." || part == "" {
			return ""
		}
	}

	return payload
}

func (p *Pages) filePath(payload string) string {
	return filepath.Join(p.contextDir, payload+fileExt)
}
