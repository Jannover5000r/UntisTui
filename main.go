package main

import (
	"os"

	untis "UntisTui/untis"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	user := os.Getenv("USER")
	pass := os.Getenv("PASS")

	untis.Main(user, pass)
}
