package main

import (
	"bmw/lib"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

// download url to file
func download(url string) (string, error) {
	glog.Infof("download url=%s", url)
	config := lib.NewReqeustConfig(nil, nil, 100, nil, nil)
	file, _, _, err := lib.HTTPGetFile(url, config)
	if err != nil {
		glog.Error(err)
		return "", err
	}

	return file, nil
}

func videoToCover(file string) (string, error) {
	glog.Infof("video=%s to cover", file)

	return runOneCmd(file, "videoToCover", func(in, out string) string {
		return fmt.Sprintf("ffmpeg -y -ss 0.1 -t 0.001 -i %s -f image2 -metadata:s:v rotate=0 -vframes 1 -vf scale=-1:400 %s", in, out)
	})

}

func thumbnailImage(file string) (string, error) {
	glog.Infof("image=%s to thumbnail", file)

	return runOneCmd(file, "thumbnailImage", func(in, out string) string {
		return fmt.Sprintf("convert %s -resize 400 %s", in, out)
	})
}

func blurImage(file string) (string, error) {
	glog.Infof("image=%s to blur", file)

	return runOneCmd(file, "blurImage", func(in, out string) string {
		return fmt.Sprintf("convert %s -blur 45x7 %s", in, out)
	})
}

func videoToWebp(file string) (string, error) {
	glog.Infof("video=%s to webp", file)

	palette, err := runOneCmd(file, "palette", func(in, out string) string {
		return fmt.Sprintf("ffmpeg -y -ss 0 -t 2 -i %s -f image2 -metadata:s:v rotate=0 -vf fps=7,scale=-1:400:flags=lanczos,palettegen %s", in, out)
	})
	if err != nil {
		glog.Error(err)
		return "", err
	}

	gif, err := runOneCmd(palette, "gif", func(in, out string) string {
		return fmt.Sprintf("ffmpeg -y -ss 0 -t 2 -v error -i %s -i %s -metadata:s:v rotate=0 -filter_complex fps=8,scale=-1:400:flags=lanczos[x];[x][1:v]paletteuse -f gif %s", file, in, out)
	})
	if err != nil {
		glog.Error(err)
		return "", err
	}

	webp, err := runOneCmd(gif, "gif", func(in, out string) string {
		return fmt.Sprintf("gif2webp -lossy %s -o %s", in, out)
	})
	if err != nil {
		glog.Error(err)
		return "", err
	}

	os.Remove(palette)
	os.Remove(gif)

	return webp, nil

}

// ===== template function =====

func runCommond(cmd string) (*exec.Cmd, error) {
	params := strings.Split(cmd, " ")

	if len(params) > 0 {
		cmd := exec.Command(params[0], params[1:]...)
		if err := cmd.Start(); err != nil {
			glog.Errorf("start command error, %s", err)
			return nil, err
		}

		if err := cmd.Wait(); err != nil {
			glog.Errorf("wait cmd to exit error, %s", err)
			return nil, err
		}

		return cmd, nil
	}

	return nil, fmt.Errorf("invalid params=%s", params)
}

type getCmdFunc func(inFile, outFile string) string

// run one cmd with one inFile and one outFile. Ignore the std pipe
func runOneCmd(inFile, prefix string, f getCmdFunc) (string, error) {
	tmpFile, err := ioutil.TempFile("", prefix)
	if err != nil {
		glog.Error(err)
		return "", err
	}

	if err = tmpFile.Close(); err != nil {
		glog.Error(err)
		return "", err
	}

	outFileName := tmpFile.Name()
	cmd := f(inFile, outFileName)
	if _, err = runCommond(cmd); err != nil {
		glog.Error(err)
		return "", err
	}

	if fileInfo, err := os.Stat(outFileName); err != nil || fileInfo.Size() == 0 {
		glog.Errorf("file create failed: %s", err)
		return "", err
	}

	return outFileName, nil
}
