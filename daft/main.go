package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jakeroggenbuck/BestNextStep/daft/step"
	"github.com/jakeroggenbuck/BestNextStep/daft/user"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
)

func getLogIn() gin.Accounts {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		fmt.Printf("ADMIN_PASSWORD not set")
		log.Fatal("ADMIN_PASSWORD not set")
	}

	return gin.Accounts{
		"Admin": password,
	}
}

func getLocalIP() string {
	ip := os.Getenv("LOCAL_IP")
	if ip == "" {
		fmt.Printf("LOCAL_IP not set")
		log.Fatal("LOCAL_IP not set")
	}

	return ip
}

func setupLogging() {
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)
}

func createDefaultElements(db *sql.DB) {
	stepRepository := step.NewSQLiteRepository(db)

	if err := stepRepository.Migrate(); err != nil {
		log.Fatal(err)
	}

	stepOne := step.Step{
		Name:  "Step One",
		Desc:  "The first step.",
		Left:  -1,
		Right: 2,
		Owner: 1,
	}
	stepTwo := step.Step{
		Name:  "Step Two",
		Desc:  "The second step.",
		Left:  1,
		Right: -1,
		Owner: 1,
	}

	createdStepOne, err := stepRepository.Create(stepOne)
	if err != nil {
		fmt.Println(err)
	}

	createdStepTwo, err := stepRepository.Create(stepTwo)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(createdStepOne)
	fmt.Println(createdStepTwo)
}

func dbExists() bool {
	if _, err := os.Stat("./sqlite.db"); err == nil {
		return true
	}
	return false
}

func main() {
	setupLogging()

	dbExisted := dbExists()
	db, err := sql.Open("sqlite3", "sqlite.db")
	if err != nil {
		log.Fatal("Database open failed")
	}

	stepRepository := step.NewSQLiteRepository(db)
	if err := stepRepository.Migrate(); err != nil {
		log.Fatal(err)
	}

	userRepository := user.NewSQLiteRepository(db)
	if err := userRepository.Migrate(); err != nil {
		log.Fatal(err)
	}

	// Create default items if db is new
	if !dbExisted {
		createDefaultElements(db)
	}

	router := gin.Default()
	router.SetTrustedProxies([]string{getLocalIP()})
	router.LoadHTMLGlob("./web/templates/**/*")

	router.Use(cors.Default())

	router.GET("/", homePage)

	// New auth for normal users in userRepository
	// https://github.com/yasaricli/gah
	// https://chenyitian.gitbooks.io/gin-tutorials/content/tdd/8.html
	authAccount := getLogIn()
	authedSubRoute := router.Group("/api/v1/", gin.BasicAuth(authAccount))

	authedSubRoute.GET("/", apiRootPage)

	authedSubRoute.GET("/all", func(c *gin.Context) {
		all, err := stepRepository.All()
		if err != nil {
			fmt.Print(err)
		}

		all_json, err := json.Marshal(all)
		if err != nil {
			fmt.Print(err)
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": string(all_json),
		})
	})

	authedSubRoute.POST("/new-user", func(c *gin.Context) {
		name := c.PostForm("name")
		password := c.PostForm("password")

		if name != "" && password != "" {
			hash, _ := HashPassword(password)
			userRepository.Create(user.User{Name: name, PasswordHash: hash})

			c.String(http.StatusOK, fmt.Sprint(userRepository.All()))
		} else {
			c.String(http.StatusNotAcceptable, "name or password empty")
		}

	})

	listenPort := os.Getenv("PORT")
	if listenPort == "" {
		listenPort = "1357"
	}

	fmt.Print("\nHosted at http://localhost:" + listenPort + "\n")
	router.Run(":" + listenPort)
}
