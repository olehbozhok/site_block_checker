package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/olehbozhok/site_block_checker/repo"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getVal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("not found env %s", key)
	}
	return value
}
func main() {
	_ = godotenv.Load(".env")

	db, err := repo.InitDB(
		getVal("DB_USER"),
		getVal("DB_PASS"),
		getVal("DB_HOST"),
		getVal("DB_DBNAME"),
		true,
	)
	checkErr(err)

	log.Println("done db init")

	urls := []string{
		"https://stopwar.in.ua",
		"https://standforukraine.com/",
		"https://rf.prayforukraine.art/",
		"http://project5270357.tilda.ws/sprotyv/eng",
		"http://project5270357.tilda.ws",
		"http://project5270357.tilda.ws/sprotyv/ru",
		"http://project5270357.tilda.ws/sprotyv/pl",
		"http://stopputin.today",
		"https://opinion-ru.com/",
		"http://stopputin.today",
		"https://help-to-stop-the-war.com/",
		"https://helpukrarmy.com/ ",
		"https://ostanovii-voiny.com/ ",
		"https://net-voine.com/ ",
		"https://protectionukraine.com/",
	}

	for _, url := range urls {
		err := db.AddURL(repo.CheckURL{
			URL: url,
		})
		if err != nil {
			log.Println("err add url:", err)
		}
	}
	log.Println("done add url")
}
