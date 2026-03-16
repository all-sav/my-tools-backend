package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mergenator/internal/infr/logger"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
)

type client struct {
	apiURL     string
	token      string
	httpClient *http.Client
	log        *zerolog.Logger
}

type GitLabUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func NewClient(apiURL, token string) GitLabClient {
	return &client{
		apiURL:     apiURL,
		token:      token,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		log:        logger.Get(),
	}
}

func (c *client) BranchExists(ctx context.Context, branch, projectID string) (bool, error) {
	escapedBranch := url.PathEscape(branch)
	url := fmt.Sprintf("%s/projects/%s/repository/branches/%s", c.apiURL, projectID, escapedBranch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("API error: %d %s", resp.StatusCode, string(body))
	}
}

func (c *client) HasOpenMR(ctx context.Context, source, target string, projectID string) (bool, int, string, error) {
	mrUrl := fmt.Sprintf(
		"%s/projects/%s/merge_requests?source_branch=%s&target_branch=%s&state=opened",
		c.apiURL, projectID, source, target)

	req, err := http.NewRequest("GET", mrUrl, nil)
	if err != nil {
		return false, 0, "", err
	}

	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// bodyString := string(body)
	// log.Printf("Ответ сервера (JSON):\n%s", bodyString)

	if resp.StatusCode != http.StatusOK {
		return false, 0, "", fmt.Errorf("API ошибка проверки MR: %d %s", resp.StatusCode, string(body))
	}

	var mrs []struct {
		ID     int    `json:"iid"`
		WebURL string `json:"web_url"`
	}
	if err := json.Unmarshal(body, &mrs); err != nil {
		return false, 0, "", err
	}

	if len(mrs) > 0 {
		return true, mrs[0].ID, mrs[0].WebURL, nil
	}

	return false, 0, "", nil
}

// Создание MR
func (c *client) CreateMR(ctx context.Context, sourceBranch, title string, repo Repository) (string, error) {
	mrUrl := fmt.Sprintf("%s/projects/%s/merge_requests", c.apiURL, repo.ProjectID)

	data := map[string]interface{}{
		"source_branch":        sourceBranch,
		"target_branch":        repo.StandBranch,
		"title":                title,
		"assignee_ids":         []int{repo.AssigneeID},
		"remove_source_branch": true,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", mrUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API ошибка: %d %s", resp.StatusCode, string(body))
	}

	var mrResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&mrResponse); err != nil {
		return "", err
	}

	mrURL, ok := mrResponse["web_url"].(string)
	if !ok {
		return "", fmt.Errorf("не удалось извлечь URL MR")
	}

	return mrURL, nil
}

// Создание ветки в удалённом репозитории
func (c *client) CreateBranch(ctx context.Context, sourceBranch, newBranch string, repo Repository) error {
	url := fmt.Sprintf("%s/projects/%s/repository/branches",
		c.apiURL, repo.ProjectID)

	data := map[string]interface{}{
		"branch": newBranch,
		"ref":    sourceBranch, // исходная ветка/коммит
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Не удалось создать ветку %s: %d %s", newBranch, resp.StatusCode, string(body))
	}

	return nil
}

// Удаление ветки в удалённом репозитории
func (c *client) DeleteBranch(ctx context.Context, branch string, repo Repository) error {
	url := fmt.Sprintf("%s/projects/%s/repository/branches/%s",
		c.apiURL, repo.ProjectID, url.PathEscape(branch))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Не удалось удалить ветку %s: %d %s", branch, resp.StatusCode, string(body))
	}

	// log.Print(resp.StatusCode, resp.Body)

	return nil
}

// Сливает указанную ветку (source) в целевую (targetBranch) через GitLab API todo: дублирует createGitlabMR()
func (c *client) MergeBranchInto(ctx context.Context, sourceBranch string, targetBranch string, projectID string) (int, error) {
	mrUrl := fmt.Sprintf(
		"%s/projects/%s/merge_requests", c.apiURL, projectID)

	data := map[string]interface{}{
		"source_branch": sourceBranch,
		"target_branch": targetBranch,
		"title":         fmt.Sprintf("Merge %s into %s", sourceBranch, targetBranch),
		"description":   "Автоматическое слияние веток by Mergenator",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", mrUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API ошибка слияния: %d %s", resp.StatusCode, string(body))
	}

	// Декодируем ответ, чтобы получить mrID
	var mrResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&mrResponse); err != nil {
		return 0, err
	}

	mrID, ok := mrResponse["iid"].(float64)
	if !ok {
		return 0, fmt.Errorf("не удалось извлечь ID MR из ответа")
	}

	return int(mrID), nil
}

func (c *client) AcceptMR(ctx context.Context, mrID int, projectId string) error {
	mrUrl := fmt.Sprintf(
		"%s/projects/%s/merge_requests/%d/merge", c.apiURL, projectId, mrID)

	// Тело запроса (даже если параметры не нужны)
	data := map[string]interface{}{
		"merge_when_pipeline_succeeds": false,
		"should_remove_source_branch":  false,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", mrUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("API ошибка принятия MR: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *client) FindUserID(ctx context.Context, username string) (int, error) {
	url := fmt.Sprintf("%s/users?username=%s", c.apiURL, username)
	c.log.Debug().Msg("connect to gitlab: " + url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GitLab API error: %d", resp.StatusCode)
	}

	var users []GitLabUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return 0, err
	}

	if len(users) == 0 {
		return 0, fmt.Errorf("пользователь %s не найден", username)
	}

	return users[0].ID, nil
}
