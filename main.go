package main

import (
    "log"
    "os"

    "github.com/joho/godotenv"
    "jobz/config"
)

func main() {
    godotenv.Load()
    if os.Getenv("JWT_SECRET") == "" {
        log.Fatal("JWT_SECRET is not set")
    }
    config.Connect()
    // ...rest
}