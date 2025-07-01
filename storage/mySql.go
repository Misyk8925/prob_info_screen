package storage

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"strings"
	"time"
)

// Task представляет задачу в нашей базе данных
type Task struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// Уникальный ключ из Jira (например, 'PROJ-123')
	// gorm:"unique" создает уникальный индекс для этого поля.
	ExternalID string `gorm:"unique;not null"`
	ProjectKey string `gorm:"not null;index"` // Индекс для быстрой фильтрации по проекту

	// Поля, которые мы синхронизируем из Jira
	Summary       string
	Status        string
	Assignee      string
	JiraUpdatedAt time.Time // Время обновления задачи в самой Jira

	// Мета-поля для нашей логики
	IsActive   bool      `gorm:"not null;default:true"` // Флаг для мягкого удаления
	LastSeenAt time.Time `gorm:"not null"`              // Когда мы последний раз видели эту задачу
}

// TableName явно указывает GORM, как назвать таблицу
func (Task) TableName() string {
	return "tasks"
}

// CustomTime для корректной десериализации времени из Jira
type CustomTime struct {
	time.Time
}

// UnmarshalJSON реализует интерфейс json.Unmarshaler
func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		ct.Time = time.Time{}
		return nil
	}

	// Поддержка различных форматов времени из Jira
	layouts := []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000+0700",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05+0700",
	}

	var err error
	for _, layout := range layouts {
		t, parseErr := time.Parse(layout, s)
		if parseErr == nil {
			*ct = CustomTime{t}
			return nil
		}
		err = parseErr
	}

	return fmt.Errorf("не удалось распарсить время '%s': %w", s, err)
}

type Storage struct {
	DB *gorm.DB
}

func InitDB() (*Storage, error) {
	dsn := os.Getenv("MYSQL_DSN")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	// Выполнение миграции для создания таблицы tasks
	if err := db.AutoMigrate(&Task{}); err != nil {
		return nil, fmt.Errorf("ошибка миграции таблицы tasks: %w", err)
	}

	fmt.Println("Миграция базы данных успешно выполнена")
	return &Storage{DB: db}, nil
}

func SyncProjectDB(db *gorm.DB, projectKey string) error {
	syncStartTime := time.Now().UTC()
	issuesFromJira, err := GetIssuesFromJira(projectKey)

	if err != nil {
		return fmt.Errorf("failed to get issues from Jira: %w", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		for _, task := range issuesFromJira {
			// Ищем существующую задачу по ExternalID
			var existingTask Task
			if err := tx.Where("external_id = ?", task.ExternalID).First(&existingTask).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					// Задача не существует, создаем ее
					if err := tx.Create(&task).Error; err != nil {
						return fmt.Errorf("failed to create issue %s: %w", task.ExternalID, err)
					}
				} else {
					return fmt.Errorf("failed to check existing issue %s: %w", task.ExternalID, err)
				}
			} else {
				// Задача существует, обновляем ее
				existingTask.Summary = task.Summary
				existingTask.Status = task.Status
				existingTask.Assignee = task.Assignee
				existingTask.JiraUpdatedAt = task.JiraUpdatedAt
				existingTask.LastSeenAt = syncStartTime
				existingTask.IsActive = true

				if err := tx.Save(&existingTask).Error; err != nil {
					return fmt.Errorf("failed to update issue %s: %w", task.ExternalID, err)
				}
			}
		}

		// Деактивация устаревших задач
		result := tx.Model(&Task{}).
			Where("project_key = ? AND is_active = ? AND last_seen_at < ?", projectKey, true, syncStartTime).
			Update("is_active", false)

		if result.Error != nil {
			return fmt.Errorf("ошибка при мягком удалении: %w", result.Error)
		}
		fmt.Printf("Деактивировано %d старых задач", result.RowsAffected)

		return nil
	})
	return err
}
func GetIssuesFromJira(projectKey string) ([]Task, error) {
	// Используем только один экземпляр клиента
	client := resty.New()
	client.SetBaseURL(os.Getenv("JIRA_URL"))
	client.SetBasicAuth(os.Getenv("JIRA_USER"), os.Getenv("JIRA_TOKEN"))

	// Проверим, есть ли у нас доступ к текущему пользователю
	resp, err := client.R().
		Get("/rest/api/2/myself")

	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к Jira: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("проверка авторизации не пройдена: %d, %s",
			resp.StatusCode(), resp.String())
	}

	fmt.Println("Авторизация в Jira успешна:", resp.String())

	// Правильный запрос JQL с ключом проекта в кавычках
	jqlQuery := fmt.Sprintf("project = \"%s\" ORDER BY key ASC", projectKey)

	// Пробуем получить задачи из проекта
	var result struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string     `json:"summary"`
				Updated CustomTime `json:"updated"`
				Status  struct {
					Name string `json:"name"`
				} `json:"status"`
				Assignee *struct {
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
			} `json:"fields"`
		} `json:"issues"`
		StartAt    int `json:"startAt"`
		MaxResults int `json:"maxResults"`
		Total      int `json:"total"`
	}

	resp, err = client.R().
		SetQueryParams(map[string]string{
			"jql":        jqlQuery,
			"startAt":    "0",
			"maxResults": "100",
			"fields":     "summary,status,assignee,updated",
		}).
		SetResult(&result).
		Get("/rest/api/2/search")

	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса задач: %w", err)
	}

	if !resp.IsSuccess() {
		// Если у нас проблема с проектом, попробуем получить список всех проектов
		var projects []struct {
			ID   string `json:"id"`
			Key  string `json:"key"`
			Name string `json:"name"`
		}

		projectResp, projectErr := client.R().
			SetHeader("Accept", "application/json").
			SetResult(&projects).
			Get("/rest/api/2/project")

		fmt.Printf("Статус запроса проектов: %d\n", projectResp.StatusCode())

		if projectErr == nil && projectResp.IsSuccess() {
			fmt.Println("Доступные проекты:")
			for _, project := range projects {
				fmt.Printf("- ID: %s, Key: %s, Name: %s\n", project.ID, project.Key, project.Name)
			}
		} else {
			fmt.Printf("Не удалось получить список проектов: %v\n", projectErr)
		}

		// Вернуть исходную ошибку
		return nil, fmt.Errorf("API request failed with status code: %d, body: %s",
			resp.StatusCode(), resp.String())
	}
	var allIssues []Task
	for _, issue := range result.Issues {
		var assigneeName string
		if issue.Fields.Assignee != nil {
			assigneeName = issue.Fields.Assignee.DisplayName
		} else {
			assigneeName = "Не назначен"
		}

		allIssues = append(allIssues, Task{
			ExternalID:    issue.Key,
			ProjectKey:    projectKey,
			Summary:       issue.Fields.Summary,
			Status:        issue.Fields.Status.Name,
			Assignee:      assigneeName,
			JiraUpdatedAt: issue.Fields.Updated.Time,
			LastSeenAt:    time.Now().UTC(),
			IsActive:      true,
		})
	}

	return allIssues, nil
}
