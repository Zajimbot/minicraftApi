package handlers

import (
    "encoding/json"
    "net/http"
    "log"
    "sync"
    "database/sql"
    "minicraft-api/models"
)

type SSEClient struct {
    mu      sync.Mutex
    clients map[chan []byte]bool
}

var sseManager = &SSEClient{
    clients: make(map[chan []byte]bool),
}

func GetAllInventory(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    rows, err := db.Query(`
        SELECT i.id, inv.itemId, inv.quantity, inv.posX, inv.posY,
               it.name, it.description, it.imageUrl, it.maxStack
        FROM Inventory inv
        JOIN Items it ON inv.itemId = it.id
        ORDER BY inv.posY, inv.posX
    `)
    if err != nil {
        log.Printf("Ошибка получения инвентаря: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var inventory []models.InventoryItem

    for rows.Next() {
        var invItem models.InventoryItem
        var item models.Item
        
        err := rows.Scan(
            &invItem.ID,
            &item.ID,
            &invItem.Quantity,
            &invItem.PosX,
            &invItem.PosY,
            &item.Name,
            &item.Description,
            &item.ImageUrl,
            &item.MaxStack,
        )
        if err != nil {
            log.Printf("Ошибка сканирования инвентаря: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        invItem.Item = item
        inventory = append(inventory, invItem)
    }

    if err = rows.Err(); err != nil {
        log.Printf("Ошибка при итерации по инвентарю: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if inventory == nil {
        inventory = make([]models.InventoryItem, 0)
    }

    if err := json.NewEncoder(w).Encode(inventory); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}

func InventoryUpdatesSSE(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Устанавливаем заголовки для SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // Создаем канал для этого клиента
    messageChan := make(chan []byte)
    
    // Регистрируем клиента
    sseManager.mu.Lock()
    sseManager.clients[messageChan] = true
    sseManager.mu.Unlock()

    log.Printf("Новый SSE клиент подключен. Всего клиентов: %d", len(sseManager.clients))

    // Убираем клиента при завершении
    defer func() {
        sseManager.mu.Lock()
        delete(sseManager.clients, messageChan)
        close(messageChan)
        sseManager.mu.Unlock()
        log.Printf("SSE клиент отключен. Всего клиентов: %d", len(sseManager.clients))
    }()

    // Создаем канал для отслеживания закрытия соединения
    notify := r.Context().Done()

    // Отправляем начальное состояние инвентаря
    go func() {
        inventory, err := getCurrentInventory()
        if err == nil {
            data, _ := json.Marshal(inventory)
            select {
            case messageChan <- data:
            default:
            }
        }
    }()

    // Ждем сообщений или закрытия соединения
    for {
        select {
        case <-notify:
            return
        case msg := <-messageChan:
            // Форматируем сообщение как SSE event
            _, err := w.Write([]byte("data: " + string(msg) + "\n\n"))
            if err != nil {
                return
            }
            w.(http.Flusher).Flush()
        }
    }
}

// BroadcastInventoryUpdate отправляет обновление инвентаря всем подключенным клиентам
func BroadcastInventoryUpdate() {
    inventory, err := getCurrentInventory()
    if err != nil {
        log.Printf("Ошибка получения инвентаря для рассылки: %v", err)
        return
    }

    data, err := json.Marshal(inventory)
    if err != nil {
        log.Printf("Ошибка маршалинга инвентаря: %v", err)
        return
    }

    sseManager.mu.Lock()
    defer sseManager.mu.Unlock()

    for ch := range sseManager.clients {
        select {
        case ch <- data:
        default:
            // Если канал заблокирован, пропускаем
        }
    }
}

// Вспомогательная функция для получения текущего состояния инвентаря
func getCurrentInventory() ([]models.InventoryItem, error) {
    rows, err := db.Query(`
        SELECT inv.id, inv.itemId, inv.quantity, inv.posX, inv.posY,
               it.name, it.description, it.imageUrl, it.maxStack
        FROM Inventory inv
        JOIN Items it ON inv.itemId = it.id
        ORDER BY inv.posY, inv.posX
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var inventory []models.InventoryItem
    for rows.Next() {
        var invItem models.InventoryItem
        var item models.Item
        
        err := rows.Scan(
            &invItem.ID,
            &item.ID,
            &invItem.Quantity,
            &invItem.PosX,
            &invItem.PosY,
            &item.Name,
            &item.Description,
            &item.ImageUrl,
            &item.MaxStack,
        )
        if err != nil {
            continue
        }
        
        invItem.Item = item
        inventory = append(inventory, invItem)
    }

    return inventory, nil
}

func SetInventoryItem(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    // Декодируем тело запроса
    var req struct {
        ItemID   int `json:"itemId"`
        Quantity int `json:"quantity"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Ошибка декодирования запроса: %v", err)
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Валидация входных данных
    if req.ItemID <= 0 {
        http.Error(w, "Invalid item ID", http.StatusBadRequest)
        return
    }
    if req.Quantity <= 0 {
        http.Error(w, "Quantity must be positive", http.StatusBadRequest)
        return
    }

    // Начинаем транзакцию
    tx, err := db.Begin()
    if err != nil {
        log.Printf("Ошибка начала транзакции: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer tx.Rollback()

    // Получаем информацию о предмете
    var itemInfo struct {
        MaxStack int
        Name     string
    }
    err = tx.QueryRow(`
        SELECT maxStack, name 
        FROM Items 
        WHERE id = $1
    `, req.ItemID).Scan(&itemInfo.MaxStack, &itemInfo.Name)

    if err == sql.ErrNoRows {
        http.Error(w, "Item not found", http.StatusNotFound)
        return
    }
    if err != nil {
        log.Printf("Ошибка получения информации о предмете: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    remainingQuantity := req.Quantity

    // Если предмет может стаковаться, пытаемся добавить к существующим стекам
    if itemInfo.MaxStack > 1 {
        // Ищем существующие слоты с этим предметом, которые не заполнены до максимума
        rows, err := tx.Query(`
            SELECT id, quantity, posX, posY 
            FROM Inventory 
            WHERE itemId = $1 AND quantity < $2
            ORDER BY quantity DESC
        `, req.ItemID, itemInfo.MaxStack)
        
        if err != nil {
            log.Printf("Ошибка поиска существующих слотов: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        for rows.Next() && remainingQuantity > 0 {
            var slot struct {
                ID       int
                Quantity int
                PosX     int
                PosY     int
            }
            err := rows.Scan(&slot.ID, &slot.Quantity, &slot.PosX, &slot.PosY)
            if err != nil {
                rows.Close()
                log.Printf("Ошибка сканирования слота: %v", err)
                http.Error(w, "Internal server error", http.StatusInternalServerError)
                return
            }

            availableSpace := itemInfo.MaxStack - slot.Quantity
            addQuantity := remainingQuantity
            if addQuantity > availableSpace {
                addQuantity = availableSpace
            }

            // Обновляем существующий слот
            _, err = tx.Exec(`
                UPDATE Inventory 
                SET quantity = quantity + $1 
                WHERE id = $2
            `, addQuantity, slot.ID)

            if err != nil {
                rows.Close()
                log.Printf("Ошибка обновления слота: %v", err)
                http.Error(w, "Internal server error", http.StatusInternalServerError)
                return
            }

            remainingQuantity -= addQuantity
        }
        rows.Close()
    }

    // Если остались предметы для добавления, ищем свободные слоты
    for remainingQuantity > 0 {
        // Ищем первый свободный слот с координатами: ширина до 8, высота > 0
        var posX, posY int
        err = tx.QueryRow(`
            SELECT posX, posY 
            FROM (
                SELECT generate_series(0, 7) AS posX,
                       generate_series(1, 4) AS posY
            ) AS all_positions
            WHERE NOT EXISTS (
                SELECT 1 FROM Inventory 
                WHERE Inventory.posX = all_positions.posX 
                AND Inventory.posY = all_positions.posY
            )
            ORDER BY posY, posX
            LIMIT 1
        `).Scan(&posX, &posY)

        if err != nil {
            log.Printf("Нет свободного места в инвентаре: %v", err)
            http.Error(w, "Inventory is full", http.StatusBadRequest)
            return
        }

        // Определяем количество для нового стека
        addQuantity := remainingQuantity
        if itemInfo.MaxStack > 1 && addQuantity > itemInfo.MaxStack {
            addQuantity = itemInfo.MaxStack
        }

        // Создаем новый слот
        _, err = tx.Exec(`
            INSERT INTO Inventory (itemId, quantity, posX, posY)
            VALUES ($1, $2, $3, $4)
        `, req.ItemID, addQuantity, posX, posY)

        if err != nil {
            log.Printf("Ошибка создания нового слота: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        remainingQuantity -= addQuantity
    }

    // Фиксируем транзакцию
    if err = tx.Commit(); err != nil {
        log.Printf("Ошибка фиксации транзакции: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Отправляем обновление всем подключенным клиентам
    BroadcastInventoryUpdate()

    // Возвращаем успешный результат
    response := map[string]bool{"success": true}
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
    }
}
