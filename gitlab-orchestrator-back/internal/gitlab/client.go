package gitlab

import (
	"encoding/json"
	"fmt"
	"gitlab-orchestrator-back/internal/config"
	"gitlab-orchestrator-back/internal/logger"
	"gitlab-orchestrator-back/internal/models"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second, // Set a timeout if needed
}

type Client struct {
	ProjectID            int    `json:"project_id"`
	BaseUrl              string `json:"base_url"`
	PrivateToken         string `json:"private_token"`
	TriggerPipelineToken string `json:"trigger_pipeline_token"`
}

// NewClient creates a new GitLab client using the application configuration
func NewClient() *Client {
	return &Client{
		ProjectID:            config.Config.GitlabProjectID,
		BaseUrl:              config.Config.GitlabAPIURL,
		PrivateToken:         config.Config.GitlabToken,
		TriggerPipelineToken: config.Config.GitlabTriggerPipelineToken,
	}
}

type Gitlab interface {
	CloneBranch(branchName string, refBranch string) error
	RunPipeline(branchName string) (int, error)
	RunJob(jobID int) error
	GetJobStatus(jobID int) (string, error)
	GetJobsFromPipeline(pipelineID int) ([]models.Job, error)
	CheckBranchExist(branchName string) (bool, error)
	CheckEnvironmentExist(branchName string) (bool, error)
	CheckVariablesIntoEnvironment(branchName string) (bool, error)
	CreateEnvironmentIntoRepository(branchName string) error
	CreateVariablesIntoEnvironment(branchName string, variables []string) error
	UpdateVariablesIntoEnvironment(branchName string, variables []string) error
}

// RunJob запускает конкретную джобу в GitLab
func (c *Client) RunJob(jobID int) error {
	logger.InfofWithCaller("Запуск джобы GitLab с ID: %d", jobID)
	url := fmt.Sprintf("%s/projects/%d/jobs/%d/play", c.BaseUrl, c.ProjectID, jobID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса на запуск джобы %d: %v", jobID, err)
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса на запуск джобы %d: %v", jobID, err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logger.ErrorfWithCaller("Ответ GitLab с кодом %d при запуске джобы %d: %s", resp.StatusCode, jobID, string(body))
		return fmt.Errorf("ошибка при запуске джобы: %s, тело: %s", resp.Status, string(body))
	}

	logger.InfofWithCaller("Джоба %d успешно запущена", jobID)
	return nil
}

// GetJobStatus получает текущий статус джобы из GitLab
func (c *Client) GetJobStatus(jobID int) (string, error) {
	logger.DebugfWithCaller("Получение статуса джобы %d", jobID)
	url := fmt.Sprintf("%s/projects/%d/jobs/%d", c.BaseUrl, c.ProjectID, jobID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса на получение статуса джобы %d: %v", jobID, err)
		return "", err
	}

	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса на получение статуса джобы %d: %v", jobID, err)
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		logger.ErrorfWithCaller("Ответ GitLab с кодом %d при получении статуса джобы %d", resp.StatusCode, jobID)
		return "", fmt.Errorf("ошибка при получении статуса джобы: %s", resp.Status)
	}

	var jobInfo struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jobInfo); err != nil {
		logger.ErrorfWithCaller("Ошибка при декодировании ответа для джобы %d: %v", jobID, err)
		return "", fmt.Errorf("ошибка при декодировании ответа: %v", err)
	}

	logger.DebugfWithCaller("Джоба %d имеет статус: %s", jobID, jobInfo.Status)
	return jobInfo.Status, nil
}
func (c *Client) GetJobsFromPipeline(pipelineID int) ([]models.Job, error) {
	logger.InfofWithCaller("Получение списка джоб для пайплайна %d", pipelineID)

	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/pipelines/" + strconv.Itoa(pipelineID) + "/jobs"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	var jobs []models.Job

	if err = json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		logger.ErrorfWithCaller("Ошибка при декодировании ответа: %v", err)
		return nil, err
	}

	logger.InfofWithCaller("Получено %d джоб для пайплайна %d", len(jobs), pipelineID)
	return jobs, nil
}

func (c *Client) CheckVariablesIntoEnvironment(branchName string) (bool, error) {
	logger.InfofWithCaller("Проверка переменных в окружении %s", branchName)
	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/variables/PRODUCTS?filter[environment_scope]=" + branchName
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return false, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		logger.InfofWithCaller("Переменные найдены для окружения %s", branchName)
		return true, nil
	case http.StatusNotFound:
		logger.InfofWithCaller("Переменные не найдены для окружения %s", branchName)
		return false, nil
	}

	if resp.StatusCode >= 400 {
		logger.ErrorfWithCaller("Ошибка при получении переменных: %s", resp.Status)
		return false, fmt.Errorf("failed to fetch variables: %s", resp.Status)
	}

	return false, nil
}

func (c *Client) UpdateVariablesIntoEnvironment(branchName string, variables []string) error {
	logger.InfofWithCaller("Обновление переменных в окружении %s", branchName)
	data := fmt.Sprintf("value=%s", strings.Join(variables, ","))
	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/variables/PRODUCTS?filter[environment_scope]=" + branchName
	req, err := http.NewRequest("PUT", url, strings.NewReader(data))
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return err
	}

	if resp.StatusCode == http.StatusCreated {
		logger.InfofWithCaller("Переменные успешно обновлены для окружения %s", branchName)
		return nil
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		logger.ErrorfWithCaller("Ошибка при обновлении переменных: %s", resp.Status)
		return fmt.Errorf("failed to update vars: %s", resp.Status)
	}

	return nil
}

func (c *Client) CreateVariablesIntoEnvironment(branchName string, variables []string) error {
	logger.InfofWithCaller("Создание переменных в окружении %s", branchName)
	data := fmt.Sprintf("key=PRODUCTS&value=%s&environment_scope=%s", variables, branchName)
	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/variables"
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return err
	}

	if resp.StatusCode == http.StatusCreated {
		logger.InfofWithCaller("Переменные успешно созданы для окружения %s", branchName)
		return nil
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		logger.ErrorfWithCaller("Ошибка при создании переменных: %s", resp.Status)
		return fmt.Errorf("failed to create vars: %s", resp.Status)
	}

	return nil
}

func (c *Client) CheckEnvironmentExist(branchName string) (bool, error) {
	logger.InfofWithCaller("Проверка существования окружения %s", branchName)
	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/environments?search=" + branchName
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return false, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return false, err
	}

	var jsonBody []map[string]interface{}

	if err = json.NewDecoder(resp.Body).Decode(&jsonBody); err != nil {
		logger.ErrorfWithCaller("Ошибка при декодировании ответа: %v", err)
		return false, err
	}

	if len(jsonBody) == 0 {
		logger.InfofWithCaller("Окружение %s не существует", branchName)
		return false, nil
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		logger.ErrorfWithCaller("Ошибка при проверке окружения: %s", resp.Status)
		return false, fmt.Errorf("failed to create environment: %s", resp.Status)
	}

	logger.InfofWithCaller("Окружение %s существует", branchName)
	return true, nil
}

func (c *Client) CreateEnvironmentIntoRepository(branchName string) error {
	logger.InfofWithCaller("Создание окружения %s в репозитории", branchName)
	data := fmt.Sprintf("name=%s", branchName)
	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/environments"
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return err
	}

	if resp.StatusCode == http.StatusCreated {
		logger.InfofWithCaller("Окружение %s успешно создано", branchName)
		return nil
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		logger.ErrorfWithCaller("Ошибка при создании окружения: %s", resp.Status)
		return fmt.Errorf("failed to create environment: %s", resp.Status)
	}

	return nil
}

func (c *Client) RunPipeline(branchName string) (int, error) {
	logger.InfofWithCaller("Запуск пайплайна для ветки %s", branchName)
	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/trigger/pipeline?ref=" + branchName + "&token=" + c.TriggerPipelineToken

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return 0, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	var pipelineResponse map[string]interface{}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при чтении тела ответа: %v", err)
		return 0, err
	}

	err = json.Unmarshal(body, &pipelineResponse)

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logger.ErrorfWithCaller("Ошибка при запуске пайплайна: %s Тело: %s", resp.Status, string(body))
		return 0, fmt.Errorf("failed to run pipeline: %s Body: %s", resp.Status, string(body))
	}

	logger.InfofWithCaller("Пайплайн для ветки %s успешно запущен", branchName)
	return int(pipelineResponse["id"].(float64)), nil
}

func (c *Client) CheckBranchExist(branchName string) (bool, error) {
	logger.InfofWithCaller("Проверка существования ветки %s", branchName)
	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/repository/branches/" + branchName
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return false, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return false, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		logger.InfofWithCaller("Ветка %s существует", branchName)
		return true, nil
	case http.StatusNotFound:
		logger.InfofWithCaller("Ветка %s не найдена", branchName)
		return false, nil
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		logger.ErrorfWithCaller("Ошибка при проверке ветки: %s", resp.Status)
		return false, fmt.Errorf("failed to check branch: %s", resp.Status)
	}

	return false, err
}

func (c *Client) CloneBranch(branchName string, refBranch string) error {
	logger.InfofWithCaller("Клонирование ветки %s из %s", branchName, refBranch)
	url := c.BaseUrl + "/projects/" + strconv.Itoa(c.ProjectID) + "/repository/branches"
	req, err := http.NewRequest("POST", url+"?branch="+branchName+"&ref="+refBranch, nil)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при создании запроса: %v", err)
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PrivateToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при отправке запроса: %v", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.ErrorWithCaller("Не удалось закрыть тело ответа")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logger.ErrorfWithCaller("Ошибка при клонировании ветки: %s Тело: %s", resp.Status, string(body))
		return fmt.Errorf("failed to clone branch: %s Body:%s", resp.Status, string(body))
	}

	logger.InfofWithCaller("Ветка %s успешно клонирована из %s", branchName, refBranch)
	return nil
}
