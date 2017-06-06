package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	rq "github.com/parnurzeal/gorequest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pushCmdBranch, pushCmdDB, pushCmdEmail, pushCmdMsg, pushCmdName string

// Uploads a database to a DBHub.io cloud.
var pushCmd = &cobra.Command{
	Use:   "push [database file]",
	Short: "Upload a database",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure a database file was given
		if len(args) == 0 {
			return errors.New("No database file specified")
		}
		// TODO: Allow giving multiple database files on the command line.  Hopefully just needs turning this
		// TODO  into a for loop
		if len(args) > 1 {
			return errors.New("Only one database can be uploaded at a time (for now)")
		}

		// Ensure the database file exists
		file := args[0]
		fi, err := os.Stat(file)
		if err != nil {
			return err
		}

		// Grab author name & email from ~/.dio.yaml config file, but allow command line flags to override them
		var pushAuthor, pushEmail string
		u, ok := viper.Get("author").(string)
		if ok {
			pushAuthor = u
		}
		v, ok := viper.Get("email").(string)
		if ok {
			pushEmail = v
		}
		if pushCmdName != "" {
			pushAuthor = pushCmdName
		}
		if pushCmdEmail != "" {
			pushEmail = pushCmdEmail
		}

		// Author name and email are required
		if pushAuthor == "" || pushEmail == "" {
			return errors.New("Both author name and email are required!")
		}

		// Ensure commit message has been provided
		if pushCmdMsg == "" {
			return errors.New("Commit message is required!")
		}

		// Determine name to store database as
		if pushCmdDB == "" {
			pushCmdDB = filepath.Base(file)
		}

		// Send the file
		req := rq.New().Post(cloud+"/db_upload").
			Type("multipart").
			Set("Branch", pushCmdBranch).
			Set("Message", pushCmdMsg).
			Set("ModTime", fi.ModTime().Format(time.RFC3339)).
			Set("Database", pushCmdDB).
			Set("Author", pushAuthor).
			Set("Email", pushEmail).
			SendFile(file)
		resp, _, errs := req.End()
		if errs != nil {
			log.Print("Errors when uploading database to the cloud:")
			for _, err := range errs {
				log.Print(err.Error())
			}
			return errors.New("Error when uploading database to the cloud")
		}
		if resp != nil && resp.StatusCode != http.StatusCreated {
			return errors.New(fmt.Sprintf("Upload failed with an error: HTTP status %d - '%v'\n",
				resp.StatusCode, resp.Status))
		}
		fmt.Printf("%s - Database upload successful.  Name: %s, size: %d, branch: %s\n", cloud,
			pushCmdDB, fi.Size(), pushCmdBranch)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVar(&pushCmdBranch, "branch", "master",
		"Remote branch the database will be uploaded to")
	pushCmd.Flags().StringVar(&pushCmdEmail, "email", "", "Email address of the author")
	pushCmd.Flags().StringVar(&pushCmdMsg, "message", "",
		"(Required) Commit message for this upload")
	pushCmd.Flags().StringVar(&pushCmdName, "author", "", "Author name")
	pushCmd.Flags().StringVar(&pushCmdDB, "dbname", "", "Override for the database name")
}
