package launcher

import (
	"io"
	"os"

	"github.com/auroradevllc/apiclient"
)

const runeliteDownloadUrl = "https://github.com/runelite/launcher/releases/latest/download/RuneLite.jar"

func DownloadRuneLite(dest string) error {
	req, err := apiclient.NewRequest(runeliteDownloadUrl)

	if err != nil {
		return err
	}

	res, err := req.Send()

	if err != nil {
		return err
	}

	defer res.Body.Close()

	f, err := os.Create(dest)

	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(f, res.Body)

	return err
}
