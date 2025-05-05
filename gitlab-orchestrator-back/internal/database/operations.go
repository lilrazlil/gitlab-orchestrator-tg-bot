package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitlab-orchestrator-back/internal"
	"gitlab-orchestrator-back/internal/logger"
	"gitlab-orchestrator-back/internal/models"
	"time"

	"gorm.io/gorm"
)

// GetNotifications получает неотправленные уведомления
func GetNotifications() ([]models.StepState, error) {
	logger.InfoWithCaller("Получение списка неотправленных уведомлений")
	var notifications []models.StepState

	result := DB.Where("send = false").
		Order("created_at asc").
		Find(&notifications)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logger.InfoWithCaller("Неотправленные уведомления не найдены")
			return []models.StepState{}, errors.New("уведомления не найдены")
		}
		logger.ErrorfWithCaller("Ошибка при получении уведомлений: %v", result.Error)
		return []models.StepState{}, fmt.Errorf("ошибка при получении уведомления: %v", result.Error)
	}

	logger.InfofWithCaller("Найдено %d неотправленных уведомлений", len(notifications))
	return notifications, nil
}

// MarkNotificationAsSent помечает уведомление как отправленное
func MarkNotificationAsSent(notificationID uint) error {
	logger.InfofWithCaller("Отметка уведомления %d как отправленного", notificationID)
	result := DB.Model(&models.StepState{}).
		Where("id = ?", notificationID).
		Update("send", true)

	if result.Error != nil {
		logger.ErrorfWithCaller("Ошибка при обновлении статуса уведомления %d: %v", notificationID, result.Error)
		return fmt.Errorf("ошибка при обновлении статуса уведомления: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		logger.WarnfWithCaller("Уведомление %d не найдено", notificationID)
		return errors.New("уведомление не найдено")
	}

	logger.InfofWithCaller("Уведомление %d успешно отмечено как отправленное", notificationID)
	return nil
}

// CreateStepNotify создает уведомление о статусе шага
func CreateStepNotify(step models.Step, status string, tx *gorm.DB) error {
	logger.InfofWithCaller("Создание уведомления для шага %d со статусом %s", step.ID, status)

	// Получаем шаг со связанными данными
	if err := tx.Preload("Pipeline.Stand").First(&step).Error; err != nil {
		logger.ErrorfWithCaller("Ошибка при получении шага %d: %v", step.ID, err)
		return fmt.Errorf("ошибка при получении шага: %v", err)
	}

	// Создаем новое состояние шага
	stepState := models.StepState{
		StandName: step.Pipeline.Stand.Name,
		StepName:  step.Name,
		UserID:    step.Pipeline.Stand.UserID,
		Status:    status,
		Order:     step.Order,
	}

	// Создаем новую запись
	if err := tx.Create(&stepState).Error; err != nil {
		logger.ErrorfWithCaller("Ошибка при создании уведомления для шага %d: %v", step.ID, err)
		return fmt.Errorf("ошибка при создании состояния шага: %v", err)
	}

	logger.InfofWithCaller("Успешно создано уведомление для шага %d (стенд: %s, статус: %s)",
		step.ID, step.Pipeline.Stand.Name, status)
	return nil
}

func UpdateJobStatus(status string, job *models.Job, tx *gorm.DB) error {
	if err := tx.Model(&job).Update("status", status).Error; err != nil {
		return fmt.Errorf("ошибка при обновлении статуса пайплайна в БД: %v", err)
	}
	return nil
}

// UpdatePipelineStatus updates the status of a pipeline in the database
func UpdatePipelineStatus(status string, pipeline *models.Pipeline, tx *gorm.DB) error {
	if err := tx.Model(&pipeline).Update("status", status).Error; err != nil {
		return fmt.Errorf("ошибка при обновлении статуса пайплайна в БД: %v", err)
	}
	return nil
}

func UpdateStepStatus(status string, step *models.Step, tx *gorm.DB) error {
	if err := tx.Model(&step).Update("status", status).Error; err != nil {
		return fmt.Errorf("ошибка при обновлении статуса шага в БД: %v", err)
	}
	return nil
}

func UpdateStandStatus(status string, stand *models.Stand, tx *gorm.DB) error {
	if err := tx.Model(&stand).Update("status", status).Error; err != nil {
		return fmt.Errorf("ошибка при обновлении статуса стенда в БД: %v", err)
	}
	return nil
}

// CreateStand creates a new stand in the database and returns the created stand with its ID
func CreateStand(stand models.Stand, tx *gorm.DB) error {
	// Check if stand with the same name already exists
	var existingStand models.Stand
	if err := tx.Where("name = ?", stand.Name).First(&existingStand).Error; err == nil {
		return errors.New("stand with this name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Save the stand to the database
	if err := tx.Create(&stand).Error; err != nil {
		return err
	}

	// GORM уже обновил объект stand и заполнил ID
	return nil
}

func UpdateStandPipeline(standName string, pipelineID int, tx *gorm.DB) error {
	var pipeline models.Pipeline
	if err := tx.Where("name = ?", standName).First(&pipeline).Error; err != nil {
		return fmt.Errorf("failed to find pipeline: %v", err)
	}

	pipeline.GitlabPipelineID = pipelineID
	if err := tx.Save(&pipeline).Error; err != nil {
		return fmt.Errorf("failed to update pipeline: %v", err)
	}

	// Создаем базовые шаги для пайплайна
	steps := internal.PopulateSteps(pipeline.ID)

	// Сохраняем шаги в БД
	for _, step := range steps {
		if err := CreateStep(step, tx); err != nil {
			return fmt.Errorf("failed to create step: %v", err)
		}
	}

	return nil
}

func CreateStandPipeline(standName string, tx *gorm.DB) error {
	// Получаем стенд по имени
	stand, err := GetStandByName(standName, tx)
	if err != nil {
		return fmt.Errorf("failed to get stand: %v", err)
	}

	// Создаем новый пайплайн
	pipeline := models.Pipeline{
		Name:      stand.Name,
		StandID:   stand.ID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Сохраняем пайплайн в БД
	_, err = CreatePipeline(pipeline, tx)
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %v", err)
	}

	return nil
}

func GetAllSubos() ([]models.Subos, error) {
	var subos []models.Subos
	if err := DB.Find(&subos).Error; err != nil {
		return nil, err
	}
	return subos, nil
}

// CreateUser creates a new user in the database
func CreateUser(user *models.User) error {
	result := DB.Create(user)
	return result.Error
}

func GetProductsFromStand(stand string, tx *gorm.DB) ([]string, error) {
	var standInfo models.Stand
	result := tx.Where("name = ?", stand).First(&standInfo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("stand not found")
		}
		return nil, result.Error
	}

	var products []string
	if err := json.Unmarshal(standInfo.Products, &products); err != nil {
		return nil, err
	}
	return products, nil
}

// GetUserByID retrieves a user by ID from the database
func GetUserByID(id uint) (*models.User, error) {
	var user models.User
	result := DB.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}
	return &user, nil
}

// GetAllUsers retrieves all users from the database
func GetAllUsers() ([]models.User, error) {
	var users []models.User
	result := DB.Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	return users, nil
}

// UpdateUser updates an existing user in the database
func UpdateUser(user *models.User) error {
	result := DB.Save(user)
	return result.Error
}

// DeleteUser deletes a user from the database
func DeleteUser(id uint) error {
	result := DB.Unscoped().Delete(&models.User{}, id) // Добавлено Unscoped() для жесткого удаления
	return result.Error
}

// GetAllStands retrieves all stands from the database
func GetAllStands() ([]models.Stand, error) {
	var stands []models.Stand
	result := DB.Find(&stands)
	if result.Error != nil {
		return nil, result.Error
	}
	return stands, nil
}

// GetStandByName retrieves a stand by name from the database
func GetStandByName(name string, tx *gorm.DB) (*models.Stand, error) {
	var stand models.Stand
	result := tx.Where("name = ?", name).First(&stand)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("stand not found")
		}
		return nil, result.Error
	}
	return &stand, nil
}

// GetPendingStandByName retrieves a stand by name with a pending status
func GetPendingStandByName(name string) (*models.Stand, error) {
	var stand models.Stand
	result := DB.Where("name = ? AND status = ?", name, "pending").First(&stand)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("pending stand not found")
		}
		return nil, result.Error
	}
	return &stand, nil
}

func GetStatusStandByName(name string, tx *gorm.DB) (string, error) {
	var stand models.Stand
	result := tx.Where("name = ?", name).First(&stand)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", errors.New("stand not found")
		}
		return "nil", result.Error
	}
	return stand.Status, nil
}

// CreateStep creates a new step in the database
func CreateStep(step models.Step, tx *gorm.DB) error {
	result := tx.Create(&step)
	return result.Error
}

// CreateJob creates a new job in the database
func CreateJob(job []models.Job, tx *gorm.DB) error {
	result := tx.Create(&job)
	return result.Error
}

// GetJobByID retrieves a job by ID from the database
func GetJobByID(id uint) (*models.Job, error) {
	var job models.Job
	result := DB.First(&job, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("job not found")
		}
		return nil, result.Error
	}
	return &job, nil
}

// GetJobsByPipelineID retrieves all jobs for a pipeline from the database
func GetJobsByPipelineID(pipelineID uint) ([]models.Job, error) {
	var jobs []models.Job
	result := DB.Where("pipeline_id = ?", pipelineID).Order("order asc").Find(&jobs)
	if result.Error != nil {
		return nil, result.Error
	}
	return jobs, nil
}

// GetStandWithSteps retrieves a stand with its steps from the database
func GetStandWithSteps(standID uint) (*models.Stand, error) {
	var stand models.Stand
	result := DB.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("steps.order ASC")
	}).First(&stand, standID)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("stand not found")
		}
		return nil, result.Error
	}
	return &stand, nil
}

// CreatePipeline creates a new pipeline in the database and returns the created pipeline with its ID
func CreatePipeline(pipeline models.Pipeline, tx *gorm.DB) (models.Pipeline, error) {
	if err := tx.Create(&pipeline).Error; err != nil {
		return models.Pipeline{}, err
	}
	return pipeline, nil
}

func GetStepByStandName(standName string, tx *gorm.DB) ([]models.Step, error) {
	var step []models.Step
	result := tx.Joins("JOIN pipelines ON steps.pipeline_id = pipelines.id").
		Joins("JOIN stands ON pipelines.stand_id = stands.id").
		Where("stands.name = ?", standName).Find(&step)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("step not found for stand %s", standName)
		}
		return nil, result.Error
	}
	return step, nil
}
