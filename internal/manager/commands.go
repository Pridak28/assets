package manager

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	filepath "path/filepath"
	"strings"
	"time"

	libFile "github.com/trustwallet/assets-go-libs/file"
	"github.com/trustwallet/assets-go-libs/path"
	"github.com/trustwallet/assets-go-libs/validation/info"
	"github.com/trustwallet/assets-go-libs/validation/tokenlist"
	"github.com/trustwallet/go-primitives/asset"
	"github.com/trustwallet/go-primitives/coin"
	"github.com/trustwallet/go-primitives/types"

	"github.com/trustwallet/assets/internal/config"
)

func CreateAssetInfoJSONTemplate(token string) error {
	c, tokenID, err := asset.ParseID(token)
	if err != nil {
		return fmt.Errorf("failed to parse token id: %v", err)
	}

	chain, ok := coin.Coins[c]
	if !ok {
		return fmt.Errorf("invalid token")
	}

	assetInfoPath := path.GetAssetInfoPath(chain.Handle, tokenID)

	var emptyStr string
	var emptyInt int
	assetInfoModel := info.AssetModel{
		Name:     &emptyStr,
		Type:     &emptyStr,
		Symbol:   &emptyStr,
		Decimals: &emptyInt,
		Website:  &emptyStr,
		Explorer: &emptyStr,
		Status:   &emptyStr,
		ID:       &tokenID,
		Links: []info.Link{
			{
				Name: &emptyStr,
				URL:  &emptyStr,
			},
		},
		Tags: []string{""},
	}

	bytes, err := json.Marshal(&assetInfoModel)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %v", err)
	}

	f, err := libFile.CreateFileWithPath(assetInfoPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to write bytes to file")
	}

	err = libFile.FormatJSONFile(assetInfoPath)
	if err != nil {
		return fmt.Errorf("failed to format json file")
	}

	return nil
}

func AddTokenToTokenListJSON(chain coin.Coin, assetID, tokenID string, tokenListType path.TokenListType) error {
	setup()

	// Check for duplicates.
	tokenListTypes := []path.TokenListType{path.TokenlistDefault, path.TokenlistExtended}
	for _, t := range tokenListTypes {
		tokenListPath := path.GetTokenListPath(chain.Handle, t)
		var list tokenlist.Model

		err := libFile.ReadJSONFile(tokenListPath, &list)
		if err != nil {
			return fmt.Errorf("failed to read data from %s: %w", tokenListPath, err)
		}

		for _, item := range list.Tokens {
			if item.Asset == assetID {
				return fmt.Errorf("duplicate asset, already exist in %s", tokenListPath)
			}
		}
	}

	var list tokenlist.Model
	tokenListPath := path.GetTokenListPath(chain.Handle, tokenListType)

	err := libFile.ReadJSONFile(tokenListPath, &list)
	if err != nil {
		return fmt.Errorf("failed to read data from %s: %w", tokenListPath, err)
	}

	assetInfo, err := getAssetInfo(chain, tokenID)
	if err != nil {
		return fmt.Errorf("failed to get token info: %w", err)
	}

	newToken := tokenlist.Token{
		Asset:    assetID,
		Type:     types.TokenType(*assetInfo.Type),
		Address:  *assetInfo.ID,
		Name:     *assetInfo.Name,
		Symbol:   *assetInfo.Symbol,
		Decimals: uint(*assetInfo.Decimals),
		LogoURI:  path.GetAssetLogoURL(config.Default.URLs.AssetsApp, chain.Handle, tokenID),
	}
	list.Tokens = append(list.Tokens, newToken)

	data, err := libFile.PrepareJSONData(&tokenlist.Model{
		Name:      fmt.Sprintf("Trust Wallet: %s", coin.Coins[chain.ID].Name),
		LogoURI:   config.Default.URLs.Logo,
		Timestamp: time.Now().Format(config.Default.TimeFormat),
		Tokens:    list.Tokens,
		Version:   tokenlist.Version{Major: list.Version.Major + 1},
	})
	if err != nil {
		return err
	}

	return libFile.CreateJSONFile(tokenListPath, data)
}

func getAssetInfo(chain coin.Coin, tokenID string) (*info.AssetModel, error) {
	path := path.GetAssetInfoPath(chain.Handle, tokenID)
	var assetModel info.AssetModel

	err := libFile.ReadJSONFile(path, &assetModel)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from info.json: %w", err)
	}

	return &assetModel, nil
}

// UploadDocument uploads a document file to an asset directory
func UploadDocument(assetID, documentPath string) error {
	// Parse asset ID
	c, tokenID, err := asset.ParseID(assetID)
	if err != nil {
		return fmt.Errorf("failed to parse asset id: %v", err)
	}

	chain, ok := coin.Coins[c]
	if !ok {
		return fmt.Errorf("unsupported blockchain: %d", c)
	}

	// Validate document file exists
	if _, err := os.Stat(documentPath); os.IsNotExist(err) {
		return fmt.Errorf("document file does not exist: %s", documentPath)
	}

	// Get asset directory path
	assetDir := filepath.Join("blockchains", chain.Handle, "assets", tokenID)

	// Check if asset directory exists
	if _, err := os.Stat(assetDir); os.IsNotExist(err) {
		return fmt.Errorf("asset directory does not exist: %s. Create the asset first using add-token command", assetDir)
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(documentPath))
	allowedExtensions := map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".txt":  true,
		".md":   true,
	}

	if !allowedExtensions[ext] {
		return fmt.Errorf("unsupported file extension: %s. Supported extensions: .pdf, .doc, .docx, .txt, .md", ext)
	}

	// Get filename from path
	filename := filepath.Base(documentPath)

	// Create destination path
	destPath := filepath.Join(assetDir, filename)

	// Check if file already exists
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("document file already exists: %s", destPath)
	}

	// Copy file
	sourceFile, err := os.Open(documentPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		// Clean up incomplete destination file on copy failure
		os.Remove(destPath)
		return fmt.Errorf("failed to copy file: %v", err)
	}

	fmt.Printf("Successfully uploaded document to: %s\n", destPath)
	return nil
}
