GET api/recipe/getAllRecipes -> Recipe[]  
GET api/recipe/getRecipe {recipe_id} -> Recipe?   
GET api/recipe/getIngredients {recipe_id} -> Ingredients[]  
GET api/recipe/search {string name} -> Recipe[20]  

GET api/item/getAllItems {} -> Item[]  
GET api/item/getItem {item_id} -> Item?

SSE api/inventory/updates 
```
const eventSource = new EventSource('/api/inventory/updates');

eventSource.onmessage = function(event) {
    const inventory = JSON.parse(event.data);
    console.log('Инвентарь обновлен:', inventory);
    // Обновляем UI
};

eventSource.onerror = function(error) {
    console.error('SSE error:', error);
};
```
GET api/inventory/getAll inventory[]  
POST api/inventory/craft {recipe_id} -> bool  

POST /api/inventory/set  
Content-Type: application/json  

{
    "itemId": 1,
    "quantity": 10
}
