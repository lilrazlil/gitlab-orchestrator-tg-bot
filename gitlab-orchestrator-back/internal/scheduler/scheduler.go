package scheduler

import (
	"fmt"
	"gitlab-orchestrator-back/internal"
	"gitlab-orchestrator-back/internal/database"
	"gitlab-orchestrator-back/internal/gitlab"
	"gitlab-orchestrator-back/internal/logger"
	"gitlab-orchestrator-back/internal/models"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	StatusPending  = "pending"
	StatusManual   = "manual"
	StatusRunning  = "running"
	StatusError    = "error"
	StatusFailed   = "failed"
	StatusSuccess  = "success"
	StatusCanceled = "canceled"
)

type Runner struct {
	gitlab                gitlab.Gitlab
	db                    *gorm.DB
	workingPending        chan struct{}
	workingCreating       chan struct{}
	jobStatuses           sync.Map
	activeStands          sync.Map // Для отслеживания активных стендов
	maxConcurrentPending  int
	maxConcurrentCreating int
}

func NewRunner(git gitlab.Gitlab, db *gorm.DB) *Runner {
	logger.InfoWithCaller("Создание нового планировщика заданий")
	return &Runner{
		gitlab:                git,
		db:                    db,
		workingPending:        make(chan struct{}, 1),
		workingCreating:       make(chan struct{}, 1),
		maxConcurrentPending:  1,
		maxConcurrentCreating: 1,
	}
}

// StartRunnerScheduler initializes and starts the scheduler with the provided GitLab client
func StartRunnerScheduler(git gitlab.Gitlab) {
	logger.InfoWithCaller("Starting task scheduler")

	runner := NewRunner(git, database.DB)

	logger.InfoWithCaller("Looking for stale stands...")
	if err := runner.recoverStaleStands(); err != nil {
		logger.ErrorfWithCaller("Error recovering stale stands: %v", err)
	}

	pendingTicker := time.NewTicker(10 * time.Second)
	createdTicker := time.NewTicker(15 * time.Second)

	go func() {
		for range pendingTicker.C {
			// Пытаемся отправить значение в канал
			select {
			case runner.workingPending <- struct{}{}: // если канал свободен
				logger.InfoWithCaller("Проверка стендов в статусе ожидания...")
				if err := runner.CheckPendingStands(); err != nil {
					logger.ErrorfWithCaller("Ошибка при проверке стендов: %v", err)
				}
				<-runner.workingPending // освобождаем канал после завершения
			default: // если канал занят
				logger.InfoWithCaller("Предыдущая проверка ещё выполняется, пропускаем")
			}
		}
	}()

	// Обработка created стендов
	go func() {
		for range createdTicker.C {
			select {
			case runner.workingCreating <- struct{}{}:
				logger.InfoWithCaller("Проверка стендов в статусе created...")
				if err := runner.CheckCreatedStands(); err != nil {
					logger.ErrorfWithCaller("Ошибка при проверке created стендов: %v", err)
				}
				<-runner.workingCreating
			default:
				logger.InfoWithCaller("Предыдущая проверка created стендов ещё выполняется, пропускаем")
			}
		}
	}()
}

func (r *Runner) recoverStaleStands() error {
	logger.InfoWithCaller("Начало восстановления зависших стендов")
	var staleStands []models.Stand

	if err := r.db.Where("status = ?", StatusRunning).Find(&staleStands).Error; err != nil {
		return fmt.Errorf("ошибка при поиске зависших стендов: %v", err)
	}

	for _, stand := range staleStands {
		// Проверяем, не находится ли стенд в активной обработке
		if _, active := r.activeStands.Load(stand.Name); active {
			logger.InfofWithCaller("Стенд %s активно обрабатывается, пропускаем", stand.Name)
			continue
		}

		logger.InfofWithCaller("Найден зависший стенд %s (ID: %d), начинаем восстановление", stand.Name, stand.ID)

		tx := r.db.Begin()

		// Обновление статуса стенда
		if err := tx.Exec("UPDATE stands SET status = ? WHERE id = ?",
			StatusPending, stand.ID).Error; err != nil {
			tx.Rollback()
			logger.ErrorfWithCaller("Ошибка при обновлении статуса стенда %s: %v", stand.Name, err)
			continue
		}

		// Обновление статуса пайплайнов
		if err := tx.Exec(`
            UPDATE pipelines 
            SET status = ?, updated_at = NOW() 
            WHERE stand_id = ? AND статус = ?`,
			StatusPending, stand.ID, StatusRunning).Error; err != nil {
			tx.Rollback()
			logger.ErrorfWithCaller("Ошибка при обновлении статусов пайплайнов для стенда %s: %v", stand.Name, err)
			continue
		}

		if err := tx.Exec(`
			UPDATE jobs
			SET status = ?, updated_at = NOW()
			WHERE step_id IN (
				SELECT steps.id
				FROM steps
				JOIN pipelines ON steps.pipeline_id = pipelines.id
				WHERE pipelines.stand_id = ? AND steps.status = ?
			)
			AND status != ?`, StatusManual, stand.ID, StatusRunning, StatusSuccess).Error; err != nil {
			tx.Rollback()
			logger.ErrorfWithCaller("Ошибка при обновлении статусов джоб для стенда %s: %v", stand.Name, err)
			continue
		}

		// Обновление статуса шагов
		if err := tx.Exec(`
            UPDATE steps 
            SET status = ?, updated_at = NOW() 
            WHERE id IN (
                SELECT steps.id 
                FROM steps 
                JOIN pipelines ON steps.pipeline_id = pipelines.id 
                WHERE pipelines.stand_id = ? AND steps.status = ?
            )`, StatusPending, stand.ID, StatusRunning).Error; err != nil {
			tx.Rollback()
			logger.ErrorfWithCaller("Ошибка при обновлении статусов шагов для стенда %s: %v", stand.Name, err)
			continue
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			logger.ErrorfWithCaller("Ошибка при коммите транзакции для стенда %s: %v", stand.Name, err)
			continue
		}

		logger.InfofWithCaller("Стенд %s успешно восстановлен", stand.Name)
	}

	return nil
}

func (r *Runner) CheckCreatedStands() error {
	var stands []models.Stand

	if err := r.db.Where("status = ?", "created").Find(&stands).Error; err != nil {
		return fmt.Errorf("ошибка при получении created стендов: %v", err)
	}

	if len(stands) == 0 {
		logger.InfoWithCaller("Стенды в статусе created не найдены")
		return nil
	}

	wg := sync.WaitGroup{}
	semaphore := make(chan struct{}, r.maxConcurrentCreating)

	for _, stand := range stands {
		if _, active := r.activeStands.Load(stand.Name); active {
			logger.InfofWithCaller("Стенд %s уже обрабатывается, пропускаем", stand.Name)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(stand models.Stand) {
			defer func() {
				<-semaphore
				wg.Done()
				r.activeStands.Delete(stand.Name)
			}()

			r.activeStands.Store(stand.Name, true)
			tx := r.db.Begin()

			if err := r.ProcessCreatingStand(stand); err != nil {
				logger.ErrorfWithCaller("Ошибка при обработке стенда %s: %v", stand.Name, err)
				tx.Rollback()
				return
			}

			if err := tx.Model(&stand).Update("status", StatusPending).Error; err != nil {
				logger.ErrorfWithCaller("Ошибка при обновлении статуса стенда %s: %v", stand.Name, err)
				tx.Rollback()
				return
			}

			tx.Commit()
			logger.InfofWithCaller("Стенд %s успешно обработан и переведен в статус pending", stand.Name)
		}(stand)
	}

	wg.Wait()
	return nil
}

func (r *Runner) CheckPendingStands() error {
	var stands []models.Stand

	if err := r.db.Where("status = ?", StatusPending).Find(&stands).Error; err != nil {
		return err
	}

	if len(stands) == 0 {
		logger.InfoWithCaller("Стенды в статусе ожидания не найдены")
		return nil
	}

	wg := sync.WaitGroup{}
	semaphore := make(chan struct{}, r.maxConcurrentPending)

	for _, stand := range stands {
		if _, active := r.activeStands.Load(stand.Name); active {
			logger.InfofWithCaller("Стенд %s уже обрабатывается, пропускаем", stand.Name)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(stand models.Stand) {
			defer func() {
				<-semaphore // Освобождаем слот
				wg.Done()
				r.activeStands.Delete(stand.Name)
			}()

			r.activeStands.Store(stand.Name, true)
			if err := r.processPendingStand(stand); err != nil {
				logger.ErrorfWithCaller("Ошибка при обработке стенда в ожидании %s: %v", stand.Name, err)
				return
			}
		}(stand)
	}

	wg.Wait()
	return nil
}

func (r *Runner) ProcessCreatingStand(stand models.Stand) error {

	tx := r.db.Begin()

	existBranch, err := r.gitlab.CheckBranchExist(stand.Name)
	if err != nil {
		return err
	}

	if !existBranch {
		err = r.gitlab.CloneBranch(stand.Name, stand.Ref)
		if err != nil {
			return err
		}
		logger.InfofWithCaller("Branch successfully %s created from %s", stand.Name, stand.Ref)
	}
	logger.InfofWithCaller("Branch exist: %v", existBranch)

	existEnv, err := r.gitlab.CheckEnvironmentExist(stand.Name)
	if err != nil {
		return err
	}
	if !existEnv {
		err = r.gitlab.CreateEnvironmentIntoRepository(stand.Name)
		if err != nil {
			return err
		}
		logger.InfofWithCaller("Environment successfully %s created", stand.Name)
	}
	logger.InfofWithCaller("Environment exist: %v", existEnv)
	products, err := database.GetProductsFromStand(stand.Name, tx)
	if err != nil {
		logger.ErrorWithCaller("Failed to get products from stand:", err)
		return err
	}

	existVariables, err := r.gitlab.CheckVariablesIntoEnvironment(stand.Name)
	if err != nil {
		logger.ErrorWithCaller("Failed to check variables into environment:", err)
		return err
	}
	if !existVariables {
		err = r.gitlab.CreateVariablesIntoEnvironment(stand.Name, products)
		if err != nil {
			logger.ErrorWithCaller("Failed to create variables into environment:", err)
			return err
		}
		logger.InfofWithCaller("Environment variables successfully created for %s", stand.Name)
	} else if existVariables {
		err = r.gitlab.UpdateVariablesIntoEnvironment(stand.Name, products)
		if err != nil {
			logger.ErrorWithCaller("Failed to update variables into environment:", err)
			return err
		}
		logger.InfofWithCaller("Environment variables successfully updated for %s", stand.Name)
	}

	err = database.CreateStandPipeline(stand.Name, tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	pipelineID, err := r.gitlab.RunPipeline(stand.Name)
	if err != nil {
		logger.ErrorWithCaller("Failed to run pipeline:", err)
		tx.Rollback()
		return err
	}
	logger.InfofWithCaller("Pipeline successfully created for %s", stand.Name)

	err = database.UpdateStandPipeline(stand.Name, pipelineID, tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	logger.InfofWithCaller("Pipeline successfully updated for %s", stand.Name)

	jobs, err := r.gitlab.GetJobsFromPipeline(pipelineID)
	if err != nil {
		logger.ErrorWithCaller("Error getting jobs from pipeline:", err)
		tx.Rollback()
		return err
	}
	logger.InfofWithCaller("Jobs successfully created for %s", stand.Name)

	JobMap := internal.JobsToMap(jobs)

	steps, err := database.GetStepByStandName(stand.Name, tx)
	if err != nil {
		logger.FatalfWithCaller("Error getting step: %v", err)
		tx.Rollback()
		return err
	}

	jobsProcess, err := internal.ProcessJobs(JobMap, steps)
	if err != nil {
		logger.FatalfWithCaller("Error processing and saving jobs: %v", err)
	}
	logger.InfofWithCaller("Jobs successfully was process for %s", stand.Name)
	err = database.CreateJob(jobsProcess, tx)
	logger.InfofWithCaller("Jobs successfully created for %s", stand.Name)

	tx.Commit()

	return nil
}

func (r *Runner) processPendingStand(stand models.Stand) error {
	var pipelines []models.Pipeline

	if err := r.db.Where("stand_id = ? AND status = ?", stand.ID, StatusPending).Find(&pipelines).Error; err != nil {
		return err
	}

	if len(pipelines) == 0 {
		logger.InfofWithCaller("Для стенда %s нет пайплайнов в статусе pending", stand.Name)
		return nil
	}
	logger.InfofWithCaller("Обработка стенда %s с ID %d", stand.Name, stand.ID)
	if err := database.UpdateStandStatus(StatusRunning, &stand, r.db); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса стенда в БД: %v", err)
	}
	for _, pipeline := range pipelines {
		if err := r.processPendingPipeline(pipeline); err != nil {
			if err := database.UpdateStandStatus(StatusError, &stand, r.db); err != nil {
				return fmt.Errorf("ошибка при обновлении статуса стенда в БД: %v", err)
			}
			logger.ErrorfWithCaller("Ошибка при обработке пайплайна %d: %v", pipeline.ID, err)
			return fmt.Errorf("ошибка при обработке пайплайна %d: %v", pipeline.ID, err)
		}
	}
	if err := database.UpdateStandStatus(StatusSuccess, &stand, r.db); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса стенда в БД: %v", err)
	}
	return nil
}

func (r *Runner) processPendingPipeline(pipeline models.Pipeline) error {
	var steps []models.Step

	if err := r.db.Where("pipeline_id = ? AND status = ?", pipeline.ID, StatusPending).
		Order("\"order\" asc").Find(&steps).Error; err != nil {
		return err
	}

	if len(steps) == 0 {
		logger.InfofWithCaller("Для пайплайна %d нет шагов в статусе pending", pipeline.ID)
		return nil
	}

	if err := database.UpdatePipelineStatus(StatusRunning, &pipeline, r.db); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса шага в БД: %v", err)
	}
	for _, step := range steps {
		if err := r.processStep(step); err != nil {
			if err = database.UpdatePipelineStatus(StatusError, &pipeline, r.db); err != nil {
				return fmt.Errorf("ошибка при обновлении статуса шага в БД: %v", err)
			}
			if err := database.CreateStepNotify(step, StatusError, r.db); err != nil {
				logger.ErrorfWithCaller("Ошибка при создании уведомления для шага %d: %v", step.ID, err)
				return fmt.Errorf("ошибка при создании уведомления для шага %d: %v", step.ID, err)
			}
			logger.ErrorfWithCaller("Ошибка при обработке шага %d: %v", step.ID, err)
			return fmt.Errorf("ошибка при обработке шага %d: %v", step.ID, err)
		}

		logger.InfofWithCaller("Шаг %d для пайплайна %d успешно обработан", step.ID, pipeline.ID)
		if err := database.CreateStepNotify(step, StatusSuccess, r.db); err != nil {
			logger.ErrorfWithCaller("Ошибка при создании уведомления для шага %d: %v", step.ID, err)
			return fmt.Errorf("ошибка при создании уведомления для шага %d: %v", step.ID, err)
		}
	}

	if err := database.UpdatePipelineStatus(StatusSuccess, &pipeline, r.db); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса шага в БД: %v", err)
	}

	logger.InfofWithCaller("Пайплайн %d успешно обработан", pipeline.ID)
	return nil
}

func (r *Runner) processStep(step models.Step) error {
	var jobs []models.Job

	if err := r.db.Where("step_id = ? AND status != ?", step.ID, StatusSuccess).Find(&jobs).Error; err != nil {
		return err
	}

	for _, job := range jobs {
		if job.Status == StatusFailed || job.Status == StatusCanceled {
			// Обновляем статус шага на failed
			if err := database.UpdateStepStatus(StatusError, &step, r.db); err != nil {
				logger.ErrorfWithCaller("Ошибка при обновлении статуса шага в БД: %v", err)
				return fmt.Errorf("ошибка при обновлении статуса шага в БД: %v", err)
			}
			logger.InfofWithCaller("Шаг %d завершился с ошибкой из-за джобы %d со статусом %s", step.ID, job.ID, job.Status)
			return fmt.Errorf("step failed due to job %d with status %s", job.ID, job.Status)
		}
	}

	hasManualJobs := false
	hasRunningJobs := false
	if err := database.UpdateStepStatus(StatusRunning, &step, r.db); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса шага в БД: %v", err)
	}
	for _, job := range jobs {
		switch job.Status {
		case StatusManual:
			hasManualJobs = true
		case StatusRunning:
			hasRunningJobs = true
		}
	}

	if hasManualJobs && !hasRunningJobs {
		for _, job := range jobs {
			if job.Status == StatusManual {
				if err := r.processJob(job); err != nil {
					if err = database.UpdateStepStatus(StatusError, &step, r.db); err != nil {
						return fmt.Errorf("ошибка при обновлении статуса шага в БД: %v", err)
					}
					logger.ErrorfWithCaller("Ошибка при обработке джобы %d: %v", job.ID, err)
					return fmt.Errorf("ошибка при обработке джобы %d: %v", job.ID, err)
				}
			}
		}
	}
	if err := database.UpdateStepStatus(StatusSuccess, &step, r.db); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса шага в БД: %v", err)
	}

	return nil
}

func (r *Runner) processJob(job models.Job) error {
	logger.InfofWithCaller("Запуск джобы %d (GitLab JobID: %d)", job.ID, job.GitlabJobID)

	// Запускаем джобу в GitLab
	if job.StartedAt == nil {
		if err := r.gitlab.RunJob(job.GitlabJobID); err != nil {
			return fmt.Errorf("ошибка при запуске джобы %d: %v", job.GitlabJobID, err)
		}
		// Обновляем время запуска джобы в БД
		if err := r.db.Model(&job).Update("started_at", time.Now()).Error; err != nil {
			return fmt.Errorf("ошибка при обновлении времени запуска джобы в БД: %v", err)
		}
	} else {
		logger.InfofWithCaller("Джоба %d ранее была запущена, просматриваем статус", job.ID)
	}

	if err := r.monitorJobStatus(job); err != nil {
		return err
	}

	if err := r.db.Model(&job).Update("finished_at", time.Now()).Error; err != nil {
		return fmt.Errorf("ошибка при обновлении времени запуска джобы в БД: %v", err)
	}

	return nil
}

func (r *Runner) monitorJobStatus(job models.Job) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	jobKey := fmt.Sprintf("job_%d", job.GitlabJobID)

	// Первоначальная проверка статуса
	status, err := r.checkAndUpdateStatus(job, jobKey)
	if err != nil {
		return err
	}
	if status {
		return nil
	}

	// Последующие проверки по тикеру
	for range ticker.C {
		status, err = r.checkAndUpdateStatus(job, jobKey)
		if err != nil {
			ticker.Stop()
			return err
		}
		if status {
			return nil
		}

	}
	return nil
}

func (r *Runner) checkAndUpdateStatus(job models.Job, jobKey string) (finished bool, err error) {
	status, err := r.gitlab.GetJobStatus(job.GitlabJobID)
	if err != nil {
		logger.ErrorfWithCaller("Ошибка при получении статуса джобы %d: %v", job.GitlabJobID, err)
		r.jobStatuses.Delete(jobKey)
		return true, err
	}

	if cachedStatus, exists := r.jobStatuses.Load(jobKey); exists && cachedStatus.(string) == status {
		return false, nil
	}

	if err = database.UpdateJobStatus(status, &job, r.db); err != nil {
		logger.ErrorfWithCaller("Ошибка при обновлении статуса джобы в БД: %v", err)
		return false, err
	}

	r.jobStatuses.Store(jobKey, status)

	if status == StatusFailed || status == StatusCanceled {
		r.jobStatuses.Delete(jobKey)
		logger.ErrorfWithCaller("Джоба %d завершилась с ошибкой со статусом %s", job.ID, status)
		return true, fmt.Errorf("stand have a failed job")
	}

	if status == StatusSuccess {
		r.jobStatuses.Delete(jobKey)
		return true, nil
	}

	return false, nil
}
