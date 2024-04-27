package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const modelRequestTimeoutSeconds = 60

type ModelClient struct {
	url string
}

func NewModelClient(url string) *ModelClient {
	client := ModelClient{
		url: url,
	}
	return &client
}

type inferRequest struct {
	Inputs []inferInput `json:"inputs"`
}

type inferInput struct {
	Name     string   `json:"name"`
	Shape    []int    `json:"shape"`
	DataType string   `json:"datatype"`
	Data     []string `json:"data"`
}

type inferResponse struct {
	ModelName    string        `json:"model_name"`
	ModelVersion string        `json:"model_version"`
	Id           string        `json:"id"`
	Outputs      []inferOutput `json:"outputs"`
}

type inferOutput struct {
	Name     string   `json:"name"`
	Shape    []int    `json:"shape"`
	DataType string   `json:"datatype"`
	Data     []string `json:"data"`
}

func (m *ModelClient) Infer(parentCtx context.Context, w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, errorJSON("streaming is not supported"), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	prompt, err := extractPrompt(r.Body)
	if err != nil {
		http.Error(w, errorJSON(err.Error()), http.StatusPreconditionFailed)
		return
	}
	if prompt == "" {
		http.Error(w, errorJSON("prompt not defined"), http.StatusPreconditionFailed)
		return
	}

	reqPayload, err := modelPayload(prompt)
	if err != nil {
		http.Error(w, errorJSON(err.Error()), http.StatusPreconditionFailed)
		return
	}

	ctx, cancel := context.WithTimeout(parentCtx, modelRequestTimeoutSeconds*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.url, bytes.NewReader(reqPayload))
	if err != nil {
		http.Error(w, errorJSON(err.Error()), http.StatusInternalServerError)
		return
	}
	type modelReturn struct {
		image string
		err   error
	}
	ch := make(chan modelReturn)
	go func() {
		image, err := m.connectToModel(req)
		ch <- modelReturn{
			image: image,
			err:   err,
		}
	}()
	ticker := time.NewTicker(time.Second * 5)

	var mr modelReturn
Loop:
	for {
		select {
		case result := <-ch:
			mr = result
			break Loop
		case <-ticker.C:
			ping(w)
			w.Write([]byte{'\n'})
			flusher.Flush()
		}
	}
	ticker.Stop()
	if mr.err != nil {
		http.Error(w, errorJSON(mr.err.Error()), http.StatusFailedDependency)
		return
	}
	respPayload := struct {
		Image string `json:"image"`
	}{
		Image: mr.image,
	}
	json.NewEncoder(w).Encode(respPayload)
}

func (m *ModelClient) connectToModel(req *http.Request) (string, error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var payload inferResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", err
	}
	if len(payload.Outputs) < 1 {
		return "", errors.New("model did not return any outputs")
	}

	output := payload.Outputs[0]
	if len(output.Data) < 1 {
		return "", errors.New("model output does not contain data field")
	}
	return output.Data[0], nil
}

func ping(w io.Writer) {
	w.Write([]byte(`{"ping":true}`))
}

func modelPayload(prompt string) ([]byte, error) {
	payload := inferRequest{
		Inputs: []inferInput{{
			Name:     "dummy",
			Shape:    []int{-1},
			DataType: "STRING",
			Data:     []string{prompt},
		}},
	}
	return json.Marshal(payload)
}

func extractPrompt(r io.Reader) (string, error) {
	payload := struct {
		Prompt string `json:"prompt"`
	}{}
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return "", fmt.Errorf("error decoding request payload: %w", err)
	}
	return payload.Prompt, nil
}

func errorJSON(s string) string {
	payload := struct {
		Err string `json:"error"`
	}{
		Err: s,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error formatting error payload: %v", err)
		return `{"error":"fatal error"}`
	}
	return string(b)
}
