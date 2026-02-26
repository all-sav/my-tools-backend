package main

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

var (
	HttpPort        string
	WSPort          string
	WSAllowedOrigin string

	GitlabApiUrl       string
	GitlabAccessToken  string
	gitlabWebhookToken string

	BackendProjectID    string
	FrontendProjectID   string
	BackendStandBranch  string
	FrontendStandBranch string
	CIMainBranch        string
	RequiredPrefix      string
	Prefix              string
	CIPrefix            string

	OverProxy  bool
	SSLCertPem string
	SSLKeyPem  string
)

const ()

func setEnvs() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла:", err)
	}

	HttpPort = os.Getenv("HTTP_PORT")
	WSPort = os.Getenv("WS_PORT")
	WSAllowedOrigin = os.Getenv("APP_URL")

	GitlabApiUrl = os.Getenv("GITLAB_API_URL")
	GitlabAccessToken = os.Getenv("GITLAB_ACCESS_TOKEN")
	gitlabWebhookToken = os.Getenv("GITLAB_WEBHOOK_TOKEN")

	BackendProjectID = os.Getenv("BACKEND_PROJECT_ID")
	FrontendProjectID = os.Getenv("FRONTEND_PROJECT_ID")
	BackendStandBranch = os.Getenv("BACKEND_STAND_BRANCH")
	FrontendStandBranch = os.Getenv("FRONTEND_STAND_BRANCH")
	CIMainBranch = os.Getenv("CI_MAIN_BRANCH")
	RequiredPrefix = os.Getenv("REQUIRED_PREFIX")
	Prefix = os.Getenv("PREFIX")
	CIPrefix = os.Getenv("CI_PREFIX")

	OverProxy = os.Getenv("OVER_PROXY") == "true"
	SSLCertPem = os.Getenv("SSL_CERT_PEM")
	SSLKeyPem = os.Getenv("SSL_KEY_PEM")
}
