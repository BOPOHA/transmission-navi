package transmission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const sessionHeader = "X-Transmission-Session-Id"

type Client struct {
	endpoint  string
	username  string
	password  string
	http      *http.Client
	sessionID string
}

type File struct {
	Name           string `json:"name"`
	Length         int64  `json:"length"`
	BytesCompleted int64  `json:"bytesCompleted"`
}

type Torrent struct {
	ID          int64   `json:"id"`
	HashString  string  `json:"hashString"`
	Name        string  `json:"name"`
	DownloadDir string  `json:"downloadDir"`
	IsFinished  bool    `json:"isFinished"`
	PercentDone float64 `json:"percentDone"`
	Status      int     `json:"status"`
	AddedDate   int64   `json:"addedDate"`
	Files       []File  `json:"files"`
}

type rpcRequest struct {
	Method    string         `json:"method"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type torrentResponse struct {
	Arguments struct {
		Torrents []Torrent `json:"torrents"`
	} `json:"arguments"`
	Result string `json:"result"`
}

func New(endpoint, username, password string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{endpoint: endpoint, username: username, password: password, http: httpClient}
}

func (c *Client) Torrents(ctx context.Context) ([]Torrent, error) {
	fields := []string{"id", "hashString", "name", "downloadDir", "isFinished", "percentDone", "status", "addedDate", "files"}
	payload, err := json.Marshal(rpcRequest{Method: "torrent-get", Arguments: map[string]any{"fields": fields}})
	if err != nil {
		return nil, fmt.Errorf("encode torrent-get request: %w", err)
	}

	var response *http.Response
	for attempt := 0; attempt < 2; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("create Transmission request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		if c.sessionID != "" {
			request.Header.Set(sessionHeader, c.sessionID)
		}
		if c.username != "" {
			request.SetBasicAuth(c.username, c.password)
		}

		response, err = c.http.Do(request)
		if err != nil {
			return nil, fmt.Errorf("call Transmission RPC: %w", err)
		}
		if response.StatusCode != http.StatusConflict {
			break
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
		c.sessionID = response.Header.Get(sessionHeader)
		if c.sessionID == "" {
			return nil, fmt.Errorf("Transmission returned 409 without %s", sessionHeader)
		}
	}

	if response == nil {
		return nil, fmt.Errorf("Transmission RPC returned no response")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("Transmission RPC returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var decoded torrentResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode Transmission response: %w", err)
	}
	if decoded.Result != "success" {
		return nil, fmt.Errorf("Transmission torrent-get failed: %s", decoded.Result)
	}
	return decoded.Arguments.Torrents, nil
}
