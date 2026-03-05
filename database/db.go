package database

import (
    "database/sql"
    "log"
    "fmt"
    "os"
    _ "github.com/lib/pq"
)

func InitDB() *sql.DB {
    host := getEnv("DB_HOST", "localhost")
    port := getEnv("DB_PORT", "5432")
    user := getEnv("DB_USER", "myuser")
    password := getEnv("DB_PASSWORD", "mypassword")
    dbname := getEnv("DB_NAME", "minicraftDB")
    
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, dbname)

    // connStr = `psql "postgresql://172.20.0.2:5432/minicraftDB?user=myuser&password=mypassword&sslmode=disable"`
    
    // Подключаемся к базе
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal("Ошибка подключения к PostgreSQL:", err)
    }

    // Проверяем подключение
    err = db.Ping()
    if err != nil {
        log.Fatal("Не удалось подключиться к PostgreSQL:", err)
    }

    fmt.Printf("Успешно подключились к базе данных %s\n", dbname)
    
    return db
}

func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}
