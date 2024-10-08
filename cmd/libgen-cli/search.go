// Copyright © 2019 Antoine Chiny <antoine.chiny@inria.fr>
// Copyright © 2019 Ryan Ciehanski <ryan@ciehanski.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package libgen_cli

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/yamamushi/libgen-cli/libgen"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:     "search",
	Short:   "Query all content hosted by Library Genesis.",
	Long:    `Searches for all resources that result from the provided query and then provides them for download.`,
	Example: "libgen search kubernetes",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 1 {
			if err := cmd.Help(); err != nil {
				fmt.Printf("error displaying CLI help: %v\n", err)
			}
			os.Exit(1)
		}

		// Get flags
		results, err := cmd.Flags().GetInt("results")
		if err != nil {
			fmt.Printf("error getting results flag: %v\n", err)
		}
		requireAuthor, err := cmd.Flags().GetBool("require-author")
		if err != nil {
			fmt.Printf("error getting require-author flag: %v\n", err)
		}
		extension, err := cmd.Flags().GetStringSlice("extension")
		if err != nil {
			fmt.Printf("error getting extension flag: %v\n", err)
		}
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			fmt.Printf("error getting output flag: %v\n", err)
		}
		year, err := cmd.Flags().GetInt("year")
		if err != nil {
			fmt.Printf("error getting output flag: %v\n", err)
		}
		publisher, err := cmd.Flags().GetString("publisher")
		if err != nil {
			fmt.Printf("error getting publisher flag: %v\n", err)
		}
		language, err := cmd.Flags().GetString("language")
		if err != nil {
			fmt.Printf("error getting language flag: %v\n", err)
		}
		useIpfs, err := cmd.Flags().GetBool("ipfs-mirrors")
		if err != nil {
			fmt.Printf("error getting ipfs-mirrors flag: %v\n", err)
		}
		sortBy, err := cmd.Flags().GetString("sort-by")
		if err != nil {
			fmt.Printf("error getting sort-by flag: %v\n", err)
		}
		sortASC, err := cmd.Flags().GetBool("sort-asc")
		if err != nil {
			fmt.Printf("error getting sort-asc flag: %v\n", err)
		}

		// Join args for complete search query in case
		// it contains spaces
		searchQuery := strings.Join(args, " ")
		fmt.Printf("++ Searching for: %s\n", searchQuery)

		var books []*libgen.Book
		var searchMirror = libgen.GetWorkingMirror(libgen.SearchMirrors)
		books, err = libgen.Search(&libgen.SearchOptions{
			Query:         searchQuery,
			SearchMirror:  searchMirror,
			Results:       results,
			Print:         true,
			RequireAuthor: requireAuthor,
			Extension:     extension,
			Year:          year,
			Publisher:     publisher,
			Language:      language,
			SortBy:        sortBy,
			SortASC:       sortASC,
		})
		if err != nil {
			fmt.Printf("error completing search query: %v\n", err)
			os.Exit(1)
		}
		if len(books) == 0 {
			fmt.Printf("\nNo results found from: %s.\n", searchMirror.String())
			os.Exit(1)
		}

		var pBookFormat string
		var bookSelection []string
		for _, b := range books {
			selectChoice := fmt.Sprintf("%8s ", color.New(color.FgHiBlue).Sprintf(b.ID))
			if len(b.Title) > 36 {
				pBookFormat = b.Title[:36] + "... by"
			} else {
				pBookFormat = b.Title + " by"
			}
			selectChoice += fmt.Sprintf("%s ", pBookFormat)
			if b.Author != "" {
				if len(b.Author) > 20 {
					selectChoice += fmt.Sprintf("%s ", color.New(color.FgYellow).Sprintf(b.Author[:17]+"..."))
				} else {
					selectChoice += fmt.Sprintf("%s ", color.New(color.FgYellow).Sprintf(b.Author))
				}
			} else {
				selectChoice += fmt.Sprintf("%s ", color.New(color.FgYellow).Sprintf("N/A"))
			}
			selectChoice += fmt.Sprintf("| %-4s ", color.New(color.FgRed).Sprintf(b.Extension))
			size, err := strconv.Atoi(b.Filesize)
			if err != nil {
				fmt.Printf("error converting string to int: %v\n", err)
				os.Exit(1)
			}
			selectChoice += fmt.Sprintf("| %v", color.New(color.FgGreen).Sprintf(humanize.Bytes(uint64(size))))
			bookSelection = append(bookSelection, selectChoice)
		}

		promptTemplate := &promptui.SelectTemplates{
			Active: `▸ {{ .ID | cyan | bold }}{{ if .Title }} ({{ .Title }}){{end}}`,
			//Inactive: `  {{ .Title | cyan }}{{ if .Title }} ({{ .Title }}){{end}}`,
			Selected: `{{ "✔" | green }} %s: {{ .ID | cyan }}{{ if .Title }} ({{ .Title }}){{end}}`,
		}

		prompt := promptui.Select{
			Label:     "Select Book",
			Items:     bookSelection,
			Templates: promptTemplate,
			Size:      results,
			IsVimMode: false,
			Keys: &promptui.SelectKeys{
				Next: promptui.Key{
					Code:    readline.CharNext,
					Display: "↓ (j)",
				},
				Prev: promptui.Key{
					Code:    readline.CharPrev,
					Display: "↑ (k)",
				},
				PageUp: promptui.Key{
					Code:    readline.CharForward,
					Display: "→ (l)",
				},
				PageDown: promptui.Key{
					Code:    readline.CharBackward,
					Display: "← (h)",
				},
			},
		}

		fmt.Println(strings.Repeat("-", 80))

		_, result, err := prompt.Run()
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}

		var selectedBook libgen.Book
		for i, b := range bookSelection {
			if b == result {
				selectedBook = *books[i]
				break
			}
		}

		if selectedBook.Author == "" {
			fmt.Printf("Download starting for: %s by N/A\n", selectedBook.Title)
		} else {
			fmt.Printf("Download starting for: %s by %s\n", selectedBook.Title, selectedBook.Author)
		}

		if err := libgen.GetDownloadURL(&selectedBook, useIpfs); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if useIpfs {
			if err := libgen.DownloadBookIPFS(&selectedBook, output); err != nil {
				fmt.Printf("error downloading %v: %v\n", selectedBook.Title, err)
				os.Exit(1)
			}
		} else {
			if err := libgen.DownloadBook(&selectedBook, output); err != nil {
				fmt.Printf("error downloading %v: %v\n", selectedBook.Title, err)
				os.Exit(1)
			}
		}

		if runtime.GOOS == "windows" {
			_, err = fmt.Fprintf(color.Output, "%s %s by %s.%s", color.GreenString("[OK]"),
				selectedBook.Title, selectedBook.Author, selectedBook.Extension)
			if err != nil {
				fmt.Printf("error writing to Windows os.Stdout: %v\n", err)
			}
		} else {
			fmt.Printf("%s %s by %s.%s\n", color.GreenString("[OK]"),
				selectedBook.Title, selectedBook.Author, selectedBook.Extension)
		}
	},
}

func init() {
	searchCmd.Flags().IntP("results", "r", 10, "controls how many "+
		"query results are displayed.")
	searchCmd.Flags().BoolP("require-author", "a", false, "controls "+
		"if the query results will return any media without a listed author.")
	searchCmd.Flags().StringSliceP("extension", "e", []string{""}, "controls if the query "+
		"results will return any media with a certain file extension.")
	searchCmd.Flags().StringP("output", "o", "", "where you want "+
		"libgen-cli to save your download.")
	searchCmd.Flags().IntP("year", "y", 0, "filters search query results by the "+
		"year provided.")
	searchCmd.Flags().StringP("publisher", "p", "", "filters search query "+
		"results by the publisher provided")
	searchCmd.Flags().StringP("language", "l", "", "filters search query "+
		"results by the language provided")
	searchCmd.Flags().BoolP("ipfs-mirrors", "i", false, "enforces libgen-cli to download "+
		"results via IPFS mirrors instead of HTTP(S) mirrors.")
	searchCmd.Flags().StringP("sort-by", "s", "", "sorts the queried results "+
		"by the specified string. (id, title, author, pub, year, lang, size, ext)")
	searchCmd.Flags().Bool("sort-asc", true, "sorts the queried results "+
		"by ascension or descension.")
}
