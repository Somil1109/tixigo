package media

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

type Cloudinary struct {
	cloudName, apiKey, apiSecret string
	client                       *http.Client
}

func NewCloudinary(cloudName, key, secret string) *Cloudinary {
	return &Cloudinary{cloudName, key, secret, http.DefaultClient}
}
func (c *Cloudinary) UploadPoster(ctx context.Context, filename string, file io.Reader) (string, error) {
	if c.cloudName == "" || c.apiKey == "" || c.apiSecret == "" {
		return "", errors.New("Cloudinary is not configured")
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sum := sha1.Sum([]byte("folder=tixigo/posters&timestamp=" + timestamp + c.apiSecret))
	signature := hex.EncodeToString(sum[:])
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(part, file); err != nil {
		return "", err
	}
	for key, value := range map[string]string{"api_key": c.apiKey, "timestamp": timestamp, "folder": "tixigo/posters", "signature": signature} {
		_ = writer.WriteField(key, value)
	}
	_ = writer.Close()
	endpoint := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", c.cloudName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("Cloudinary returned %s", res.Status)
	}
	var out struct {
		URL string `json:"secure_url"`
	}
	if json.NewDecoder(res.Body).Decode(&out) != nil || out.URL == "" {
		return "", errors.New("invalid Cloudinary response")
	}
	return out.URL, nil
}
