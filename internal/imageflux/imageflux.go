package imageflux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/sohosai/whip-script/internal/core"
)

func GetEncryptionKey(urlString string, auth_token string) (string, string, error) {
	// GETリクエストを実行してデータを取得
	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/vnd.apple.mpegurl")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	// レスポンスボディを読み取って処理
	body, err := io.ReadAll(res.Body)
	if err != nil {
		core.ErrorLog(err.Error())
	}
	var indexUrl string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "http") {
			indexUrl = strings.Trim(line, "\r\n ")
			break
		}
	}
	core.Log(fmt.Sprintf(`
****************************************************
indexUrl: %v
****************************************************

`, indexUrl))

	req, err = http.NewRequest("GET", indexUrl, nil)
	if err != nil {
		core.ErrorLog(err.Error())
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/vnd.apple.mpegurl")

	client = &http.Client{}
	res, err = client.Do(req)
	if err != nil {
		core.ErrorLog(err.Error())
		return "", "", err
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		core.ErrorLog(err.Error())
		return "", "", err
	}
	dataForKid := string(body)

	// データを改行で分割してKIDを取得
	lines := strings.Split(dataForKid, "\n")
	var kid string
	for _, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-KEY") {
			rawURI := strings.Split(strings.Split(line, `URI="`)[1], `"`)[0]
			parsedURL, err := url.Parse(rawURI)
			if err != nil {
				return "", "", err
			}
			kid = parsedURL.Query().Get("kid")
			break
		}
	}

	if kid == "" {
		return "", "", nil
	}

	// POSTリクエストを実行してキーを取得
	keyRequestBody, err := json.Marshal(map[string]string{"kid": kid})
	if err != nil {
		return "", "", err
	}
	r := core.RequestPayload{
		Target:     "ImageFlux_20200707.GetEncryptKey",
		Body:       bytes.NewBuffer(keyRequestBody),
		Auth_token: auth_token,
	}

	body, err = r.ExecuteImageFluxAPI()
	if err != nil {
		return "", "", err
	}
	type EncryptKeyResponse struct {
		EncryptKey string `json:"encrypt_key"`
	}
	var keyResponse EncryptKeyResponse
	if err := json.Unmarshal(body, &keyResponse); err != nil {
		return "", "", err
	}

	return keyResponse.EncryptKey, indexUrl, nil
}
