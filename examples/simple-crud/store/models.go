// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.23.0

package store

import ()

type Dosing struct {
	RecipeID     string `json:"recipe_id"`
	IngredientID string `json:"ingredient_id"`
	Quantity     int64  `json:"quantity" validate:"required,gt=0"`
	Unit         string `json:"unit"`
}

type Ingredient struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Recipe struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
