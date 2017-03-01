package main

import (
	"bufio"
	"io"
	"log"
	"os"

	"strings"

	"io/ioutil"

	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsclient "github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Storage interface {
	Append(string, io.Reader) error
}

type FsStorage struct {
	Base string
}

type S3Storage struct {
	Bucket     string
	Uploader   *s3manager.Uploader
	Downloader *s3manager.Downloader
}

func NewS3Storage(bucket string, config awsclient.ConfigProvider) *S3Storage {
	return &S3Storage{
		Bucket:     bucket,
		Downloader: s3manager.NewDownloader(config),
		Uploader:   s3manager.NewUploader(config),
	}
}

func (s *FsStorage) Append(path string, body io.Reader) error {
	writer, err := s.get(path)
	if err != nil {
		return err
	}

	buf := bufio.NewWriter(writer)

	defer buf.Flush()
	defer writer.Close()

	_, err = io.Copy(buf, body)
	if err != nil {
		return err
	}

	return nil
}

func (s *FsStorage) get(path string) (io.WriteCloser, error) {
	fullPath := s.Base + path

	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (s *S3Storage) Append(path string, body io.Reader) error {
	byteBuf, err := s.get(path)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	bodyBuf, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	byteBuf = append(byteBuf, bodyBuf...)
	buf := bytes.NewBuffer(byteBuf)

	err = s.put(path, buf)
	if err != nil {
		return err
	}

	return nil
}

func (s *S3Storage) get(path string) ([]byte, error) {
	if strings.HasPrefix(path, "/") {
		path = strings.SplitAfterN(path, "/", 2)[1]
	}

	writer := aws.NewWriteAtBuffer([]byte{})
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(path),
	}

	_, err := s.Downloader.Download(writer, input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if strings.Contains(awsErr.Error(), "status code: 404") {
				return []byte{}, nil
			}
		}

		return []byte{}, err
	}

	return writer.Bytes(), nil
}

func (s *S3Storage) put(path string, body io.Reader) error {
	if strings.HasPrefix(path, "/") {
		path = strings.SplitAfterN(path, "/", 2)[1]
	}

	input := &s3manager.UploadInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(path),
		Body:   body,
	}

	_, err := s.Uploader.Upload(input)
	if err != nil {
		return err
	}

	return nil
}
