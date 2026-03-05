package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "log"
    "strconv"
    "strings"
    "minicraft-api/models"
)

func GetAllItems(w http.ResponseWriter, r *http.Request) {
    log.Println("GetAllItems")
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    rows, err := db.Query(`
        SELECT id, name, description, imageUrl, maxStack
        FROM Items
        ORDER BY id
    `)
    if err != nil {
        log.Printf("Ошибка получения предметов: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var items []models.Item

    for rows.Next() {
        var item models.Item
        
        err := rows.Scan(
            &item.ID,
            &item.Name,
            &item.Description,
            &item.ImageUrl,
            &item.MaxStack,
        )
        if err != nil {
            log.Printf("Ошибка сканирования предмета: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        items = append(items, item)
    }

    if err = rows.Err(); err != nil {
        log.Printf("Ошибка при итерации по предметам: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if items == nil {
        items = make([]models.Item, 0)
    }

    if err := json.NewEncoder(w).Encode(items); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}

func GetItem(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    pathParts := strings.Split(r.URL.Path, "/")
    if len(pathParts) < 3 {
        http.Error(w, "ID предмета не указан", http.StatusBadRequest)
        return
    }
    
    idStr := pathParts[len(pathParts)-1]
    if idStr == "" {
        http.Error(w, "ID предмета не указан", http.StatusBadRequest)
        return
    }

    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Некорректный ID предмета", http.StatusBadRequest)
        return
    }

    var item models.Item
    
    err = db.QueryRow(`
        SELECT id, name, description, imageUrl, maxStack
        FROM Items
        WHERE id = $1
    `, id).Scan(
        &item.ID,
        &item.Name,
        &item.Description,
        &item.ImageUrl,
        &item.MaxStack,
    )
    
    if err == sql.ErrNoRows {
        http.Error(w, "Предмет не найден", http.StatusNotFound)
        return
    }
    if err != nil {
        log.Printf("Ошибка получения предмета с ID %d: %v", id, err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if err := json.NewEncoder(w).Encode(item); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}

func SearchItems(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    name := r.URL.Query().Get("name")
    if name == "" {
        http.Error(w, "Parameter 'name' is required", http.StatusBadRequest)
        return
    }

    rows, err := db.Query(`
        SELECT id, name, description, imageUrl, maxStack
        FROM Items
        WHERE name ILIKE '%' || $1 || '%'
        ORDER BY name
        LIMIT 50
    `, name)
    if err != nil {
        log.Printf("Ошибка поиска предметов: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var items []models.Item

    for rows.Next() {
        var item models.Item
        
        err := rows.Scan(
            &item.ID,
            &item.Name,
            &item.Description,
            &item.ImageUrl,
            &item.MaxStack,
        )
        if err != nil {
            log.Printf("Ошибка сканирования предмета: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        items = append(items, item)
    }

    if err = rows.Err(); err != nil {
        log.Printf("Ошибка при итерации по предметам: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if items == nil {
        items = make([]models.Item, 0)
    }

    if err := json.NewEncoder(w).Encode(items); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}
