package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"order-service/internal/domain"
)

type paymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type paymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
}

type PaymentHTTPClient struct {
	client  *http.Client
	baseURL string
}

func NewPaymentHTTPClient(client *http.Client, baseURL string) domain.PaymentClient {
	return &PaymentHTTPClient{
		client:  client,
		baseURL: baseURL,
	}
}

func (c *PaymentHTTPClient) Authorize(orderID string, amount int64) (*domain.PaymentResult, error) {
	reqBody := paymentRequest{
		OrderID: orderID,
		Amount:  amount,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("paymentClient: failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(
		c.baseURL+"/payments",
		"application/json",
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("paymentClient: HTTP call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, errors.New("paymentClient: payment service returned 503")
	}

	var payResp paymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&payResp); err != nil {
		return nil, fmt.Errorf("paymentClient: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("paymentClient: unexpected status %d: %s", resp.StatusCode, payResp.Message)
	}

	return &domain.PaymentResult{
		TransactionID: payResp.TransactionID,
		Status:        payResp.Status,
	}, nil
}
