package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type GitLabWebhook struct {
	ObjectKind string `json:"object_kind"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Ref        string `json:"ref"`
	Branch     string `json:"branch"`
	Project    struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		URL  string `json:"web_url"`
	} `json:"project"`
	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commits"`
}

func handleWebhook(c *gin.Context) {
	_, err := validateToken(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	//rawData, _ := c.GetRawData()
	//fmt.Println("Сырые данные:", string(rawData))

	var webhookData GitLabWebhook

	if err := c.ShouldBindJSON(&webhookData); err != nil {
		c.String(http.StatusBadRequest, "Ошибка парсинга JSON: "+err.Error())
		return
	}

	webhookData.Branch = strings.TrimPrefix(webhookData.Ref, "refs/heads/")

	if err := validateBranchPrefix(webhookData.Branch); err != nil {
		log.Println(err)
		c.Status(http.StatusOK)
		return
	}

	if webhookData.ObjectKind == "push" {
		onPush(webhookData)
	}

	c.Status(http.StatusOK)
}

func validateToken(c *gin.Context) (bool, error) {
	token := c.GetHeader("X-Gitlab-Token")
	if token != gitlabWebhookToken {
		return false, errors.New("неверный токен")
	}

	return true, nil
}

func onPush(webhookData GitLabWebhook) {
	ciBranch := makeCIBranchName(webhookData.Branch)
	projectID := strconv.Itoa(webhookData.Project.ID)

	// Проверка существования CI-ветки
	log.Printf("CI: %s Pr: %s", ciBranch, projectID)
	ciExists, err := branchExistsInRepo(ciBranch, projectID)
	if err != nil {
		log.Printf("Ошибка проверки CI‑ветки: %v", err)
		return
	}
	if !ciExists {
		log.Println("CI-ветка не найдена")
		return
	}

	// Проверка открытого MR для CI‑ветки
	hasMR, _, _, err := hasOpenMR(ciBranch, getStandBranchByProjectID(projectID), projectID)
	if err != nil {
		log.Printf("Ошибка проверки MR для CI‑ветки: %v", err)
		return
	}
	if !hasMR {
		log.Println("не найден MR для CI-ветки")
		return
	}

	// Мержим исходную ветку в CI-ветку
	hasMR, mrID, _, err := hasOpenMR(webhookData.Branch, ciBranch, projectID)
	if err != nil {
		log.Printf("ошибка проверки существующих MR: %v", err)
		return
	}
	if !hasMR {
		mrID, err = mergeBranchInto(webhookData.Branch, ciBranch, projectID)
		if err != nil {
			log.Printf("не удалось создать MR: %v", err)
			return
		}
		log.Printf("Засыпаем...")
		time.Sleep(3 * time.Second)
		log.Printf("Просыпаемся...")
	}
	if err := acceptMergeRequest(mrID, projectID); err != nil {
		log.Printf("не удалось принять MR %d: %v", mrID, err)
	}
}
