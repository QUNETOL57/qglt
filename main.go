package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var draftSuffix = "Draft: "

var config Config

type Config struct {
	GLURL            string
	GLPrivateToken   string
	GLAssigneeID     int
	GLProjectID      int
	GLReviewerIDs    []int
	GLTargetBranches []string
	MeteorLink       string
	UserPrefix       string
}

func loadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	assigneeID, err := strconv.Atoi(os.Getenv("GL_ASSIGNEE_ID"))
	if err != nil {
		log.Fatalf("Error converting GL_ASSIGNEE_ID: %v", err)
	}

	projectID, err := strconv.Atoi(os.Getenv("GL_PROJECT_ID"))
	if err != nil {
		log.Fatalf("Error converting GL_PROJECT_ID: %v", err)
	}

	reviewerIDsStr := os.Getenv("GL_REVIEWER_IDS")
	reviewerIDs := make([]int, 0)
	for _, idStr := range strings.Split(reviewerIDsStr, ",") {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Fatalf("Error converting GL_REVIEWER_IDS: %v", err)
		}
		reviewerIDs = append(reviewerIDs, id)
	}

	targetBranches := strings.Split(os.Getenv("GL_TARGET_BRANCHES"), ",")

	config = Config{
		GLURL:            os.Getenv("GL_URL"),
		GLPrivateToken:   os.Getenv("GL_PRIVATE_TOKEN"),
		GLAssigneeID:     assigneeID,
		GLProjectID:      projectID,
		GLReviewerIDs:    reviewerIDs,
		GLTargetBranches: targetBranches,
		MeteorLink:       os.Getenv("METEOR_LINK"),
		UserPrefix:       os.Getenv("USER_PREFIX"),
	}
}

func send(targetBranch string, sourceBranch string, title string, description string) {
	logPrefix := "|" + targetBranch + "|"

	data := map[string]interface{}{
		"source_branch": sourceBranch,
		"target_branch": targetBranch,
		"title":         title,
		"description":   description,
		"assignee_id":   config.GLAssigneeID,
		"reviewer_ids":  config.GLReviewerIDs,
		"squash":        true,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(logPrefix+"Ошибка при преобразовании данных в JSON:", err)
		return
	}

	// Создание HTTP-запроса
	url := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests", config.GLURL, config.GLProjectID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(logPrefix+"Ошибка при создании HTTP-запроса:", err)
		return
	}

	// Установка заголовков запроса
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", config.GLPrivateToken)

	//Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(logPrefix+"Ошибка при выполнении HTTP-запроса:", err)
		return
	}
	defer resp.Body.Close()

	////Проверка статуса ответа
	if resp.StatusCode != http.StatusCreated {
		fmt.Println(logPrefix+"Ошибка при создании Merge Request:", resp.Status)
		return
	}

	fmt.Println(logPrefix + "Merge Request успешно создан!")
}

func main() {
	loadConfig()
	// Название задачи, копируется из метеора, обязательно в формате "ВВ-11111 текст"
	taskName := ""
	// Название ветки feature/48904
	sourceBranch := ""

	taskMeteorLink := config.MeteorLink
	re := regexp.MustCompile(`ВВ-\d+`)

	title := re.ReplaceAllString(taskName, "")
	title = config.UserPrefix + " " + sourceBranch + title

	match := re.FindStringSubmatch(taskName)
	if len(match) > 0 {
		taskMeteorLink += match[0]
	}
	for i := 0; i < len(config.GLTargetBranches); i++ {
		newTitle := title
		if config.GLTargetBranches[i] != "dev" {
			newTitle = draftSuffix + newTitle
		}
		fmt.Println(newTitle)
		send(config.GLTargetBranches[i], sourceBranch, newTitle, taskMeteorLink)
	}
}
