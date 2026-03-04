GET api/recipe/getAllRecipes -> Recipe[]
GET api/recipe/getRecipe {recipe_id} -> Recipe? 
GET api/recipe/getIngredients {recipe_id} -> Ingredients[]
GET api/recipe/search {string name} -> Recipe[20]

GET api/item/getAllItems {} -> Item[]
GET api/item/getItem {item_id} -> Item?

SSE api/inventory/updates
POST api/inventory/craft {recipe_id} -> bool
