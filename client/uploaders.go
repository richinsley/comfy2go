package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/richinsley/comfy2go/graphapi"
)

type ImageType string

const (
	InputImageType  ImageType = "input"
	TempImageType   ImageType = "temp"
	OutputImageType ImageType = "output"
)

func (c *ComfyClient) UploadFileFromReader(r io.Reader, filename string, overwrite bool, filetype ImageType, subfolder string, targetProperty *graphapi.ImageUploadProperty) (string, error) {
	// Create a buffer to store the request body
	var requestBody bytes.Buffer

	// Create a multipart writer to wrap the file (like FormData)
	writer := multipart.NewWriter(&requestBody)

	// Create a form-file for the image and copy the image data into it
	formFile, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(formFile, r)
	if err != nil {
		return "", err
	}

	// Add the overwrite field
	_ = writer.WriteField("overwrite", fmt.Sprintf("%v", overwrite))

	// Add the file type field
	_ = writer.WriteField("type", fmt.Sprintf("%v", filetype))

	// Add the subfolder field
	if subfolder != "" {
		_ = writer.WriteField("subfolder", fmt.Sprintf("%v", subfolder))
	}

	// Close the writer to finalize the body content
	writer.Close()

	// Create the request
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/upload/image", c.serverBaseAddress), &requestBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode == 200 {
		// Decode the JSON response
		var data map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return "", err
		}

		// Get the image name from the response
		name, ok := data["name"].(string)
		if !ok {
			return "", fmt.Errorf("invalid response format")
		}

		// if we were provided an ImageUploadProperty target, set the property value
		if targetProperty != nil {
			targetProperty.SetFilename(name)
		}

		// return the actual name that was chosen from the server side.  It may be different
		// from the filename we provided.  the data field also contains the given type and subfolder,
		// but we should already know that
		return name, nil
	} else {
		return "", fmt.Errorf("error: %d - %s", resp.StatusCode, resp.Status)
	}
}

func (c *ComfyClient) UploadFileFromPath(filePath string, overwrite bool, filetype ImageType, subfolder string, targetProperty *graphapi.ImageUploadProperty) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return c.UploadFileFromReader(file, filepath.Base(filePath), overwrite, filetype, subfolder, targetProperty)
}

func (c *ComfyClient) UploadImage(img image.Image, filename string, overwrite bool, filetype ImageType, subfolder string, targetProperty *graphapi.ImageUploadProperty) (string, error) {
	// Encode the image to PNG format into a bytes buffer
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return "", err
	}

	// Get the bytes from the buffer
	byteArray := buffer.Bytes()

	// Create an io.Reader from the bytes
	reader := bytes.NewReader(byteArray)
	return c.UploadFileFromReader(reader, filepath.Base(filename), overwrite, filetype, subfolder, targetProperty)
}
