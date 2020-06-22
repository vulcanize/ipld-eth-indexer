package main

import (
	"github.com/vulcanize/ipfs-blockchain-watcher/cmd"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	cmd.Execute()
}
