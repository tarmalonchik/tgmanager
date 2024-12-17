package tgmanager

import (
	"encoding/json"
)

func wrapPayload(nodeName string, payload []byte) ([]byte, error) {
	data := storageData{
		NodeName: nodeName,
		Payload:  payload,
	}
	return json.Marshal(data)
}

func unwrapPayload(data []byte) (payload []byte, nodeName string, err error) {
	var dataContainer storageData

	if err = json.Unmarshal(data, &dataContainer); err != nil {
		return nil, "", err
	}
	return dataContainer.Payload, dataContainer.NodeName, nil
}
