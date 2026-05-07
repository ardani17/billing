package adapter

// OLTCapability mendefinisikan fitur operasional yang dapat didukung brand/model OLT.
type OLTCapability string

const (
	CapabilitySNMPSystemProbe    OLTCapability = "snmp_system_probe"
	CapabilityPONMonitoring      OLTCapability = "pon_monitoring"
	CapabilityONTList            OLTCapability = "ont_list"
	CapabilityONTSignal          OLTCapability = "ont_signal"
	CapabilitySFPMonitoring      OLTCapability = "sfp_monitoring"
	CapabilityTrafficStats       OLTCapability = "traffic_stats"
	CapabilityAlarmPolling       OLTCapability = "alarm_polling"
	CapabilityAlarmTrap          OLTCapability = "alarm_trap"
	CapabilityUnregisteredONT    OLTCapability = "unregistered_ont"
	CapabilityONTProvisioning    OLTCapability = "ont_provisioning"
	CapabilityServicePort        OLTCapability = "service_port"
	CapabilityONTReboot          OLTCapability = "ont_reboot"
	CapabilityProvisioningDryRun OLTCapability = "provisioning_dry_run"
)

// CapabilitySet menyimpan dukungan fitur per profile.
type CapabilitySet map[OLTCapability]bool

// Supports mengembalikan true jika capability tersedia.
func (s CapabilitySet) Supports(capability OLTCapability) bool {
	return s != nil && s[capability]
}

func zteC320Capabilities() CapabilitySet {
	return CapabilitySet{
		CapabilitySNMPSystemProbe: true,
		CapabilityPONMonitoring:   true,
		CapabilityONTList:         true,
		CapabilityONTSignal:       true,
		CapabilitySFPMonitoring:   true,
		CapabilityTrafficStats:    true,
		CapabilityAlarmPolling:    true,
		CapabilityAlarmTrap:       true,
		CapabilityUnregisteredONT: true,
		CapabilityONTProvisioning: true,
		CapabilityServicePort:     true,
		CapabilityONTReboot:       true,
	}
}
