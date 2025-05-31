package mail

import (
	"backend/config"
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/jordan-wright/email"
)

type MailService struct {
	config *config.Config
	// 用于防止短时间内重复发送相同邮件
	sentMails sync.Map
}

func NewMailService() *MailService {
	return &MailService{
		config: config.GetConfig(),
	}
}

// shouldRetry 判断是否应该重试
func (s *MailService) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "i/o timeout")
}

// preventDuplicateSend 防止短时间内重复发送相同邮件
func (s *MailService) preventDuplicateSend(mailKey string) bool {
	key := fmt.Sprintf("%s_%d", mailKey, time.Now().Unix()/300) // 5分钟内相同邮件只发送一次
	_, loaded := s.sentMails.LoadOrStore(key, true)
	go func() {
		time.Sleep(5 * time.Minute)
		s.sentMails.Delete(key)
	}()
	return !loaded
}

// sendMailInternal 内部邮件发送函数
func (s *MailService) sendMailInternal(e *email.Email) error {
	addr := fmt.Sprintf("%s:%d", s.config.Mail.Host, s.config.Mail.Port)
	auth := smtp.PlainAuth("", s.config.Mail.Username, s.config.Mail.Password, s.config.Mail.Host)

	tlsConfig := &tls.Config{
		ServerName:         s.config.Mail.Host,
		InsecureSkipVerify: true, // 注意：生产环境建议为 false
		MinVersion:         tls.VersionTLS12,
	}

	// 只允许特定端口和协议组合，避免自动切换
	if s.config.Mail.UseTLS {
		switch s.config.Mail.Port {
		case 465:
			// 465 端口只允许 SSL/TLS
			return e.SendWithTLS(addr, auth, tlsConfig)
		case 587:
			// 587 端口只允许 STARTTLS
			return e.SendWithStartTLS(addr, auth, tlsConfig)
		default:
			return fmt.Errorf("不支持的端口和TLS组合: 端口%d UseTLS=%v", s.config.Mail.Port, s.config.Mail.UseTLS)
		}
	} else {
		// 非加密，通常只用于 25 端口
		if s.config.Mail.Port == 25 {
			return e.Send(addr, auth)
		}
		return fmt.Errorf("不支持的端口和非TLS组合: 端口%d UseTLS=%v", s.config.Mail.Port, s.config.Mail.UseTLS)
	}
}

// SendMail 发送邮件
func (s *MailService) SendMail(to string, subject string, content string) error {
	mailKey := fmt.Sprintf("%s_%s_%s", to, subject, content[:20])
	if !s.preventDuplicateSend(mailKey) {
		return fmt.Errorf("检测到重复发送请求，已跳过")
	}

	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", s.config.Mail.FromName, s.config.Mail.FromAddress)
	e.To = []string{to}
	e.Subject = subject
	e.HTML = []byte(content)

	err := s.sendMailInternal(e)
	if err != nil && s.shouldRetry(err) {
		// 仅对特定错误进行一次重试
		time.Sleep(2 * time.Second)
		err = s.sendMailInternal(e)
	}
	if isShortResponseError(err) {
		// 记录警告，但视为成功
		fmt.Printf("[邮件警告] 发送成功但响应异常: %v\n", err)
		return nil
	}
	if err != nil {
		return fmt.Errorf("发送邮件失败: %v", err)
	}
	return nil
}

// SendHTMLMail 发送HTML格式邮件
func (s *MailService) SendHTMLMail(to []string, subject, htmlContent string) error {
	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", s.config.Mail.FromName, s.config.Mail.FromAddress)
	e.To = to
	e.Subject = subject
	e.HTML = []byte(htmlContent)

	err := s.sendMailInternal(e)
	if err != nil && s.shouldRetry(err) {
		// 仅对特定错误进行一次重试
		time.Sleep(2 * time.Second)
		err = s.sendMailInternal(e)
	}
	if isShortResponseError(err) {
		fmt.Printf("[邮件警告] 发送成功但响应异常: %v\n", err)
		return nil
	}
	if err != nil {
		return fmt.Errorf("发送HTML邮件失败: %v", err)
	}
	return nil
}

// SendInvitationCode 发送邀请码邮件
func (s *MailService) SendInvitationCode(to string, code string, expiresAt *time.Time) error {
	// 邀请码邮件模板
	const invitationTemplate = `
	<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
		<h2 style="color: #333;">邀请码通知</h2>
		<p style="font-size: 16px; line-height: 1.5;">亲爱的用户：</p>
		<p style="font-size: 16px; line-height: 1.5;">您的邀请码已准备就绪，请查收：</p>
		<div style="background-color: #f5f5f5; padding: 15px; margin: 20px 0; border-radius: 5px;">
			<p style="font-size: 24px; font-weight: bold; text-align: center; color: #007bff;">{{.Code}}</p>
		</div>
		{{if .ExpiresAt}}
		<p style="font-size: 14px; color: #666;">请注意：此邀请码将在 {{.ExpiresAt}} 过期。</p>
		{{end}}
		<p style="font-size: 14px; line-height: 1.5;">使用说明：</p>
		<ol style="font-size: 14px; line-height: 1.5;">
			<li>访问<a href="https://mi.jamyido.cn/register">https://mi.jamyido.cn/register</a> 注册页面</li>
			<li>填写注册信息</li>
			<li>在邀请码输入框中输入上方的邀请码</li>
		</ol>
		<hr style="margin: 24px 0;">
		<div style="text-align: center;">
			<p style="font-size: 15px; color: #333; font-weight: bold;">加入一测QQ群获取最新动态与交流：</p>
			<img src='https://fastly.jsdelivr.net/gh/Bangumoe/static_files@master/commen/qrcode_1748690072135.jpg' alt='QQ群二维码' style='width:220px;max-width:100%;border-radius:12px;border:1px solid #eee;box-shadow:0 2px 8px #eee;'>
			<p style="font-size: 13px; color: #666;">QQ群号：1041349925<br>（扫码或搜索群号加入，欢迎反馈建议！）</p>
		</div>
		<p style="font-size: 12px; color: #999; margin-top: 20px;">此邮件由系统自动发送，请勿回复。</p>
	</div>
	`

	// 准备模板数据
	data := struct {
		Code      string
		ExpiresAt string
	}{
		Code:      code,
		ExpiresAt: "",
	}

	if expiresAt != nil {
		data.ExpiresAt = expiresAt.Format("2006-01-02 15:04:05")
	}

	// 解析和执行模板
	tmpl, err := template.New("invitation").Parse(invitationTemplate)
	if err != nil {
		return fmt.Errorf("解析邀请码邮件模板失败: %v", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("生成邀请码邮件内容失败: %v", err)
	}

	// 发送邮件
	return s.SendMail(to, "您的邀请码已就绪", body.String())
}

// SendCustomMail 发送自定义邮件
func (s *MailService) SendCustomMail(to []string, subject string, content string, isHTML bool) error {
	if isHTML {
		return s.SendHTMLMail(to, subject, content)
	}

	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", s.config.Mail.FromName, s.config.Mail.FromAddress)
	e.To = to
	e.Subject = subject
	if isHTML {
		e.HTML = []byte(content)
	} else {
		e.Text = []byte(content)
	}

	addr := fmt.Sprintf("%s:%d", s.config.Mail.Host, s.config.Mail.Port)
	auth := smtp.PlainAuth("", s.config.Mail.Username, s.config.Mail.Password, s.config.Mail.Host)

	var err error
	if s.config.Mail.UseTLS {
		tlsConfig := &tls.Config{
			ServerName:         s.config.Mail.Host,
			InsecureSkipVerify: true, // 在开发环境中可以使用
			MinVersion:         tls.VersionTLS12,
		}
		err = e.SendWithTLS(addr, auth, tlsConfig)
	} else {
		err = e.Send(addr, auth)
	}

	if err != nil {
		return fmt.Errorf("发送自定义邮件失败: %v", err)
	}
	return nil
}

func isShortResponseError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "short response") ||
		strings.Contains(err.Error(), "\u0000\u0000\u0000")
}
