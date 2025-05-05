package internal

import (
	"encoding/json"
	"fmt"
	"gitlab-orchestrator-back/internal/models"
	"sort"
	"strings"
)

var (
	stageOrderMap = map[string]int{
		"terraform": 1,
		"ansible":   2,
		"helm":      3,
	}
)

func ProcessJobs(jobMap map[string]map[string]models.Job, steps []models.Step) ([]models.Job, error) {
	var stepID uint
	var jobResult []models.Job

	for stageName, jobs := range jobMap {
		stageOrder := stageOrderMap[stageName]
		// TODO: добавить обработку неизвестных стейджей
		if stageOrder == 0 {
			continue // Пропускаем неизвестные stage
		}
		for _, e := range steps {
			if e.Order == stageOrder {
				stepID = e.ID
			}
		}
		// Получаем все имена джобов для сортировки
		jobNames := make([]string, 0, len(jobs))
		for jobName := range jobs {
			jobNames = append(jobNames, jobName)
		}

		SortNumericalPrefixStrings(jobNames)

		// Создаем записи для каждого джоба
		for jobOrder, jobName := range jobNames {
			job := jobs[jobName]

			job.StepID = stepID // ID соответствующего шага
			job.Order = jobOrder + 1
			job.GitlabJobID = int(job.ID)
			job.ID = 0 // Обнуляем ID, чтобы создать новую запись в базе данных
			jobResult = append(jobResult, job)

		}
	}
	return jobResult, nil
}

func JobsToMap(pipeline []models.Job) map[string]map[string]models.Job {
	jobMap := make(map[string]map[string]models.Job)
	for _, job := range pipeline {
		if jobMap[job.Stage] == nil {
			jobMap[job.Stage] = make(map[string]models.Job)
		}

		jobMap[job.Stage][job.Name] = job
	}
	return jobMap
}

func SortNumericalPrefixStrings(items []string) {
	sort.Slice(items, func(i, j int) bool {
		// Извлекаем числовые префиксы
		numI := extractNumber(items[i])
		numJ := extractNumber(items[j])
		return numI < numJ
	})
}

func extractNumber(s string) int {
	// Удаляем квадратные скобки
	s = strings.Trim(s, "[]")
	// Берём числовую часть до первого дефиса
	parts := strings.Split(s, "-")
	if len(parts) > 0 {
		num := 0
		_, err := fmt.Sscanf(parts[0], "%d", &num)
		if err != nil {
			return 0
		}
		return num
	}
	return 0
}

func PopulateSteps(pipelineID uint) []models.Step {
	var steps []models.Step
	defaultSteps := []struct {
		Name        string
		Description string
		Order       int
		Status      string
	}{
		{Name: "Creating vm", Description: "Initial creation step", Order: 1, Status: "pending"},
		{Name: "Executing automation", Description: "Kubernetes installation", Order: 2, Status: "pending"},
		{Name: "Executing helm", Description: "Running helm", Order: 3, Status: "pending"},
	}

	// Create each step
	for _, stepInfo := range defaultSteps {
		step := models.Step{
			Name:        stepInfo.Name,
			Description: stepInfo.Description,
			Order:       stepInfo.Order,
			PipelineID:  pipelineID,
			Status:      stepInfo.Status,
		}
		steps = append(steps, step)
	}
	return steps
}

func PopulateStand(req struct {
	NameStand string   `json:"nameStand"`
	Products  []string `json:"products"`
	UserID    int64    `json:"userID"`
	Ref       string   `json:"ref"`
}) (models.Stand, error) {
	// Convert products array to JSON
	productsJSON, err := json.Marshal(req.Products)
	if err != nil {
		return models.Stand{}, err
	}

	// Populate and return the Stand structure
	return models.Stand{
		Name:     req.NameStand,
		UserID:   uint(req.UserID),
		Products: productsJSON,
		Status:   "created",
		Ref:      req.Ref,
	}, nil
}
