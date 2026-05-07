package adapter

import (
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

func zteBuildAddONTCommands(params domain.AddONTParams) ([]string, error) {
	if params.ONTIndex <= 0 {
		return nil, fmt.Errorf("zte ont index wajib lebih dari 0")
	}
	iface, err := zteCLIInterfaceForPort(params.PONPortIndex)
	if err != nil {
		return nil, err
	}
	return []string{
		fmt.Sprintf("interface %s", iface),
		fmt.Sprintf("onu %d type auto sn %s ont-lineprofile-id %d ont-srvprofile-id %d",
			params.ONTIndex, params.SerialNumber, params.LineProfileID, params.ServiceProfileID),
		"exit",
	}, nil
}

func zteBuildRemoveONTCommands(params domain.RemoveONTParams) ([]string, error) {
	if params.ONTIndex <= 0 {
		return nil, fmt.Errorf("zte ont index wajib lebih dari 0")
	}
	iface, err := zteCLIInterfaceForPort(params.PONPortIndex)
	if err != nil {
		return nil, err
	}
	return []string{
		fmt.Sprintf("interface %s", iface),
		fmt.Sprintf("no onu %d", params.ONTIndex),
		"exit",
	}, nil
}

func zteBuildAddServicePortCommand(params domain.AddServicePortParams) (string, error) {
	if params.ONTIndex <= 0 {
		return "", fmt.Errorf("zte ont index wajib lebih dari 0")
	}
	iface, err := zteCLIInterfaceForPort(params.PONPortIndex)
	if err != nil {
		return "", err
	}
	gemPort := params.GemPort
	if gemPort <= 0 {
		gemPort = 1
	}
	return fmt.Sprintf(
		"service-port add vlan %d %s ont %d gemport %d",
		params.VLANID, iface, params.ONTIndex, gemPort,
	), nil
}

func zteBuildRemoveServicePortCommand(params domain.RemoveServicePortParams) (string, error) {
	if params.ONTIndex <= 0 {
		return "", fmt.Errorf("zte ont index wajib lebih dari 0")
	}
	iface, err := zteCLIInterfaceForPort(params.PONPortIndex)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("no service-port vlan %d %s ont %d", params.VLANID, iface, params.ONTIndex), nil
}

func zteBuildRebootONTCommands(params domain.RebootONTParams) ([]string, error) {
	if params.ONTIndex <= 0 {
		return nil, fmt.Errorf("zte ont index wajib lebih dari 0")
	}
	iface, err := zteCLIInterfaceForPort(params.PONPortIndex)
	if err != nil {
		return nil, err
	}
	return []string{
		fmt.Sprintf("interface %s", iface),
		fmt.Sprintf("onu reset %d", params.ONTIndex),
		"exit",
	}, nil
}
