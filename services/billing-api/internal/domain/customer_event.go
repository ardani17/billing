package domain

// CustomerCreatedPayload adalah payload event customer.created.
type CustomerCreatedPayload struct {
	CustomerID       string `json:"customer_id"`
	Name             string `json:"name"`
	PackageID        string `json:"package_id"`
	ConnectionMethod string `json:"connection_method"`
	RouterID         string `json:"router_id,omitempty"`
}

// CustomerActivatedPayload adalah payload event customer.activated.
type CustomerActivatedPayload struct {
	CustomerID       string `json:"customer_id"`
	Name             string `json:"name"`
	PackageID        string `json:"package_id"`
	ConnectionMethod string `json:"connection_method"`
	PPPoEUsername    string `json:"pppoe_username,omitempty"`
	PPPoEPassword    string `json:"pppoe_password,omitempty"`
	RouterID         string `json:"router_id,omitempty"`
}

// CustomerIsolatedPayload adalah payload event customer.isolated.
type CustomerIsolatedPayload struct {
	CustomerID    string `json:"customer_id"`
	Name          string `json:"name"`
	RouterID      string `json:"router_id,omitempty"`
	PPPoEUsername string `json:"pppoe_username,omitempty"`
}

// CustomerUnblockedPayload adalah payload event customer.unblocked.
type CustomerUnblockedPayload struct {
	CustomerID    string `json:"customer_id"`
	Name          string `json:"name"`
	RouterID      string `json:"router_id,omitempty"`
	PPPoEUsername string `json:"pppoe_username,omitempty"`
}

// CustomerTerminatedPayload adalah payload event customer.terminated.
type CustomerTerminatedPayload struct {
	CustomerID    string `json:"customer_id"`
	Name          string `json:"name"`
	RouterID      string `json:"router_id,omitempty"`
	PPPoEUsername string `json:"pppoe_username,omitempty"`
}

// PackageChangedPayload adalah payload event package.changed.
type PackageChangedPayload struct {
	CustomerID       string `json:"customer_id"`
	OldPackageID     string `json:"old_package_id"`
	NewPackageID     string `json:"new_package_id"`
	ConnectionMethod string `json:"connection_method"`
	RouterID         string `json:"router_id,omitempty"`
}
