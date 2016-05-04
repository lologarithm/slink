package main

import (
	"io/ioutil"
	"log"
	"strings"
)

func main() {
	messages := []Message{}
	messageMap := map[string]Message{}
	// 1. Read defs.ng
	data, err := ioutil.ReadFile("defs.ng")
	if err != nil {
		log.Printf("Failed to read definition file: %s", err)
		return
	}
	// Parse types
	lines := strings.Split(string(data), "\n")

	message := Message{}
	for _, line := range lines {
		if len(line) > 6 {
			if line[:5] == "class" {
				parts := strings.Split(line, " ")
				message.Name = parts[1]
			}
		}
		if len(line) > 0 {
			if line[0] == '}' {
				messages = append(messages, message)
				messageMap[message.Name] = message
				message = Message{}
			} else if line[0] == ' ' {
				parts := strings.Split(line, " ")
				field := MessageField{
					Name:  parts[1],
					Type:  parts[2],
					Order: len(message.Fields),
				}
				switch field.Type {
				case "byte":
					field.Size = 1
				case "uint16", "int16":
					field.Size = 2
				case "uint32", "int32":
					field.Size = 4
				case "uint64", "int64":
					field.Size = 8
				case "string":
					field.Size = 4
				}
				message.SelfSize += field.Size
				message.Fields = append(message.Fields, field)
			}
		}
	}

	// 2. Write Go classes
	WriteGo(messages, messageMap)

	// 3. Generate c# classes
	WriteCS(messages, messageMap)

}

// Message is a message that can be serialized across network.
type Message struct {
	Name     string
	Fields   []MessageField
	SelfSize int
}

// MessageField is a single field of a message.
type MessageField struct {
	Name  string
	Type  string
	Order int
	Size  int
}
