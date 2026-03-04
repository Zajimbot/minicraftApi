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

func CraftItem(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    // Получаем ID рецепта из URL
    pathParts := strings.Split(r.URL.Path, "/")
    if len(pathParts) < 4 {
        http.Error(w, "ID рецепта не указан", http.StatusBadRequest)
        return
    }
    
    recipeIDStr := pathParts[len(pathParts)-1]
    if recipeIDStr == "" {
        http.Error(w, "ID рецепта не указан", http.StatusBadRequest)
        return
    }

    recipeID, err := strconv.Atoi(recipeIDStr)
    if err != nil {
        http.Error(w, "Некорректный ID рецепта", http.StatusBadRequest)
        return
    }

    // Декодируем тело запроса с ингредиентами
    var craftReq models.CraftRequest
    if err := json.NewDecoder(r.Body).Decode(&craftReq); err != nil {
        log.Printf("Ошибка декодирования запроса: %v", err)
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Начинаем транзакцию
    tx, err := db.Begin()
    if err != nil {
        log.Printf("Ошибка начала транзакции: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer tx.Rollback() // Откатываем транзакцию в случае ошибки

    var recipe struct {
        ID           int
        ItemID       int
        Quantity     int
        Duration     int
        ItemName     string
        IsConsumable bool
    }

    err = tx.QueryRow(`
        SELECT r.id, r.itemId, r.quantity, r.duration, 
               i.name, i.consumable
        FROM Recipes r
        JOIN Items i ON r.itemId = i.id
        WHERE r.id = $1
    `, recipeID).Scan(
        &recipe.ID,
        &recipe.ItemID,
        &recipe.Quantity,
        &recipe.Duration,
        &recipe.ItemName,
        &recipe.IsConsumable,
    )

    if err == sql.ErrNoRows {
        sendJSONResponse(w, false)
        return
    }
    if err != nil {
        log.Printf("Ошибка получения рецепта: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Получаем ингредиенты рецепта из базы
    rows, err := tx.Query(`
        SELECT itemId, quantity, posX, posY
        FROM Ingredients
        WHERE recipeId = $1
    `, recipeID)
    if err != nil {
        log.Printf("Ошибка получения ингредиентов рецепта: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    recipeIngredients := make(map[string]models.CraftIngredient)
    for rows.Next() {
        var ing models.CraftIngredient
        err := rows.Scan(&ing.ItemID, &ing.Quantity, &ing.PosX, &ing.PosY)
        if err != nil {
            log.Printf("Ошибка сканирования ингредиента: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        key := getIngredientKey(ing.PosX, ing.PosY)
        recipeIngredients[key] = ing
    }

    // Проверяем, что все необходимые ингредиенты присутствуют в запросе
    if len(recipeIngredients) != len(craftReq.Ingredients) {
        sendJSONResponse(w, false)
        return
    }

    // Проверяем соответствие ингредиентов и наличие достаточного количества в инвентаре
    for _, reqIng := range craftReq.Ingredients {
        key := getIngredientKey(reqIng.PosX, reqIng.PosY)
        recipeIng, exists := recipeIngredients[key]
        
        if !exists {
            sendJSONResponse(w, false)
            return
        }

        // Проверяем соответствие itemId и количества
        if recipeIng.ItemID != reqIng.ItemID || recipeIng.Quantity != reqIng.Quantity {
            sendJSONResponse(w, false)
            return
        }

        // Проверяем наличие предметов в инвентаре
        var inventoryQuantity int
        err = tx.QueryRow(`
            SELECT quantity 
            FROM Inventory 
            WHERE itemId = $1 AND posX = $2 AND posY = $3
        `, reqIng.ItemID, reqIng.PosX, reqIng.PosY).Scan(&inventoryQuantity)

        if err == sql.ErrNoRows {
            sendJSONResponse(w, false)
            return
        }
        if err != nil {
            log.Printf("Ошибка проверки инвентаря: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        if inventoryQuantity < reqIng.Quantity {
            sendJSONResponse(w, false)
            return
        }
    }

    // Все проверки пройдены, выполняем крафт

    // 1. Уменьшаем количество ингредиентов в инвентаре
    for _, reqIng := range craftReq.Ingredients {
        // Проверяем, является ли предмет расходуемым
        var isConsumable bool
        err = tx.QueryRow(`
            SELECT consumable FROM Items WHERE id = $1
        `, reqIng.ItemID).Scan(&isConsumable)

        if err != nil {
            log.Printf("Ошибка проверки расходуемости предмета: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        // Если предмет не расходуемый, не уменьшаем его количество
        if !isConsumable {
            continue
        }

        // Уменьшаем количество расходуемых предметов
        _, err = tx.Exec(`
            UPDATE Inventory 
            SET quantity = quantity - $1 
            WHERE itemId = $2 AND posX = $3 AND posY = $4 AND quantity >= $1
        `, reqIng.Quantity, reqIng.ItemID, reqIng.PosX, reqIng.PosY)

        if err != nil {
            log.Printf("Ошибка обновления инвентаря: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        // Удаляем записи с нулевым количеством
        _, err = tx.Exec(`
            DELETE FROM Inventory 
            WHERE itemId = $1 AND posX = $2 AND posY = $3 AND quantity <= 0
        `, reqIng.ItemID, reqIng.PosX, reqIng.PosY)

        if err != nil {
            log.Printf("Ошибка удаления пустых слотов: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
    }

    // 2. Добавляем результат крафта в инвентарь
    // Ищем свободный слот или существующий стак с таким же предметом
    var targetPosX, targetPosY int
    var existingSlot bool

    // Сначала ищем существующий слот с таким же предметом, если позволяет maxStack
    var maxStack int
    err = tx.QueryRow(`
        SELECT maxStack FROM Items WHERE id = $1
    `, recipe.ItemID).Scan(&maxStack)
    if err != nil {
        log.Printf("Ошибка получения maxStack: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if maxStack > 1 {
        // Ищем слот с таким же предметом, который не заполнен до максимума
        err = tx.QueryRow(`
            SELECT posX, posY 
            FROM Inventory 
            WHERE itemId = $1 AND quantity < $2
            ORDER BY quantity DESC
            LIMIT 1
        `, recipe.ItemID, maxStack).Scan(&targetPosX, &targetPosY)
        
        if err == nil {
            existingSlot = true
        }
    }

    if existingSlot {
        // Добавляем к существующему слоту
        _, err = tx.Exec(`
            UPDATE Inventory 
            SET quantity = quantity + $1 
            WHERE posX = $2 AND posY = $3
        `, recipe.Quantity, targetPosX, targetPosY)
    } else {
        // Ищем первый свободный слот
        err = tx.QueryRow(`
            SELECT posX, posY 
            FROM (
                SELECT generate_series(0, 9) AS posX,
                       generate_series(0, 4) AS posY
            ) AS all_positions
            WHERE NOT EXISTS (
                SELECT 1 FROM Inventory 
                WHERE Inventory.posX = all_positions.posX 
                AND Inventory.posY = all_positions.posY
            )
            ORDER BY posY, posX
            LIMIT 1
        `).Scan(&targetPosX, &targetPosY)

        if err != nil {
            log.Printf("Нет свободного места в инвентаре: %v", err)
            http.Error(w, "Inventory is full", http.StatusBadRequest)
            return
        }

        // Создаем новый слот
        _, err = tx.Exec(`
            INSERT INTO Inventory (itemId, quantity, posX, posY)
            VALUES ($1, $2, $3, $4)
        `, recipe.ItemID, recipe.Quantity, targetPosX, targetPosY)
    }

    if err != nil {
        log.Printf("Ошибка добавления результата крафта: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Фиксируем транзакцию
    if err = tx.Commit(); err != nil {
        log.Printf("Ошибка фиксации транзакции: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    BroadcastInventoryUpdate()
    // Возвращаем успешный результат
    sendJSONResponse(w, true)
}

// Вспомогательная функция для создания ключа по координатам
func getIngredientKey(posX, posY int) string {
    return strconv.Itoa(posX) + ":" + strconv.Itoa(posY)
}

// Вспомогательная функция для отправки JSON ответа
func sendJSONResponse(w http.ResponseWriter, success bool) {
    response := map[string]bool{"success": success}
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}



