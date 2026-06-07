package main

import (
	"fmt"
	"log"
	"strings"

	ipc "github.com/james-barrow/golang-ipc"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login [jagex_url]",
	Short: "Handles Jagex Launcher links for authentication",
	Long:  `Processes the Jagex Launcher link to handle authentication`,
	// Enforce that exactly 1 argument (the URL) must be passed
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rawPayload := args[0] // This will capture "jagex:code=abcdef123456"
		prefix := "jagex:"

		if strings.HasPrefix(rawPayload, prefix) {
			loginCode := strings.TrimPrefix(rawPayload, prefix)

			loginSendIPC(loginCode)
		} else {
			fmt.Printf("Error: Malformed payload received. Must start with '%s'\n", prefix)
			// Returning a non-zero status code tells the container ecosystem the execution failed
			cmd.PrintErrln("Invalid Jagex protocol token format.")
		}
	},
}

func loginSendIPC(loginCode string) {
	c, err := ipc.StartClient("goat", nil)

	if err != nil {
		log.Fatal("Unable to start IPC client")
	}

	defer c.Close()

	fmt.Println("Waiting for main instance connection...")

	for {
		message, err := c.Read()

		if err != nil {
			log.Fatal("Client error:", err)
		}

		if message.MsgType == -1 {
			if message.Status == "Reconnecting" {
				c.Close()
				return
			}

			if message.Status == "Connected" {
				fmt.Println("Sending login code")

				// Write code back to main process
				err = c.Write(1, []byte(loginCode))

				if err != nil {
					log.Fatal("Unable to write Jagex login code", err)
				}
			}
		} else if message.MsgType == 2 {
			// Acknowledged code
			fmt.Println("Code received.")
			break
		}
	}
}
