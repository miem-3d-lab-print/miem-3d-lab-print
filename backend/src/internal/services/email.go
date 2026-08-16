package services

import (
	"encoding/base64"
	"fmt"
	"net/smtp"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/config"
)

type EmailService struct {
	cfg config.SMTPConfig
}

func NewEmailService(cfg config.SMTPConfig) *EmailService {
	return &EmailService{cfg: cfg}
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

func (s *EmailService) SendApplicationCreated(to, number string) error {
	subject := fmt.Sprintf("Заявка принята, №%s", number)
	body := fmt.Sprintf("Ваша заявка №%s принята в обработку.\n\nМы уведомим вас об изменении статуса.", number)
	return s.send(to, subject, body)
}

func (s *EmailService) SendNewApplicationToAdmin(
	to, number, applicantName, applicantEmail, materialName, desiredDate, purpose string,
) error {
	subject := fmt.Sprintf("Новая заявка №%s", number)
	body := fmt.Sprintf(
		"Поступила новая заявка №%s.\n\nЗаявитель: %s\nEmail: %s\nМатериал: %s\nЖелаемая дата: %s\nЦель: %s",
		number, applicantName, applicantEmail, materialName, desiredDate, purpose,
	)
	return s.send(to, subject, body)
}

func (s *EmailService) SendStatusChanged(to, number, status, rejectionReason string) error {
	var subject, body string
	switch status {
	case "in_review":
		subject = fmt.Sprintf("Заявка №%s на рассмотрении", number)
		body = fmt.Sprintf("Ваша заявка №%s принята на рассмотрение.", number)
	case "printing":
		subject = fmt.Sprintf("Заявка №%s в печати", number)
		body = fmt.Sprintf("Ваша заявка №%s передана в печать.", number)
	case "ready":
		subject = fmt.Sprintf("Заявка №%s готова к выдаче", number)
		body = fmt.Sprintf("Ваша заявка №%s готова. Заберите её в ауд. 211.", number)
	case "issued":
		subject = fmt.Sprintf("Заявка №%s выдана", number)
		body = fmt.Sprintf("Ваша заявка №%s выдана. Спасибо!", number)
	case "rejected":
		subject = fmt.Sprintf("Заявка №%s отклонена", number)
		body = fmt.Sprintf("Ваша заявка №%s отклонена.\n\nПричина: %s", number, rejectionReason)
	default:
		return nil
	}
	return s.send(to, subject, body)
}
