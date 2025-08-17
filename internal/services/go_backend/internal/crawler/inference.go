package crawler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

// Parses a 'DataQuery' into a structured output by passing a valid struct
func ParseLLMResponse[T any](raw string) (T, error) {
	var result T
	err := json.Unmarshal([]byte(raw), &result)
	return result, err
}

const endpoint = "https://api.groq.com/openai/v1/chat/completions"
const model = "llama-3.3-70b-versatile"

func getAPIKey() string {
	return os.Getenv("GROQ_API_KEY")
}

// Converts an empty struct into a struct's field-type
func StrucutreOutput(outputStruct any) string {
	o := reflect.TypeOf(outputStruct)
	var b strings.Builder
	b.WriteString("{\n")
	for i := 0; i < o.NumField(); i++ {
		field := o.Field(i)
		b.WriteString(fmt.Sprintf("%05d\n %v: %v,", field, field.Type))
	}
	b.WriteString("}")
	return b.String()
}

// DataQuestion inputs a query (string) and data (string),
// returns a structured output from LLM specified by the user.
// Pass 'nil' for structure for making basic queries
func DataQuery(prompt, query, data string, structure any) (any, error) {
	var output_format string
	httpClient := http.Client{Timeout: 2 * time.Second}
	if structure != nil {
		output_format = StrucutreOutput(structure)
		prompt = prompt + fmt.Sprintf("\n Return your response in the specified JSON format ONLY.\n")
	} else {
		output_format = ""
	}
	content := fmt.Sprintf("%v\n%v \n Question: %v \n Context: %v", prompt, output_format, query, data)
	userMsg := LLMMessage{Role: "user", Content: content}
	jsonMsg := LLMMessage{Role: "system", Content: "You are a strucured output generator"}
	jsonData, err := json.Marshal(LLMQuery{Model: model, Messages: []LLMMessage{userMsg, jsonMsg}})
	if err != nil {
		return "", err
	}

	//build request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Errorf("an error occured while making LLM-request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	apiKey := getAPIKey()
	req.Header.Set("Authorization", apiKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Errorf("	(DataQuery error) Error returned from LLM response: %v", err)
		return "", err
	}

	defer resp.Body.Close()
	//decode data into struct 'LLMResponse'

	body, _ := ioutil.ReadAll(resp.Body)

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Errorf("	(DataQuery error) Error returned when parsing LLM response: %v", err)
	}

	output := apiResp.Choices[0].Message.Content
	if structure == nil {
		return output, nil
	}
	if err := json.Unmarshal([]byte(output), &structure); err != nil {
		return "", err
	}

	return structure, nil

	return "", errors.New("DataQuery Error, no http client specified")
}

// Compares 2 types of entities. The user must prompt: "You are a geospatial expert".
// then a query (ex. "which raster is about soil properties?")
func CompareEntities(prompt, query, dataOne, dataTwo string) any {
	data := fmt.Sprintf("1: %v\n 2:%v")
	out, err := DataQuery(prompt, query, data, nil)
	if err != nil {
		fmt.Errorf("	(CompareEntities Error): %v", err)
	}
	return out
}
