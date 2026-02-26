package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"mergenator/internal/config"
)

type Repository struct {
	StandBranch string
	ProjectId   string
	AssigneeId  int
}

// Проверка существования ветки в репозитории через API
func BranchExistsInRepo(branch string, projectId string) (bool, error) {
	escapedBranch := url.PathEscape(branch)
	branchesUrl := fmt.Sprintf(
		"%s/projects/%s/repository/branches/%s",
		config.GitlabApiUrl, projectId, escapedBranch)

	req, err := http.NewRequest("GET", branchesUrl, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Private-Token", config.GitlabAccessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf(
			"API ошибка: %d %s", resp.StatusCode, string(body))
	}
}

func HasOpenMR(source, target string, projectID string) (bool, int, string, error) {
	mrUrl := fmt.Sprintf(
		"%s/projects/%s/merge_requests?source_branch=%s&target_branch=%s&state=opened",
		config.GitlabApiUrl, projectID, source, target)

	req, err := http.NewRequest("GET", mrUrl, nil)
	if err != nil {
		return false, 0, "", err
	}

	req.Header.Set("Private-Token", config.GitlabAccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Преобразуем в строку
	bodyString := string(body)
	// Выводим в лог
	log.Printf("Ответ сервера (JSON):\n%s", bodyString)

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
func CreateGitLabMR(sourceBranch, title string, repo Repository) (string, error) {
	mrUrl := fmt.Sprintf("%s/projects/%s/merge_requests", config.GitlabApiUrl, repo.ProjectId)

	data := map[string]interface{}{
		"source_branch":        sourceBranch,
		"target_branch":        repo.StandBranch,
		"title":                title,
		"assignee_ids":         []int{repo.AssigneeId},
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
	req.Header.Set("Private-Token", config.GitlabAccessToken)

	resp, err := httpClient.Do(req)
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
func CreateRemoteBranch(sourceBranch, newBranch string, repo Repository) error {
	url := fmt.Sprintf("%s/projects/%s/repository/branches",
		config.GitlabApiUrl, repo.ProjectId)

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
	req.Header.Set("Private-Token", config.GitlabAccessToken)

	resp, err := httpClient.Do(req)
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
func DeleteRemoteBranch(branch string, repo Repository) error {
	url := fmt.Sprintf("%s/projects/%s/repository/branches/%s",
		config.GitlabApiUrl, repo.ProjectId, url.PathEscape(branch))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Private-Token", config.GitlabAccessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Не удалось удалить ветку %s: %d %s", branch, resp.StatusCode, string(body))
	}

	log.Print(resp.StatusCode, resp.Body)

	return nil
}

// Сливает указанную ветку (source) в целевую (targetBranch) через GitLab API todo: дублирует createGitlabMR()
func MergeBranchInto(source, targetBranch string, projectID string) (int, error) {
	mrUrl := fmt.Sprintf(
		"%s/projects/%s/merge_requests", config.GitlabApiUrl, projectID)

	data := map[string]interface{}{
		"source_branch": source,
		"target_branch": targetBranch,
		"title":         fmt.Sprintf("Merge %s into %s", source, targetBranch),
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
	req.Header.Set("Private-Token", config.GitlabAccessToken)

	resp, err := httpClient.Do(req)
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

func AcceptMergeRequest(mrID int, projectId string) error {
	mrUrl := fmt.Sprintf(
		"%s/projects/%s/merge_requests/%d/merge", config.GitlabApiUrl, projectId, mrID)

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
	req.Header.Set("Private-Token", config.GitlabAccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
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

type GitLabUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func FindGitLabUserID(username string) (int, error) {
	url := fmt.Sprintf("%s/users?username=%s", config.GitlabApiUrl, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Private-Token", config.GitlabAccessToken)

	resp, err := httpClient.Do(req)
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
