package exposed_opencode

import (
	"encoding/json"
)

// convertSSEEventToACP converts an OpenCode SSE event data string to an ACP event JSON string.
// Returns empty string if the event cannot be converted.
func convertSSEEventToACP(data string) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return ""
	}

	var eventPayload map[string]json.RawMessage
	if payload, ok := raw["payload"]; ok {
		json.Unmarshal(payload, &eventPayload)
	} else {
		eventPayload = raw
	}

	var eventType string
	json.Unmarshal(eventPayload["type"], &eventType)

	var props map[string]json.RawMessage
	if propsRaw, ok := eventPayload["properties"]; ok {
		json.Unmarshal(propsRaw, &props)
	}
	if props == nil {
		return ""
	}

	switch eventType {
	case "message.updated":
		return convertMessageUpdatedEvent(props)
	case "message.part.updated":
		return convertPartUpdatedEvent(props)
	}

	return ""
}

func convertMessageUpdatedEvent(props map[string]json.RawMessage) string {
	infoRaw, ok := props["info"]
	if !ok {
		return ""
	}
	var info struct {
		ID         string `json:"id"`
		Role       string `json:"role"`
		ModelID    string `json:"modelID,omitempty"`
		ProviderID string `json:"providerID,omitempty"`
	}
	json.Unmarshal(infoRaw, &info)

	role := info.Role
	if role == "assistant" {
		role = "agent"
	}

	acpMsg := map[string]interface{}{
		"id":    info.ID,
		"role":  role,
		"parts": []interface{}{},
	}
	if info.ModelID != "" {
		acpMsg["model"] = info.ModelID
	}

	acpEvent := map[string]interface{}{
		"type":    "acp.message.updated",
		"message": acpMsg,
	}
	result, _ := json.Marshal(acpEvent)
	return string(result)
}

func convertPartUpdatedEvent(props map[string]json.RawMessage) string {
	partRaw, ok := props["part"]
	if !ok {
		return ""
	}
	var part struct {
		ID        string          `json:"id"`
		MessageID string          `json:"messageID"`
		Type      string          `json:"type"`
		Content   string          `json:"content,omitempty"`
		Text      string          `json:"text,omitempty"`
		Tool      string          `json:"tool,omitempty"`
		Input     json.RawMessage `json:"input,omitempty"`
		Output    string          `json:"output,omitempty"`
		State     json.RawMessage `json:"state,omitempty"`
		Thinking  string          `json:"thinking,omitempty"`
		Reasoning string          `json:"reasoning,omitempty"`
	}
	json.Unmarshal(partRaw, &part)

	acpPart := convertPartToACP(part.ID, part.Type, part.Content, part.Text, part.Tool,
		part.Input, part.Output, part.State, part.Thinking, part.Reasoning)

	acpEvent := map[string]interface{}{
		"type": "acp.message.updated",
		"message": map[string]interface{}{
			"id":    part.MessageID,
			"role":  "agent",
			"parts": []interface{}{acpPart},
		},
	}
	result, _ := json.Marshal(acpEvent)
	return string(result)
}

// convertMessageToACP converts an OpenCode message JSON to ACP message format (for REST responses).
func convertMessageToACP(raw json.RawMessage) map[string]interface{} {
	var msg struct {
		Info struct {
			ID         string `json:"id"`
			Role       string `json:"role"`
			ModelID    string `json:"modelID,omitempty"`
			ProviderID string `json:"providerID,omitempty"`
		} `json:"info"`
		Parts []json.RawMessage `json:"parts"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil
	}

	role := msg.Info.Role
	if role == "assistant" {
		role = "agent"
	}

	acpParts := make([]map[string]interface{}, 0, len(msg.Parts))
	for _, partRaw := range msg.Parts {
		var part struct {
			ID        string          `json:"id"`
			Type      string          `json:"type"`
			Content   string          `json:"content,omitempty"`
			Text      string          `json:"text,omitempty"`
			Tool      string          `json:"tool,omitempty"`
			Input     json.RawMessage `json:"input,omitempty"`
			Output    string          `json:"output,omitempty"`
			State     json.RawMessage `json:"state,omitempty"`
			Thinking  string          `json:"thinking,omitempty"`
			Reasoning string          `json:"reasoning,omitempty"`
		}
		if err := json.Unmarshal(partRaw, &part); err != nil {
			continue
		}

		acpPart := convertPartToACP(part.ID, part.Type, part.Content, part.Text, part.Tool,
			part.Input, part.Output, part.State, part.Thinking, part.Reasoning)
		acpParts = append(acpParts, acpPart)
	}

	result := map[string]interface{}{
		"id":    msg.Info.ID,
		"role":  role,
		"parts": acpParts,
	}
	if msg.Info.ModelID != "" {
		result["model"] = msg.Info.ModelID
	}
	return result
}

// convertPartToACP converts an OpenCode message part to ACP format.
func convertPartToACP(id, typ, content, text, tool string,
	input json.RawMessage, output string, state json.RawMessage,
	thinking, reasoning string) map[string]interface{} {

	acpPart := map[string]interface{}{
		"id": id,
	}

	switch typ {
	case "text":
		acpPart["content_type"] = "text/plain"
		c := text
		if c == "" {
			c = content
		}
		acpPart["content"] = c
	case "tool":
		acpPart["content_type"] = "tool/call"
		acpPart["name"] = tool
		if input != nil {
			acpPart["content"] = string(input)
		}
		metadata := map[string]interface{}{}
		if state != nil {
			var stateStr string
			var stateObj map[string]interface{}
			if err := json.Unmarshal(state, &stateStr); err == nil {
				metadata["status"] = stateStr
			} else if err := json.Unmarshal(state, &stateObj); err == nil {
				for k, v := range stateObj {
					metadata[k] = v
				}
			}
		}
		if output != "" {
			metadata["output"] = output
		}
		if len(metadata) > 0 {
			acpPart["metadata"] = metadata
		}
	case "thinking", "reasoning":
		acpPart["content_type"] = "text/thinking"
		c := thinking
		if c == "" {
			c = reasoning
		}
		if c == "" {
			c = content
		}
		acpPart["content"] = c
	default:
		acpPart["content_type"] = "text/plain"
		c := text
		if c == "" {
			c = content
		}
		acpPart["content"] = c
	}

	return acpPart
}
