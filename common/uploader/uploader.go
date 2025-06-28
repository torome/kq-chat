package uploader

import (
	"context"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Uploader interface {
	Upload(ctx context.Context, file io.Reader, filename string) (url string, savePath string, err error)
	GetFullURL(path string) string
	GetCoverFullURL(path string) string
}

type uploaderImpl struct {
	cfg       config.UploadConf
	ossClient *oss.Client
	bucket    *oss.Bucket
}

func NewUploader(cfg config.UploadConf) Uploader {
	if cfg.Storage == "oss" {
		client, err := oss.New(cfg.OSS.Endpoint, cfg.OSS.AccessKeyId, cfg.OSS.AccessKeySecret)
		if err != nil {
			panic(err)
		}
		bucket, err := client.Bucket(cfg.OSS.BucketName)
		if err != nil {
			panic(err)
		}
		return &uploaderImpl{cfg: cfg, ossClient: client, bucket: bucket}
	}
	return &uploaderImpl{cfg: cfg}
}

func (u *uploaderImpl) Upload(ctx context.Context, file io.Reader, filename string) (string, string, error) {
	savePath := "upload/" + filename

	if u.cfg.Storage == "oss" {
		err := u.bucket.PutObject(savePath, file)
		if err != nil {
			return "", "", err
		}
		return u.cfg.OSS.BaseURL + "/" + savePath, savePath, nil
	}

	localPath := filepath.Join(u.cfg.LocalPath, savePath)
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}
	out, err := os.Create(localPath)
	if err != nil {
		return "", "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		return "", "", err
	}
	return "/" + savePath, savePath, nil
}

func (u *uploaderImpl) GetFullURL(path string) string {
	if path == "" {
		return "https://file.qkaifa.com/avatar.png"
	}
	switch strings.ToLower(u.cfg.Storage) {
	case "oss":
		return u.cfg.OSS.BaseURL + path
	case "local":
		return u.cfg.LocalBaseURL + path
	default:
		return path
	}
}

func (u *uploaderImpl) GetCoverFullURL(path string) string {
	if path == "" {
		return "https://file.qkaifa.com/bg2.jpeg"
	}
	switch strings.ToLower(u.cfg.Storage) {
	case "oss":
		return u.cfg.OSS.BaseURL + path
	case "local":
		return u.cfg.LocalBaseURL + path
	default:
		return path
	}
}
