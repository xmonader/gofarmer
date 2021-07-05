package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/tfexplorer/models/generated/phonebook"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/urfave/cli"
)

var (
	db             client.Directory
	isAlphaNumeric = regexp.MustCompile(`^[A-Za-z][0-9]+$`).MatchString
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Go Farmer!!")

	// threebotNameLabel := canvas.NewText("3Bot name", color.White)
	threebotNameInput := widget.NewEntry()
	emailInput := widget.NewEntry()

	// nameContainer := container.NewHBox(threebotNameLabel, threebotNameInput, layout.NewSpacer())
	//
	// farmNameLabel := canvas.NewText("Farm Name", color.White)
	farmNameInput := widget.NewEntry()
	// farmContainer := container.NewHBox(farmNameLabel, farmNameInput, layout.NewSpacer())

	// wordsLabel := canvas.NewText("Words", color.White)
	wordsInput := widget.NewMultiLineEntry()
	// wordsContainer := container.NewHBox(wordsLabel, wordsInput, layout.NewSpacer())

	// tftAddressLabel := canvas.NewText("tftAddress", color.White)
	tftAddressInput := widget.NewEntry()
	// tftAddressContainer := container.NewHBox(tftAddressLabel, tftAddressInput, layout.NewSpacer())

	// buttonRegister := widget.NewButton("Register Farm", func() {
	// 	log.Println("tapped")
	// })
	errorsLabel := widget.NewLabel("")

	form := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{Text: "3Bot Name", Widget: threebotNameInput, HintText: "should end with .3bot"},
			{Text: "Email", Widget: emailInput},
			{Text: "Words", Widget: wordsInput, HintText: "leave empty to generate"},
			{Text: "Farm Name", Widget: farmNameInput},

			{Text: "TFT Address", Widget: tftAddressInput},
			{Widget: errorsLabel},
		},
		SubmitText: "Register your farm",
		OnSubmit: func() { // optional, handle form submission
			log.Println(threebotNameInput.Text, emailInput.Text, farmNameInput.Text, wordsInput.Text, tftAddressInput.Text)
			errs := validateData(threebotNameInput.Text, emailInput.Text, farmNameInput.Text, tftAddressInput.Text)
			errorsLabel.Text = strings.Join(errs, "\n")

			log.Println(errs)
			log.Println("Form submitted:")
			log.Println("multiline:")
			// myWindow.Close()
		},
	}
	tabs := container.NewAppTabs(
		container.NewTabItem("Identity", widget.NewLabel("Hello")),
		container.NewTabItem("Register Farm", form),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	myWindow.SetContent(tabs)
	myWindow.ShowAndRun()
}

func validateData(name, email, farm, tftAddress string) []string {
	errs := make([]string, 0)
	if name == "" {
		errs = append(errs, "3bot name can't be empty")
	}
	if len(name) > 4 && !strings.HasSuffix(name, ".3bot") {
		errs = append(errs, "3bot name needs to have .3bot suffix")
	}
	if email == "" || !strings.Contains(email, "@") {
		errs = append(errs, "email is required and needs to be a valid string")
	}
	if !isAlphaNumeric(farm) {
		errs = append(errs, "farm needs to be alphanumeric")
	}
	return errs

}
func registerFarm(name, email, farmName, tftAddress string, tid int) error {
	addresses := make([]directory.WalletAddress, 1)
	address := directory.WalletAddress{Address: tftAddress, Asset: "TFT"}
	addresses = append(addresses, address)
	farm := directory.Farm{
		Name:            name,
		ThreebotID:      int64(tid),
		Email:           schema.Email(email),
		WalletAddresses: addresses,
	}

	farmID, err := db.FarmRegister(farm)
	if err != nil {
		return err
	}
	farm.ID = farmID
	return err
}

func generateID(c *cli.Context, url, name, email, seedPath string) (user phonebook.User, ui *tfexplorer.UserIdentity, err error) {
	ui = &tfexplorer.UserIdentity{}

	k, err := identity.GenerateKeyPair()
	if err != nil {
		return phonebook.User{}, ui, err
	}

	ui = tfexplorer.NewUserIdentity(k, 0)

	user = phonebook.User{
		Name:        name,
		Email:       email,
		Pubkey:      hex.EncodeToString(ui.Key().PublicKey),
		Description: "",
	}

	httpClient, err := client.NewClient(url, ui)
	if err != nil {
		return user, ui, err
	}

	id, err := httpClient.Phonebook.Create(user)
	if err != nil {
		return user, ui, errors.Wrap(err, "failed to register user")
	}

	// Update UserData with created id
	ui.ThreebotID = uint64(id)

	// Saving new seed struct

	if err := ui.Save(seedPath); err != nil {
		return user, ui, errors.Wrap(err, "failed to save seed")
	}

	fmt.Println("Your ID is: ", id)
	fmt.Println("Seed saved in: ", seedPath, " Please make sure you have it backed up.")
	return user, ui, nil
}
func getSeedPath() (location string, err error) {
	// Get home directory for current user
	dir, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "Cannot get current user home directory")
	}
	if dir == "" {
		return "", errors.Wrap(err, "Cannot get current user home directory")
	}
	expandedDir, err := homedir.Expand(dir)
	if err != nil {
		return "", err
	}
	os.MkdirAll(expandedDir, 0755)

	path := filepath.Join(expandedDir, ".config", "tffarmer.seed")
	return path, nil

}
