package handlers

import (
    "encoding/json"
    "net/http"
    "log"
    "sync"
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
