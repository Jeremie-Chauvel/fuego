package views

import (
	"slices"
	"strings"

	"simple-crud/store"
	"simple-crud/templa/admin"
	"simple-crud/templa/components"

	"github.com/go-fuego/fuego"
)

func (rs Ressource) deleteRecipe(c fuego.ContextNoBody) (any, error) {
	id := c.QueryParam("id") // TODO use PathParam
	err := rs.RecipesQueries.DeleteRecipe(c.Context(), id)
	if err != nil {
		return nil, err
	}

	return c.Redirect(301, "/admin/recipes")
}

func (rs Ressource) adminRecipes(c fuego.ContextNoBody) (fuego.Templ, error) {
	searchParams := components.SearchParams{
		Name:    c.QueryParam("name"),
		PerPage: c.QueryParamInt("perPage", 20),
		Page:    c.QueryParamInt("page", 1),
		URL:     "/admin/ingredients",
		Lang:    c.MainLang(),
	}
	recipes, err := rs.RecipesQueries.GetRecipes(c.Context())
	if err != nil {
		return nil, err
	}

	return admin.RecipeList(recipes, searchParams), nil
}

func (rs Ressource) adminOneRecipe(c *fuego.ContextWithBody[store.UpdateRecipeParams]) (fuego.Templ, error) {
	id := c.QueryParam("id") // TODO use PathParam

	if c.Request().Method == "PUT" {
		updateRecipeBody, err := c.Body()
		if err != nil {
			return nil, err
		}

		updateRecipeBody.ID = id

		_, err = rs.RecipesQueries.UpdateRecipe(c.Context(), updateRecipeBody)
		if err != nil {
			return nil, err
		}

		c.Response().Header().Set("HX-Trigger", "entity-updated")
	}

	recipe, err := rs.RecipesQueries.GetRecipe(c.Context(), id)
	if err != nil {
		return nil, err
	}

	dosings, err := rs.IngredientsQueries.GetIngredientsOfRecipe(c.Context(), id)
	if err != nil {
		return nil, err
	}

	allIngredients, err := rs.IngredientsQueries.GetIngredients(c.Context())
	if err != nil {
		return nil, err
	}

	slices.SortFunc(allIngredients, func(a, b store.Ingredient) int {
		return strings.Compare(a.Name, b.Name)
	})

	return admin.RecipePage(admin.RecipePageProps{
		Recipe:         recipe,
		Dosings:        dosings,
		AllIngredients: allIngredients,
	}), nil
}

func (rs Ressource) editRecipe(c *fuego.ContextWithBody[store.UpdateRecipeParams]) (any, error) {
	updateRecipeArgs, err := c.Body()
	if err != nil {
		return "", err
	}

	updateRecipeArgs.ID = c.QueryParam("id") // TODO use PathParam

	recipe, err := rs.RecipesQueries.UpdateRecipe(c.Context(), updateRecipeArgs)
	if err != nil {
		return "", err
	}

	return c.Redirect(301, "/admin/recipes/one?id="+recipe.ID)
}

func (rs Ressource) adminAddRecipes(c *fuego.ContextWithBody[store.CreateRecipeParams]) (any, error) {
	body, err := c.Body()
	if err != nil {
		return "", err
	}

	_, err = rs.RecipesQueries.CreateRecipe(c.Context(), body)
	if err != nil {
		return "", err
	}

	return c.Redirect(301, "/admin/recipes")
}

func (rs Ressource) adminAddDosing(c *fuego.ContextWithBody[store.CreateDosingParams]) (any, error) {
	body, err := c.Body()
	if err != nil {
		return "", err
	}

	_, err = rs.DosingQueries.CreateDosing(c.Context(), body)
	if err != nil {
		return "", err
	}

	return c.Redirect(301, "/admin/recipes/one?id="+body.RecipeID)
}
