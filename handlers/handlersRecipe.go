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


var db *sql.DB

func SetDB(database *sql.DB) {
    db = database
}

func GetAllRecipes(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    rows, err := db.Query(`
        SELECT r.id, r.itemId, r.quantity, r.duration, 
               i.name, i.description, i.maxStack
        FROM Recipes r
        JOIN Items i ON r.itemId = i.id
        ORDER BY r.id
    `)
    if err != nil {
        log.Printf("Ошибка получения рецептов: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var recipes []models.Recipe

    for rows.Next() {
        var recipe models.Recipe
        var resultItem models.Item
        
        err := rows.Scan(
            &recipe.ID, 
            &resultItem.ID, 
            &recipe.Quantity, 
            &recipe.Duration,
            &resultItem.Name, 
            &resultItem.Description, 
            &resultItem.MaxStack,
        )
        if err != nil {
            log.Printf("Ошибка сканирования рецепта: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        recipe.ResultItem = resultItem

        ingredientRows, err := db.Query(`
            SELECT ing.quantity, ing.posX, ing.posY,
                   i.id, i.name, i.description, i.maxStack
            FROM Ingredients ing
            JOIN Items i ON ing.itemId = i.id
            WHERE ing.recipeId = $1
            ORDER BY ing.posY, ing.posX
        `, recipe.ID)
        if err != nil {
            log.Printf("Ошибка получения ингредиентов для рецепта %d: %v", recipe.ID, err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        var ingredients []models.Ingredient
        for ingredientRows.Next() {
            var ingredient models.Ingredient
            var item models.Item
            
            err := ingredientRows.Scan(
                &ingredient.Quantity, 
                &ingredient.PosX, 
                &ingredient.PosY,
                &item.ID, 
                &item.Name, 
                &item.Description, 
                &item.MaxStack,
            )
            if err != nil {
                log.Printf("Ошибка сканирования ингредиента: %v", err)
                ingredientRows.Close()
                http.Error(w, "Internal server error", http.StatusInternalServerError)
                return
            }
            
            ingredient.Item = item
            ingredients = append(ingredients, ingredient)
        }
        ingredientRows.Close()
        
        recipe.Ingredients = ingredients
        recipes = append(recipes, recipe)
    }

    if err = rows.Err(); err != nil {
        log.Printf("Ошибка при итерации по рецептам: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if err := json.NewEncoder(w).Encode(recipes); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}


func GetRecipes(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    pathParts := strings.Split(r.URL.Path, "/")
    if len(pathParts) < 3 {
        http.Error(w, "ID рецепта не указан", http.StatusBadRequest)
        return
    }
    
    idStr := pathParts[len(pathParts)-1]
    if idStr == "" {
        http.Error(w, "ID рецепта не указан", http.StatusBadRequest)
        return
    }

    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Некорректный ID рецепта", http.StatusBadRequest)
        return
    }

    var recipe models.Recipe
    var resultItem models.Item
    
    err = db.QueryRow(`
        SELECT r.id, r.itemId, r.quantity, r.duration, 
               i.name, i.description, i.maxStack
        FROM Recipes r
        JOIN Items i ON r.itemId = i.id
        WHERE r.id = $1
    `, id).Scan(
        &recipe.ID, 
        &resultItem.ID, 
        &recipe.Quantity, 
        &recipe.Duration,
        &resultItem.Name, 
        &resultItem.Description, 
        &resultItem.MaxStack,
    )
    
    if err == sql.ErrNoRows {
        http.Error(w, "Рецепт не найден", http.StatusNotFound)
        return
    }
    if err != nil {
        log.Printf("Ошибка получения рецепта с ID %d: %v", id, err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    recipe.ResultItem = resultItem

    ingredientRows, err := db.Query(`
        SELECT ing.quantity, ing.posX, ing.posY,
               i.id, i.name, i.description, i.maxStack
        FROM Ingredients ing
        JOIN Items i ON ing.itemId = i.id
        WHERE ing.recipeId = $1
        ORDER BY ing.posY, ing.posX
    `, recipe.ID)
    if err != nil {
        log.Printf("Ошибка получения ингредиентов для рецепта %d: %v", recipe.ID, err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer ingredientRows.Close()

    var ingredients []models.Ingredient
    for ingredientRows.Next() {
        var ingredient models.Ingredient
        var item models.Item
        
        err := ingredientRows.Scan(
            &ingredient.Quantity, 
            &ingredient.PosX, 
            &ingredient.PosY,
            &item.ID, 
            &item.Name, 
            &item.Description, 
            &item.MaxStack,
        )
        if err != nil {
            log.Printf("Ошибка сканирования ингредиента: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        ingredient.Item = item
        ingredients = append(ingredients, ingredient)
    }

    if err = ingredientRows.Err(); err != nil {
        log.Printf("Ошибка при итерации по ингредиентам: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    recipe.Ingredients = ingredients

    if err := json.NewEncoder(w).Encode(recipe); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}


func getIngredients(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    rows, err := db.Query(`
        SELECT i.id, i.itemId, i.quantity, i.posX, i.posY,
               it.id, it.name, it.description, it.maxStack
        FROM Ingredients i
        JOIN Items it ON i.itemId = it.id
        ORDER BY i.recipeId, i.posY, i.posX
    `)
    if err != nil {
        log.Printf("Ошибка получения ингредиентов: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var ingredients []models.Ingredient

    for rows.Next() {
        var ingredient models.Ingredient
        var item models.Item
        var ingredientID int
        
        err := rows.Scan(
            &ingredientID,
            &ingredient.Quantity,
            &ingredient.PosX,
            &ingredient.PosY,
            &item.ID,
            &item.Name,
            &item.Description,
            &item.MaxStack,
        )
        if err != nil {
            log.Printf("Ошибка сканирования ингредиента: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        ingredient.Item = item
        ingredients = append(ingredients, ingredient)
    }

    if err = rows.Err(); err != nil {
        log.Printf("Ошибка при итерации по ингредиентам: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if err := json.NewEncoder(w).Encode(ingredients); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}

func GetSearchRescipe(w http.ResponseWriter, r *http.Request) {
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
        SELECT r.id, r.itemId, r.quantity, r.duration, 
               i.name, i.description, i.maxStack
        FROM Recipes r
        JOIN Items i ON r.itemId = i.id
        WHERE i.name ILIKE '%' || $1 || '%'
        ORDER BY r.id
        LIMIT 20
    `, name)
    if err != nil {
        log.Printf("Ошибка поиска рецептов: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var recipes []models.Recipe

    for rows.Next() {
        var recipe models.Recipe
        var resultItem models.Item
        
        err := rows.Scan(
            &recipe.ID, 
            &resultItem.ID, 
            &recipe.Quantity, 
            &recipe.Duration,
            &resultItem.Name, 
            &resultItem.Description, 
            &resultItem.MaxStack,
        )
        if err != nil {
            log.Printf("Ошибка сканирования рецепта: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        recipe.ResultItem = resultItem

        ingredientRows, err := db.Query(`
            SELECT ing.quantity, ing.posX, ing.posY,
                   i.id, i.name, i.description, i.maxStack
            FROM Ingredients ing
            JOIN Items i ON ing.itemId = i.id
            WHERE ing.recipeId = $1
            ORDER BY ing.posY, ing.posX
        `, recipe.ID)
        if err != nil {
            log.Printf("Ошибка получения ингредиентов для рецепта %d: %v", recipe.ID, err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        var ingredients []models.Ingredient
        for ingredientRows.Next() {
            var ingredient models.Ingredient
            var item models.Item
            
            err := ingredientRows.Scan(
                &ingredient.Quantity, 
                &ingredient.PosX, 
                &ingredient.PosY,
                &item.ID, 
                &item.Name, 
                &item.Description, 
                &item.MaxStack,
            )
            if err != nil {
                log.Printf("Ошибка сканирования ингредиента: %v", err)
                ingredientRows.Close()
                http.Error(w, "Internal server error", http.StatusInternalServerError)
                return
            }
            
            ingredient.Item = item
            ingredients = append(ingredients, ingredient)
        }
        ingredientRows.Close()
        
        recipe.Ingredients = ingredients
        recipes = append(recipes, recipe)
    }

    if err = rows.Err(); err != nil {
        log.Printf("Ошибка при итерации по рецептам: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if recipes == nil {
        recipes = make([]models.Recipe, 0)
    }

    if err := json.NewEncoder(w).Encode(recipes); err != nil {
        log.Printf("Ошибка кодирования JSON: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}
