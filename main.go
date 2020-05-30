package main

import (
	"github.com/vulcanize/ipfs-chain-watcher/cmd"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	cmd.Execute()
}
