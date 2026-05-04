package domain

import "time"

type DHCPServer struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Interface     string `json:"interface"`
	AddressPool   string `json:"address_pool"`
	LeaseTime     string `json:"lease_time"`
	Authoritative string `json:"authoritative,omitempty"`
	Disabled      bool   `json:"disabled"`
	Comment       string `json:"comment,omitempty"`
}

type DHCPLease struct {
	ID           string `json:"id"`
	Server       string `json:"server,omitempty"`
	Address      string `json:"address,omitempty"`
	MACAddress   string `json:"mac_address"`
	HostName     string `json:"host_name,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	Status       string `json:"status,omitempty"`
	Dynamic      bool   `json:"dynamic"`
	Disabled     bool   `json:"disabled"`
	ExpiresAfter string `json:"expires_after,omitempty"`
	LastSeen     string `json:"last_seen,omitempty"`
	Comment      string `json:"comment,omitempty"`
	Managed      bool   `json:"managed"`
}

type DHCPNetwork struct {
	ID        string   `json:"id"`
	Address   string   `json:"address"`
	Gateway   string   `json:"gateway,omitempty"`
	DNSServer []string `json:"dns_server,omitempty"`
	Domain    string   `json:"domain,omitempty"`
	Comment   string   `json:"comment,omitempty"`
}

type DHCPBinding struct {
	ID            string
	TenantID      string
	RouterID      string
	CustomerID    string
	RouterLeaseID string
	Server        string
	MACAddress    string
	IPAddress     string
	HostName      string
	Comment       string
	Disabled      bool
	Status        string
	LastSyncAt    *time.Time
	SyncStatus    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

type DHCPBindingResponse struct {
	ID            string     `json:"id"`
	RouterID      string     `json:"router_id"`
	CustomerID    string     `json:"customer_id,omitempty"`
	RouterLeaseID string     `json:"router_lease_id,omitempty"`
	Server        string     `json:"server"`
	MACAddress    string     `json:"mac_address"`
	IPAddress     string     `json:"ip_address"`
	HostName      string     `json:"host_name,omitempty"`
	Comment       string     `json:"comment"`
	Disabled      bool       `json:"disabled"`
	Status        string     `json:"status"`
	LastSyncAt    *time.Time `json:"last_sync_at,omitempty"`
	SyncStatus    string     `json:"sync_status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (b *DHCPBinding) ToResponse() *DHCPBindingResponse {
	if b == nil {
		return nil
	}
	return &DHCPBindingResponse{
		ID:            b.ID,
		RouterID:      b.RouterID,
		CustomerID:    b.CustomerID,
		RouterLeaseID: b.RouterLeaseID,
		Server:        b.Server,
		MACAddress:    b.MACAddress,
		IPAddress:     b.IPAddress,
		HostName:      b.HostName,
		Comment:       b.Comment,
		Disabled:      b.Disabled,
		Status:        b.Status,
		LastSyncAt:    b.LastSyncAt,
		SyncStatus:    b.SyncStatus,
		CreatedAt:     b.CreatedAt,
		UpdatedAt:     b.UpdatedAt,
	}
}

type DHCPBindingListParams struct {
	RouterID string
	Page     int
	PageSize int
	Search   string
}

type DHCPBindingListResult struct {
	Data       []*DHCPBindingResponse `json:"items"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

type CreateDHCPBindingRequest struct {
	CustomerID string `json:"customer_id,omitempty"`
	Server     string `json:"server,omitempty"`
	MACAddress string `json:"mac_address" validate:"required"`
	IPAddress  string `json:"ip_address" validate:"required"`
	HostName   string `json:"host_name,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Disabled   bool   `json:"disabled"`
}

type UpdateDHCPBindingRequest struct {
	CustomerID *string `json:"customer_id,omitempty"`
	Server     *string `json:"server,omitempty"`
	MACAddress *string `json:"mac_address,omitempty"`
	IPAddress  *string `json:"ip_address,omitempty"`
	HostName   *string `json:"host_name,omitempty"`
	Comment    *string `json:"comment,omitempty"`
	Disabled   *bool   `json:"disabled,omitempty"`
}

type DeleteDHCPBindingRequest struct {
	ConfirmMAC string `json:"confirm_mac" validate:"required"`
}
