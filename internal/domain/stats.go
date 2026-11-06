package domain

// Stats holds the key figures shown on the dashboard.
type Stats struct {
	AvailableSkis       int `json:"availableSkis"`
	RentedSkis          int `json:"rentedSkis"`
	TotalSkis           int `json:"totalSkis"`
	RegisteredCustomers int `json:"registeredCustomers"`
}
