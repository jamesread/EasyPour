package cvar

const (
	TypeString = "string"
	TypeInt    = "int"
	TypeBool   = "bool"

	KeySiteTitle  = "site_title"
	KeyAppriseURL = "apprise_url"

	DefaultSiteTitle = "EasyPour"

	CategorySite          = "Site"
	CategoryNotifications = "Notifications"
)

// Def is a catalog entry for a configuration variable.
type Def struct {
	Key, MainType, Title, Description, Category, ValueString string
	Ordinal, ValueInt                                         int
}

// Defaults returns the built-in cvar catalog. siteTitle seeds site_title on first insert only.
func Defaults(siteTitle string) []Def {
	if siteTitle == "" {
		siteTitle = DefaultSiteTitle
	}
	return []Def{
		{
			Key: KeySiteTitle, MainType: TypeString, ValueString: siteTitle,
			Title: "Site title", Description: "Shown in the header and browser tab.",
			Category: CategorySite, Ordinal: 10,
		},
		{
			Key: KeyAppriseURL, MainType: TypeString, ValueString: "",
			Title: "Apprise URL",
			Description: "Apprise API notify URL for new orders " +
				"(e.g. http://apprise:8000/notify/ or http://apprise:8000/notify/mytag). Leave empty to disable.",
			Category: CategoryNotifications, Ordinal: 20,
		},
	}
}
