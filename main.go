package main

import (
    "log"
    "net/http"
    "minicraft-api/database"
    "minicraft-api/handlers"
)

func main() {
    db := database.InitDB()
    defer db.Close()
    
    handlers.SetDB(db)


    http.HandleFunc("/api/recipes", handlers.GetAllRecipes)
    http.HandleFunc("/api/recipes/", handlers.GetRecipes) 
    http.HandleFunc("/api/recipes/search", handlers.GetSearchRescipe)
    http.HandleFunc("/api/items", handlers.GetAllItems)
    http.HandleFunc("/api/items/", handlers.GetItem)
    http.HandleFunc("/api/items/search", handlers.SearchItems)
    http.HandleFunc("/api/inventory/craft/", handlers.CraftItem)

    http.HandleFunc("/api/inventory/set", handlers.SetInventoryItem) 
    http.HandleFunc("/api/inventory/getAll", handlers.GetAllInventory)
    http.HandleFunc("/api/inventory/updates", handlers.InventoryUpdatesSSE)

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
