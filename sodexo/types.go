package sodexo

type MenuDay struct {
	MenuDayID         int64       `json:"menuDayId"`
	MenuID            int64       `json:"menuId"`
	Date              string      `json:"date"`
	Meal              interface{} `json:"meal"`
	IsFakeMenuDay     bool        `json:"isFakeMenuDay"`
	DayParts          []DayPart   `json:"dayParts"`
	MenuItems         []MenuItem  `json:"menuItems"`
	FoodItemSourceKey interface{} `json:"foodItemSourceKey"`
	CurrentMenu       bool        `json:"currentMenu"`
}

type DayPart struct {
	DayPartName DayPartName     `json:"dayPartName"`
	MenuDayID   int64           `json:"menuDayId"`
	Courses     []CourseElement `json:"courses"`
}

type CourseElement struct {
	CourseName string     `json:"courseName"`
	MenuDayID  int64      `json:"menuDayId"`
	MenuItems  []MenuItem `json:"menuItems"`
}

type MenuItem struct {
	MenuItemID                 int64       `json:"menuItemId"`
	MenuDayID                  int64       `json:"menuDayId"`
	FoodItemID                 int64       `json:"foodItemId"`
	StartTime                  string      `json:"startTime"`
	EndTime                    string      `json:"endTime"`
	Meal                       DayPartName `json:"meal"`
	Course                     string      `json:"course"`
	CourseSortOrder            int64       `json:"courseSortOrder"`
	MenuItemSortOrder          int64       `json:"menuItemSortOrder"`
	FoodItemSourceKey          string      `json:"foodItemSourceKey"`
	FormalName                 string      `json:"formalName"`
	Number                     string      `json:"number"`
	Description                *string     `json:"description"`
	FoodSubCategoryDescription string      `json:"foodSubCategoryDescription"`
	IsFavorite                 bool        `json:"isFavorite"`
	IsVegan                    bool        `json:"isVegan"`
	IsVegetarian               bool        `json:"isVegetarian"`
	IsMindful                  bool        `json:"isMindful"`
	Kcal                       float64     `json:"kcal"`
	Calories                   string      `json:"calories"`
	CaloriesFromFat            string      `json:"caloriesFromFat"`
	Fat                        string      `json:"fat"`
	SaturatedFat               string      `json:"saturatedFat"`
	TransFat                   string      `json:"transFat"`
	PolyunsaturatedFat         string      `json:"polyunsaturatedFat"`
	Cholesterol                string      `json:"cholesterol"`
	Sodium                     string      `json:"sodium"`
	DietaryFiber               string      `json:"dietaryFiber"`
	Sugar                      string      `json:"sugar"`
	Protein                    string      `json:"protein"`
	Potassium                  string      `json:"potassium"`
	PortionSize                string      `json:"portionSize"`
	Portion                    string      `json:"portion"`
	GramWeight                 string      `json:"gramWeight"`
	Carbohydrates              string      `json:"carbohydrates"`
	Iron                       string      `json:"iron"`
	Calcium                    string      `json:"calcium"`
	VitaminA                   string      `json:"vitaminA"`
	VitaminC                   string      `json:"vitaminC"`
	Allergens                  []Allergen  `json:"allergens"`
	PriceWithTax               int64       `json:"priceWithTax"`
	Tax                        int64       `json:"tax"`
	CurrencySymbol             string      `json:"currencySymbol"`
	IsOrderItem                bool        `json:"isOrderItem"`
	UomID                      int64       `json:"uomId"`
}

type Allergen struct {
	Name     string      `json:"name"`
	Contains string      `json:"contains"`
	Icon     interface{} `json:"icon"`
	Allergen string      `json:"allergen"`
}

type DayPartName string

const (
	Breakfast DayPartName = "Breakfast"
	Brunch    DayPartName = "Brunch"
	Dinner    DayPartName = "Dinner"
	Lunch     DayPartName = "Lunch"
)
