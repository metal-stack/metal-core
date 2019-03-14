package domain

import "git.f-i-ts.de/cloud-native/metal/metal-core/models"

func IPMIAddress(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.Address != nil {
		return *ipmi.Address
	}
	return ""
}

func IPMIInterface(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.Interface != nil {
		return *ipmi.Interface
	}
	return ""
}

func IPMIMAC(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.Mac != nil {
		return *ipmi.Mac
	}
	return ""
}

func IPMIUser(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.User != nil {
		return *ipmi.User
	}
	return ""
}

func IPMIPassword(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.Password != nil {
		return *ipmi.Password
	}
	return ""
}
