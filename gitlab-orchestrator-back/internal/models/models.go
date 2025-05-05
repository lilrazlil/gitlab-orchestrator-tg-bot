package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

// TableName Explicitly specify the table name
func (User) TableName() string {
	return "users" // Ensure this matches the actual table name in the database
}

type User struct {
	ID     int64   `gorm:"primaryKey;uniqueIndex;not null" json:"chat_id"`
	Name   string  `gorm:"not null" json:"name"`
	Role   string  `gorm:"not null" json:"role"`
	Stands []Stand `gorm:"foreignKey:UserID"` // One user can have many stands
}
type Subos struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type Stand struct {
	ID                uint           `gorm:"primaryKey"`
	Name              string         `gorm:"not null;uniqueIndex"`
	UserID            uint           `gorm:"index;not null"`     // Внешний ключ к пользователю
	User              User           `gorm:"foreignKey:UserID"`  // Стенд принадлежит пользователю
	Products          datatypes.JSON `gorm:"type:json"`          // Продукты хранятся как JSON
	Pipelines         []Pipeline     `gorm:"foreignKey:StandID"` // Один стенд может иметь много шагов
	CurrentPipelineID uint           `gorm:"index"`              // ID текущего пайплайна
	Status            string         `gorm:"not null"`
	Ref               string         `gorm:"not null"` // бренча в GitLab
	CreatedAt         time.Time      `gorm:"index"`
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}
type Pipeline struct {
	ID               uint      `gorm:"primaryKey"`
	Name             string    `gorm:"not null"`
	StandID          uint      `gorm:"index;not null"`             // Внешний ключ к шагу
	Stand            Stand     `gorm:"foreignKey:StandID"`         // Пайплайн принадлежит шагу
	Status           string    `gorm:"not null;default:'pending'"` // Статус выполнения пайплайна
	Steps            []Step    `gorm:"foreignKey:PipelineID"`      // Один пайплайн может иметь много джобов
	GitlabPipelineID int       `gorm:"index"`                      // ID пайплайна в GitLab
	CreatedAt        time.Time `gorm:"index"`
	UpdatedAt        time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
	DeletedAt        gorm.DeletedAt
}

// Step представляет шаг выполнения в стенде
type Step struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string
	Order       int      `gorm:"not null;default:0;index"` // Порядок выполнения шага
	PipelineID  uint     `gorm:"index;not null"`           // Внешний ключ к стенду
	Pipeline    Pipeline `gorm:"foreignKey:PipelineID"`    // Шаг принадлежит пайплайну
	Jobs        []Job    `gorm:"foreignKey:StepID"`        // Один шаг может запускать много Джоб
	Status      string   `gorm:"not null"`                 // Статус выполнения шага
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt
}

// Job представляет задачу, выполняемую в шаге
type Job struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null" json:"name"`
	Description string
	StepID      uint      `gorm:"index;not null"`    // Внешний ключ к пайплайну
	Step        Step      `gorm:"foreignKey:StepID"` // Джоб принадлежит пайплайну
	GitlabJobID int       `gorm:"index"`             // ID джоба в GitLab
	Stage       string    `json:"stage"`
	Status      string    `gorm:"not null" json:"status"` // Статус выполнения джоба
	Order       int       `gorm:"not null;default:0"`     // Порядок выполнения джоба
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	DeletedAt   gorm.DeletedAt
}

type StepState struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	StandName string         `json:"stand_name" gorm:"type:varchar(255);not null"`
	StepName  string         `json:"step_name" gorm:"type:varchar(255);not null"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Status    string         `json:"status" gorm:"type:varchar(50)"`
	Send      bool           `json:"send" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	Order     int            `json:"order" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Связь с пользователем
	User User `json:"-" gorm:"foreignKey:UserID"`
}
