package api

func (c *apiClient) SetMachineLEDStateOn(machineID string) error {
	err := c.SetMachineLEDStateOn(machineID)
	if err != nil {
		return err
	}
	return nil
}

func (c *apiClient) SetMachineLEDStateOff(machineID string) error {
	err := c.SetMachineLEDStateOff(machineID)
	if err != nil {
		return err
	}
	return nil
}
