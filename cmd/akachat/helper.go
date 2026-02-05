package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"
)

var (
	userA, userB, userC User
	httpClient          = &http.Client{Timeout: 10 * time.Second}
	baseURL             = "http://localhost:10011"
)

func initUser() {
	userA = User{
		ID:         "7237260465",
		PrivateKey: "HheE1MM3ciGE5hBzbfXNNeW4W4QatfAkBZgee962CWENsQrWWagNemxb8hreYnxZa2AmS1fx9MSYnbKCXGDzemV",
		Token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOiI3MjM3MjYwNDY1IiwiVXNlclR5cGUiOjEsIlBsYXRmb3JtSUQiOjAsImV4cCI6MTc3NzE4Mzk4MSwibmJmIjoxNzY5NDA3OTIxLCJpYXQiOjE3Njk0MDc5ODF9.JzRkA0grclslhxQTGVAiqKIzzfwFmg_KdHs-RluMwpM",
	}
	userB = User{
		ID:         "4138007321",
		PrivateKey: "3YMrwyXU2hNKDrUbxUUTBTr8HTSjLAiafWmGsmnUAVg8mMnH4osbPKEqiwkP2npstDA8uRzpUbDG1EZC2Pyvcur9",
		Token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOiI0MTM4MDA3MzIxIiwiVXNlclR5cGUiOjEsIlBsYXRmb3JtSUQiOjAsImV4cCI6MTc3Nzk3MTYxMCwibmJmIjoxNzcwMTk1NTUwLCJpYXQiOjE3NzAxOTU2MTB9.FpIijKiwH3EcDT_pD0G6zFi3MhIW7bV9Nsscb0BPMlA",
	}
	userC = User{
		ID:         "5834654941",
		PrivateKey: "4MbCTDNAszFXV2ZUnkPni7oQJRs7DxbyJkGvfY2YNdtJcyG8QkXuW4MET62NQBNebMRqNVuTbuew3N1BoKs2ppn",
		Token:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOiI3MjM3MjYwNDY1IiwiVXNlclR5cGUiOjEsIlBsYXRmb3JtSUQiOjAsImV4cCI6MTc3NzE4Mzk4MSwibmJmIjoxNzY5NDA3OTIxLCJpYXQiOjE3Njk0MDc5ODF9.JzRkA0grclslhxQTGVAiqKIzzfwFmg_KdHs-RluMwpM",
	}
}

func doPost[T any](url string, body any, token string) (*APIResponse[T], error) {
	rawReq, _ := json.MarshalIndent(body, "", "  ")
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(rawReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("operationID", newOperationID())
	req.Header.Set("token", token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawResp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(rawResp))
	}

	var result APIResponse[T]
	if err := json.Unmarshal(rawResp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func newOperationID() string {
	return uuid.NewString()
}
