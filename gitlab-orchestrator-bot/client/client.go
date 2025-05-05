package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gitlab-orchestrator-bot/config"
	"io"
	"net/http"
	"sync"
	"time"
)

var httpClient = &http.Client{
	Timeout: time.Second * 20,
}

type Notifications struct {
	ID        uint   `json:"id"`
	UserID    int64  `json:"user_id"`
	StandName string `json:"stand_name"`
	StepName  string `json:"step_name"`
	Order     int    `json:"order"`
	Status    string `json:"status"`
}

// Cache for Subos data
var (
	subosCache      map[string]string
	subosCacheTime  time.Time
	subosCacheMutex sync.RWMutex
)

func SendMarkNotification(notificationID uint) error {
	jsonData := []byte(fmt.Sprintf(`{"notificationID": %d}`, notificationID))

	resp, err := httpClient.Post(fmt.Sprintf("%s/notify", config.Config.BackendURL),
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to patch notification: %s, body: %s", resp.Status, string(bodyBytes))
	}

	return nil
}

func FetchNotifications() ([]Notifications, error) {
	var response []Notifications

	resp, err := httpClient.Get(fmt.Sprintf("%s/notify", config.Config.BackendURL))
	if err != nil {
		return []Notifications{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusNoContent {
		return []Notifications{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return []Notifications{}, fmt.Errorf("failed to fetch all subos, status: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return []Notifications{}, err
	}
	return response, nil

}

func init() {
	// Start a goroutine to clear the cache every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			clearSubosCache()
		}
	}()
}

func SendPatchRequest(url string, body interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonBody))

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// clearSubosCache clears the Subos cache
func clearSubosCache() {
	subosCacheMutex.Lock()
	defer subosCacheMutex.Unlock()
	subosCache = nil
	subosCacheTime = time.Time{}
}

// FetchSubos получает список продуктов от бэкенда
func FetchSubos() (map[string]string, error) {
	subosCacheMutex.RLock()
	if len(subosCache) > 0 && time.Since(subosCacheTime) < 5*time.Minute {
		cachedSubos := subosCache
		subosCacheMutex.RUnlock()
		return cachedSubos, nil
	}
	subosCacheMutex.RUnlock()

	resp, err := httpClient.Get(fmt.Sprintf("%s/subos", config.Config.BackendURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch subos, status: %s", resp.Status)
	}

	var subos map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&subos); err != nil {
		return nil, err
	}
	// Update the cache with new data
	subosCacheMutex.Lock()
	subosCache = subos
	subosCacheTime = time.Now()
	subosCacheMutex.Unlock()
	return subosCache, nil
}

// FetchAllSubos получает полный список объектов продуктов от бэкенда
func FetchAllSubos() (map[string]string, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/subos", config.Config.BackendURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch all subos, status: %s", resp.Status)
	}
	var subos map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&subos); err != nil {
		return nil, err
	}
	return subos, nil
}

// CreateStand отправляет запрос на создание стенда
func SendStandToBackend(jsonData []byte) (string, error) {
	resp, err := httpClient.Post(fmt.Sprintf("%s/stands", config.Config.BackendURL),
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create stand: %s, body: %s", resp.Status, string(bodyBytes))
	}

	var response map[string]interface{}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", err
	}

	return response["message"].(string), nil
}

// FetchAllStands получает список стендов
func FetchAllStands() ([]map[string]interface{}, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/stands", config.Config.BackendURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch stands, status: %s", resp.Status)
	}

	var stands []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stands); err != nil {
		return nil, err
	}
	return stands, nil
}

// GetStandDeployments получает версии продуктов на стенде
func GetStandDeployments(standName string) (map[string]string, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/stands/%s/deployments", config.Config.BackendURL, standName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch stand deployments, status: %s", resp.Status)
	}

	var deployments map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
		return nil, err
	}
	return deployments, nil
}

// GetUsers получает список пользователей с их ролями
func GetUsers() ([]map[string]any, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/users", config.Config.BackendURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch users, status: %s", resp.Status)
	}

	var users []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	return users, nil
}
