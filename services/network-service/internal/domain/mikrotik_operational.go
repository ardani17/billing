package domain

type RouterInterface struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	MTU      int    `json:"mtu"`
	MAC      string `json:"mac_address,omitempty"`
	Running  bool   `json:"running"`
	Disabled bool   `json:"disabled"`
	RXByte   int64  `json:"rx_byte"`
	TXByte   int64  `json:"tx_byte"`
	RXPacket int64  `json:"rx_packet"`
	TXPacket int64  `json:"tx_packet"`
	Comment  string `json:"comment,omitempty"`
}

type RouterTrafficSample struct {
	Interface string `json:"interface"`
	RXBps     int64  `json:"rx_bps"`
	TXBps     int64  `json:"tx_bps"`
	RXPackets int64  `json:"rx_packets_per_second"`
	TXPackets int64  `json:"tx_packets_per_second"`
}

type RouterIPPoolUsage struct {
	Name         string   `json:"name"`
	Ranges       []string `json:"ranges"`
	Used         int      `json:"used"`
	Total        int      `json:"total"`
	Available    int      `json:"available"`
	UsagePercent int      `json:"usage_percent"`
	WarningLevel string   `json:"warning_level"`
}

type RouterFirewallRule struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Chain    string `json:"chain,omitempty"`
	Action   string `json:"action,omitempty"`
	List     string `json:"list,omitempty"`
	Address  string `json:"address,omitempty"`
	Disabled bool   `json:"disabled"`
	Comment  string `json:"comment,omitempty"`
}

type RouterLogEntry struct {
	ID      string `json:"id"`
	Time    string `json:"time"`
	Topics  string `json:"topics"`
	Message string `json:"message"`
}

type RouterLogFilter struct {
	Topic  string
	Search string
	Limit  int
}
