package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	_ "image/gif"
	_ "image/png"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)



func convertToJPEG(data []byte) ([]byte, error) {

	img, _, err := image.Decode(
		bytes.NewReader(data),
	)

	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	jpeg.Encode(&buf, img, &jpeg.Options{
		Quality: 85,
	})

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}


func sendMessage(
	client *whatsmeow.Client,
	remoteJid string,
	message string,
	imageURL string,
) error {

	if client == nil {
		return fmt.Errorf("whatsapp client is nil")
	}

	jid, err := types.ParseJID(remoteJid)

	if err != nil {
		return fmt.Errorf("invalid jid %s: %w", remoteJid, err)
	}

	ctx := context.Background()

	exists, err := client.IsOnWhatsApp(
		ctx,
		[]string{jid.User},
	)
	
	if err != nil {
		return fmt.Errorf("check whatsapp number: %w", err)
	}
	
	if len(exists) == 0 {
		return fmt.Errorf(
			"number is not registered on whatsapp: %s",
			remoteJid,
		)
	}


	if imageURL == "" {

		_, err := client.SendMessage(
			ctx,
			jid,
			getTextMessage(message),
		)

		if err != nil {
			return fmt.Errorf("send text: %w", err)
		}

		log.Printf(
			"Text message sent to %s",
			remoteJid,
		)

		return nil
	}


	imageBytes, err := downloadImage(imageURL)
	if err != nil {
		return err
	}
	
	imageBytes, err = resizeImage(imageBytes)
	if err != nil {
		return err
	}

	jpegBytes, err := convertToJPEG(imageBytes) 
	if err != nil { 
		return err 
	}
	
	err = sendImage(
		client,
		ctx,
		jid,
		message,
		jpegBytes,
		"image/jpeg",
	)

	if err != nil {
		return fmt.Errorf("send image: %w", err)
	}

	log.Printf(
		"Image message sent to %s",
		remoteJid,
	)

	return nil
}


func getTextMessage(text string) *waE2E.Message {

	return &waE2E.Message{
		Conversation: proto.String(text),
	}
}


func resizeImage(imageBytes []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, err
	}

	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	const targetWidth = 80

	targetHeight := srcH * targetWidth / srcW

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	draw.Draw(dst, dst.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	xdraw.CatmullRom.Scale(
		dst,
		dst.Bounds(),
		img,
		srcBounds,
		draw.Over,
		nil,
	)

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, dst, &jpeg.Options{
		Quality: 85,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}


func sendImage(
	cli *whatsmeow.Client,
	ctx context.Context,
	targetJID types.JID,
	caption string,
	imageBytes []byte,
	mimeType string,
) error {

	fmt.Printf("Image bytes length=%d\n", len(imageBytes))
	fmt.Printf("First 20 bytes=%v\n", imageBytes[:min(len(imageBytes), 20)])


	uploadResp, err := cli.Upload(
		ctx,
		imageBytes,
		whatsmeow.MediaImage,
	)

	if err != nil {
		return fmt.Errorf("upload image: %w", err)
	}

	imageMsg := &waE2E.ImageMessage{
		Caption:  proto.String(caption),
		Mimetype: proto.String(mimeType),

		URL:           &uploadResp.URL,
		DirectPath:    &uploadResp.DirectPath,
		MediaKey:      uploadResp.MediaKey,
		FileEncSHA256: uploadResp.FileEncSHA256,
		FileSHA256:    uploadResp.FileSHA256,
		FileLength:    &uploadResp.FileLength,
		JPEGThumbnail: imageBytes[:min(len(imageBytes), 1024)],	
		}


	msg := &waE2E.Message{ImageMessage: imageMsg}

	sendResp, err := cli.SendMessage(ctx, targetJID, msg)

	if err != nil {
		return fmt.Errorf("send image: %w", err)
	}

	log.Printf("WhatsApp message ID: %s", sendResp.ID)
	log.Printf("Server timestamp: %d", sendResp.Timestamp)

	return nil
}


func detectMime(data []byte) string {
	_, format, err := image.DecodeConfig(bytes.NewReader(data))

	if err != nil {
		return "image/jpeg"
	}

	switch format {
	case "png":
		return "image/png"
	case "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}


func downloadImage(url string) ([]byte, error) {

	if strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") {

		return downloadHTTPImage(url)
	}

	file, err := os.Open(url)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	return io.ReadAll(file)
}


func downloadHTTPImage(url string) ([]byte, error) {

	resp, err := http.Get(url)

	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"invalid image status: %d",
			resp.StatusCode,
		)
	}

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf(
			"read image: %w",
			err,
		)
	}

	return data, nil
}


func retrySend(
	client *whatsmeow.Client,
	jid types.JID,
	msg *waE2E.Message,
) error {

	for i := 0; i < 3; i++ {

		_, err := client.SendMessage(
			context.Background(),
			jid,
			msg,
		)

		if err == nil {
			return nil
		}

		log.Printf(
			"send attempt %d failed: %v",
			i+1,
			err,
		)

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf(
		"failed after retries",
	)
}