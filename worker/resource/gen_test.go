package main

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestDownload(t *testing.T) {
	url := "http://7xqgm2.com2.z0.glb.qiniucdn.com/moment/cover/snap/image/46cf3b53-6c07-4349-b764-e00b4227e632/0/0_blur"

	file, err := download(url)
	fmt.Println(file)
	if err != nil {
		t.Error(err)
	}

	fileInfo, err := os.Stat(file)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("filesize:", fileInfo.Size())

	if err := os.Remove(file); err != nil {
		t.Error(err)
	}

}

func TestVideoToCover(t *testing.T) {
	url := "http://7xqgm2.com2.z0.glb.qiniucdn.com/snap/video/c8383057-545a-42f2-b039-a9922a72154b/0/0"

	file, err := download(url)
	defer os.Remove(file)

	if err != nil {
		t.Error(err)
	}

	newFile, err := videoToCover(file)
	defer os.Remove(newFile)
	if err != nil {
		t.Error(err)
	}

	imageFile, _ := os.Open(newFile)
	defer imageFile.Close()

	image, _ := os.Create("./video_cover.jpeg")
	defer image.Close()

	io.Copy(image, imageFile)

}

func TestVideoToWebp(t *testing.T) {
	url := "http://7xqgm2.com2.z0.glb.qiniucdn.com/snap/video/c8383057-545a-42f2-b039-a9922a72154b/0/0"

	file, err := download(url)
	defer os.Remove(file)

	if err != nil {
		t.Error(err)
	}

	newFile, err := videoToWebp(file)
	defer os.Remove(newFile)
	if err != nil {
		t.Error(err)
	}

	imageFile, _ := os.Open(newFile)
	defer imageFile.Close()

	image, _ := os.Create("./video_cover.webp")
	defer image.Close()

	io.Copy(image, imageFile)

}

func TestBlurImage(t *testing.T) {
	url := "http://7xqgm2.com2.z0.glb.qiniucdn.com/snap/video/c8383057-545a-42f2-b039-a9922a72154b/0/0"

	file, err := download(url)
	defer os.Remove(file)

	if err != nil {
		t.Error(err)
	}

	newFile, err := videoToCover(file)
	defer os.Remove(newFile)
	if err != nil {
		t.Error(err)
	}

	newBlurFile, err := blurImage(newFile)
	defer os.Remove(newBlurFile)
	if err != nil {
		t.Error(err)
	}

	imageFile, _ := os.Open(newBlurFile)
	defer imageFile.Close()

	image, _ := os.Create("./video_cover_blur.jpeg")
	defer image.Close()

	io.Copy(image, imageFile)
}
