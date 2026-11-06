package domain

// Customer is a registered person with their ID document.
type Customer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IDType   string `json:"idType"`
	IDNumber string `json:"idNumber"`
}

// NewCustomer creates a customer. The store assigns the ID.
func NewCustomer(name, idType, idNumber string) (Customer, error) {
	var customer Customer
	if err := customer.Update(name, idType, idNumber); err != nil {
		return Customer{}, err
	}
	return customer, nil
}

// Update replaces the customer data. On error the customer stays unchanged.
func (c *Customer) Update(name, idType, idNumber string) error {
	name, err := required(name, "Name")
	if err != nil {
		return err
	}
	idType, err = required(idType, "Ausweistyp")
	if err != nil {
		return err
	}
	idNumber, err = required(idNumber, "Ausweisnummer")
	if err != nil {
		return err
	}
	c.Name, c.IDType, c.IDNumber = name, idType, idNumber
	return nil
}
