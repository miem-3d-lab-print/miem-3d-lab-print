package services

import (
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/config"
)

type EmailService struct {
	cfg     config.SMTPConfig
	siteURL string
}

func NewEmailService(cfg config.SMTPConfig, siteURL string) *EmailService {
	return &EmailService{cfg: cfg, siteURL: strings.TrimRight(siteURL, "/")}
}

func (s *EmailService) applicationURL(applicationID string, admin bool) string {
	if admin {
		return fmt.Sprintf("%s/admin/applications/%s", s.siteURL, applicationID)
	}
	return fmt.Sprintf("%s/app/applications/%s", s.siteURL, applicationID)
}

func (s *EmailService) send(to, subject, body string) error {
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	encodedSubject := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
	msg := []byte(
		"From: " + s.cfg.From + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + encodedSubject + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body + "\r\n",
	)
	addr := s.cfg.Host + ":" + s.cfg.Port
	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, msg)
}

func (s *EmailService) SendOTP(to, code string) error {
	subject := "Ваш код подтверждения. Никому не сообщайте его!"
	body := fmt.Sprintf("Ваш код: %s\n\nКод действителен %d минут.", code, int(otpExpiresIn.Minutes()))
	return s.send(to, subject, body)
}

func (s *EmailService) SendApplicationCreated(to, number, title, applicationID string) error {
	subject := fmt.Sprintf("Заявка «%s» принята, №%s", title, number)
	body := fmt.Sprintf("Ваша заявка «%s», №%s принята в обработку.\n\nОткрыть заявку: %s\n\nМы уведомим вас об изменении статуса.", title, number, s.applicationURL(applicationID, false))
	return s.send(to, subject, body)
}

func (s *EmailService) SendNewApplicationToAdmin(
	to, number, title, applicationID, applicantName, applicantEmail, materialName, desiredDate, purpose string,
) error {
	subject := fmt.Sprintf("Новая заявка «%s», №%s", title, number)
	body := fmt.Sprintf(
		"Поступила новая заявка «%s», №%s.\n\nОткрыть заявку: %s\n\nЗаявитель: %s\nEmail: %s\nМатериал: %s\nЖелаемая дата: %s\nЦель: %s",
		title, number, s.applicationURL(applicationID, true), applicantName, applicantEmail, materialName, desiredDate, purpose,
	)
	return s.send(to, subject, body)
}

func (s *EmailService) SendStatusChanged(to, number, title, applicationID, status, rejectionReason string) error {
	var subject, body string
	switch status {
	case "in_review":
		subject = fmt.Sprintf("Заявка «%s», №%s на рассмотрении", title, number)
		body = fmt.Sprintf("Ваша заявка «%s», №%s принята на рассмотрение.", title, number)
	case "printing":
		subject = fmt.Sprintf("Заявка «%s», №%s в печати", title, number)
		body = fmt.Sprintf("Ваша заявка «%s», №%s передана в печать.", title, number)
	case "ready":
		subject = fmt.Sprintf("Заявка «%s», №%s готова к выдаче", title, number)
		body = fmt.Sprintf("Ваша заявка «%s», №%s готова. Заберите её в ауд. 211.", title, number)
	case "issued":
		subject = fmt.Sprintf("Заявка «%s», №%s выдана", title, number)
		body = fmt.Sprintf("Ваша заявка «%s», №%s выдана. Спасибо!", title, number)
	case "rejected":
		subject = fmt.Sprintf("Заявка «%s», №%s отклонена", title, number)
		body = fmt.Sprintf("Ваша заявка «%s», №%s отклонена.\n\nПричина: %s", title, number, rejectionReason)
	default:
		return nil
	}
	body += fmt.Sprintf("\n\nОткрыть заявку: %s", s.applicationURL(applicationID, false))
	return s.send(to, subject, body)
}
