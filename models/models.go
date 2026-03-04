package models


type Item struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    ImageUrl    string `json:"ImageUrl"`   
    MaxStack    int    `json:"maxStack"`
}

type Ingredient struct {
    Item     Item `json:"item"`
    Quantity int  `json:"quantity"`
    PosX     int  `json:"posX"`
    PosY     int  `json:"posY"`
}

type Recipe struct {
    ID          int          `json:"id"`
    ResultItem  Item         `json:"resultItem"`
    Quantity    int          `json:"quantity"`
    Duration    int          `json:"duration"`
    Ingredients []Ingredient `json:"ingredients"`
}

type InventoryItem struct {
    ID       int `json:"id"`
    Item     Item `json:"item"`
    Quantity int `json:"quantity"`
    PosX     int `json:"posX"`
    PosY     int `json:"posY"`
}

type CraftRequest struct {
    Ingredients []CraftIngredient `json:"ingredients"`
}

type CraftIngredient struct {
    ItemID   int `json:"itemId"`
    Quantity int `json:"quantity"`
    PosX     int `json:"posX"`
    PosY     int `json:"posY"`
}

