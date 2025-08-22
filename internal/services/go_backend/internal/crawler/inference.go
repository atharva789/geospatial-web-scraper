package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
	"github.com/joho/godotenv"
)

const envFilePath = "/Users/thorbthorb/Downloads/geospatial-web-scraper/internal/services/go_backend/.env"

// Parses a 'DataQuery' into a structured output by passing a valid struct
func ParseLLMResponse[T any](raw string) (T, error) {
	var result T
	err := json.Unmarshal([]byte(raw), &result)
	return result, err
}

const endpoint = "https://api.groq.com/openai/v1/chat/completions"
const model = "llama-3.3-70b-versatile"

func getAPIKey() string {
	// load enviroment variables
	if err := godotenv.Load(envFilePath); err != nil {
		fmt.Println("Error loading GROQ API key: ", err)
	}
	apiKey := os.Getenv("GROQ_API_KEY")
	apiKey = strings.TrimSpace(apiKey)
	return apiKey
}

// Converts an empty struct into a struct's field-type
func StrucutreOutput(outputStruct any) string {
	o := reflect.TypeOf(outputStruct)
	var b strings.Builder
	b.WriteString("{\n")
	for i := 0; i < o.NumField(); i++ {
		field := o.Field(i)
		b.WriteString(fmt.Sprintf("%05d\n %v: %v,", i, field, field.Type))
	}
	b.WriteString("}")
	return b.String()
}

// DataQuestion inputs a query (string) and data (string),
// returns a structured output from LLM specified by the user.
// Pass 'nil' for structure for making basic queries
func DataQuery(prompt, query, data string, structure any) (any, error) {
	var output_format string
	httpClient := http.Client{Timeout: 5 * time.Second}
	if structure != nil {
		output_format = StrucutreOutput(structure)
		prompt = prompt + fmt.Sprintf("\n Return your response in the specified JSON format ONLY.\n")
	} else {
		output_format = ""
	}
	content := fmt.Sprintf("%v\n%v \n Question: %v \n Context: %v", prompt, output_format, query, data)
	userMsg := LLMMessage{Role: "user", Content: content}
	jsonMsg := LLMMessage{Role: "system", Content: "You are a strucured output generator"}
	jsonData, err := json.Marshal(LLMQuery{Model: model, Messages: []LLMMessage{jsonMsg, userMsg}})
	if err != nil {
		fmt.Errorf("Error converting LLMMessage to JSON. %v", err)
	}

	//build request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("an error occured while making LLM-request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	apiKey := getAPIKey()
	fmt.Println("Groq API Key: ", apiKey)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("	(DataQuery error) Error returned from LLM response: %v", err)
	}

	defer resp.Body.Close()
	//decode data into struct 'LLMResponse'

	body, err := io.ReadAll(resp.Body)	
	fmt.Printf("Body-type: %v", reflect.TypeOf(body))
	
	var apiResp GroqApiResp
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("	(DataQuery error) Error returned when parsing LLM response into struct. Response body %v, Error: %v", string(body), err)
	}
	
	if apiResp.Error != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM Error: status=%d, api_error=%s, raw=%s", resp.StatusCode, apiResp.Error, string(body))	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("GROQ API Error: no choices returned in response, raw=%s", string(body))
	}
	output := apiResp.Choices[0].Message.Content
	if structure == nil {
		return output, nil
	}

	if reflect.ValueOf(structure).Kind() != reflect.Ptr {
		return nil, fmt.Errorf("structure must be a pointer to starget type")
	}
	// remember structure should be a pointer to a struct which the user wants the data to be
	if err := json.Unmarshal([]byte(output), structure); err != nil {
		return "", err
	}

	return structure, nil
}

// Compares 2 types of entities. The user must prompt: "You are a geospatial expert".
// then a query (ex. "which raster is about soil properties?")
func CompareEntities(prompt, query, dataOne, dataTwo string) any {
	data := fmt.Sprintf("1: %v\n 2:%v", dataOne, dataTwo)
	out, err := DataQuery(prompt, query, data, nil)
	if err != nil {
		fmt.Errorf("	(CompareEntities Error): %v", err)
	}
	return out
}
