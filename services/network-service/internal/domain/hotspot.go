package domain

type HotspotUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Password    string `json:"password,omitempty"`
	Profile     string `json:"profile,omitempty"`
	LimitUptime string `json:"limit_uptime,omitempty"`
	Uptime      string `json:"uptime,omitempty"`
	BytesIn     int64  `json:"bytes_in"`
	BytesOut    int64  `json:"bytes_out"`
	Disabled    bool   `json:"disabled"`
	Comment     string `json:"comment,omitempty"`
	Managed     bool   `json:"managed"`
}

type HotspotProfile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RateLimit   string `json:"rate_limit,omitempty"`
	SharedUsers string `json:"shared_users,omitempty"`
	AddressPool string `json:"address_pool,omitempty"`
	Transparent bool   `json:"transparent_proxy"`
	Comment     string `json:"comment,omitempty"`
}

type HotspotActiveSession struct {
	ID         string `json:"id"`
	User       string `json:"user"`
	Address    string `json:"address,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
	Uptime     string `json:"uptime,omitempty"`
	BytesIn    int64  `json:"bytes_in"`
	BytesOut   int64  `json:"bytes_out"`
	Server     string `json:"server,omitempty"`
}

type CreateHotspotUserRequest struct {
	Name        string `json:"name"`
	Password    string `json:"password"`
	Profile     string `json:"profile"`
	LimitUptime string `json:"limit_uptime"`
	Comment     string `json:"comment"`
}

type UpdateHotspotUserRequest struct {
	Password    *string `json:"password,omitempty"`
	Profile     *string `json:"profile,omitempty"`
	LimitUptime *string `json:"limit_uptime,omitempty"`
	Disabled    *bool   `json:"disabled,omitempty"`
	Comment     *string `json:"comment,omitempty"`
}

type HotspotLoginTemplateRequest struct {
	BrandName    string `json:"brand_name"`
	PrimaryColor string `json:"primary_color"`
	SupportPhone string `json:"support_phone"`
	Message      string `json:"message"`
}

type HotspotLoginTemplate struct {
	FileName string `json:"file_name"`
	HTML     string `json:"html"`
}
