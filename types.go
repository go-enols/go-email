package email

type EmailProto int

const (
	IMAP EmailProto = iota
	POP3
	SMTP
)

type LoginParams struct {
	Host  string
	Port  int
	User  string
	Pwd   string
	Proto EmailProto

	ClientId     string // oauth2认证需要
	RefreshToken string // oauth2认证需要
}
