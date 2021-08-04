package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
)

var (
	isAlphaNumeric = regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString
	explorersNames = []string{"Mainnet", "Testnet", "Devnet"}
	explorersUrls  = map[string]string{"Mainnet": "https://explorer.grid.tf", "Testnet": "https://explorer.testnet.grid.tf", "Devnet": "https://explorer.devnet.grid.tf"}
	// SeedVersion1 (binary seed)
	SeedVersion1 = MustParse("1.0.0")
	// SeedVersion11 (json mnemonic)
	SeedVersion11 = MustParse("1.1.0")
	// SeedVersionLatest link to latest seed version
	SeedVersionLatest     = SeedVersion11
	threebotId        int = 0
	userid                = &UserIdentity{}
)

func main() {
	var expclient *Client

	myApp := app.New()
	myWindow := myApp.NewWindow("Go Farmer!!")
	explorerUrl, _ := explorersUrls["Mainnet"]

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
	errorsIdentityLabel := widget.NewLabel("")
	errorsFarmLabel := widget.NewLabel("")
	infoIdentityLabel := widget.NewLabel("")
	infoFarmLabel := widget.NewLabel("")
	// farmsLabel := widget.NewLabel("")
	errorsFarmTableLabel := widget.NewLabel("")

	formIdentity := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{Text: "3Bot Name", Widget: threebotNameInput, HintText: "should end with .3bot"},
			{Text: "Email", Widget: emailInput},
			{Text: "Words", Widget: wordsInput, HintText: "leave empty to generate"},
			{Widget: infoIdentityLabel},
			{Widget: errorsIdentityLabel},
		},
		SubmitText: "Register your identity",
		OnSubmit: func() { // optional, handle form submission
			log.Println(threebotNameInput.Text, emailInput.Text, farmNameInput.Text, wordsInput.Text, tftAddressInput.Text)
			errs := validateIdentityData(threebotNameInput.Text, emailInput.Text, wordsInput.Text)
			errorsIdentityLabel.Text = strings.Join(errs, "\n")
			if len(errs) == 0 {
				seedpath, err := getSeedPath()
				if err != nil {
					println(err)
					os.Exit(1)
				}
				doGen := func() {
					_, ui, err := generateID(explorerUrl, threebotNameInput.Text, emailInput.Text, seedpath, wordsInput.Text)
					if err != nil {
						fmt.Println(err)
						fmt.Println(ui)
						errorsIdentityLabel.Text = fmt.Sprintf("Error while generating identity %s", err)
						dialog.ShowError(fmt.Errorf(errorsIdentityLabel.Text), myWindow)

					} else {
						infoIdentityLabel.Text = fmt.Sprintf("your 3Bot ID is %d: and seed is saved at %s", ui.ThreebotID, seedpath)
						fmt.Println("menoms: ", ui.Mnemonic)
						wordsInput.SetText(ui.Mnemonic)
						dialog.ShowInformation("Success", infoIdentityLabel.Text, myWindow)
						threebotId = int(ui.ThreebotID)
						expclient, err = NewClient(explorerUrl, ui)
						if err != nil {
							fmt.Println("failed to get explorer client: ", err)
							dialog.ShowError(fmt.Errorf("failed to get explorer client"), myWindow)
						}
					}
				}
				errorsIdentityLabel.Text = ""
				if _, err = os.Stat(seedpath); !os.IsNotExist(err) {
					dialog.ShowConfirm("Overwriting your 3Bot Identity", "Are you sure you want to  overwrite the existing identity? Make sure to backup your seed file.?\n\n", func(b bool) {
						if b {

							doGen()
						}

					}, myWindow)

				} else {
					doGen()

				}

			}

			log.Println(errs)

		},
	}

	seedpath, err := getSeedPath()
	if err != nil {
		println(err)
		os.Exit(1)
	}
	if _, err = os.Stat(seedpath); !os.IsNotExist(err) {
		userid.Load(seedpath)
		threebotId = int(userid.ThreebotID)
		if expclient, err = NewClient(explorerUrl, userid); err == nil {
			if u, err := expclient.Phonebook.Get(userid.ThreebotID); err == nil {
				wordsInput.Text = userid.Mnemonic
				emailInput.Text = u.Email
				threebotNameInput.Text = u.Name
			} else {

				fmt.Println("failed to get explorer client: ", err)
			}

		}

	}

	formFarm := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{Text: "Farm Name", Widget: farmNameInput},
			{Text: "TFT Address", Widget: tftAddressInput, HintText: "valid TFT address (56 characters)"},
			{Widget: infoFarmLabel},
			{Widget: errorsFarmLabel},
		},
		SubmitText: "Register your farm",
		OnSubmit: func() { // optional, handle form submission
			log.Println(threebotNameInput.Text, emailInput.Text, farmNameInput.Text, wordsInput.Text, tftAddressInput.Text)
			errs := validateData(threebotNameInput.Text, emailInput.Text, farmNameInput.Text, tftAddressInput.Text)
			errorsFarmLabel.Text = strings.Join(errs, "\n")
			if len(errs) == 0 && threebotId > 0 {
				if farm, err := registerFarm(expclient, farmNameInput.Text, emailInput.Text, tftAddressInput.Text, threebotId); err == nil {

					infoFarmLabel.Text = fmt.Sprintf("farm with ID %d is created", farm.ID)
					dialog.ShowInformation("Farm Registered!", infoFarmLabel.Text, myWindow)
				} else {
					errorsFarmLabel.Text = fmt.Sprintf("Error while registering farm %s", err)
					dialog.ShowError(fmt.Errorf(errorsFarmLabel.Text), myWindow)
				}
				log.Println(errs)
				// log.Println("Form submitted:")
				// log.Println("multiline:")
				// myWindow.Close()
			}
		},
	}

	// // Simple listing/printing  of the user's farms
	// formListFarms := &widget.Form{
	// 	Items: []*widget.FormItem{ // we can specify items in the constructor
	// 		{Widget: farmsLabel},
	// 		{Widget: errorsFarmLabel},
	// 	},
	// 	SubmitText: "List Farms",
	// 	OnSubmit: func() { // optional, handle form submission
	// 		if threebotId > 0 {
	// 			// listFarms(expclient *Client) (farmsRet []Farm, err error)
	// 			if _, farmsString, err := listFarms(expclient, threebotId); err == nil {
	// 				farmsLabel.Text = farmsString
	// 			} else {
	// 				errorsFarmLabel.Text = fmt.Sprintf("Error %s", err)
	// 				dialog.ShowError(fmt.Errorf(errorsFarmLabel.Text), myWindow)
	// 			}
	// 		}
	// 	},
	// }

	formFarms := &widget.Form{
		Items: []*widget.FormItem{
			{Widget: errorsFarmTableLabel},
		},
		SubmitText: "Reload Farms",
		OnSubmit: func() {

			myFarms := []string{}
			if farmsRet, _, err := listFarms(expclient, threebotId); err == nil {
				if len(farmsRet) > 0 {
					for _, f := range farmsRet {
						myFarms = append(myFarms, []string{f.Email, f.Name}...)
						// fmt.Println(f.Name)
						// fmt.Println("-------------------------------------------")
					}
				}
			} else {
				errorsFarmLabel.Text = fmt.Sprintf("Error %s", err)
				dialog.ShowError(fmt.Errorf(errorsFarmLabel.Text), myWindow)
			}
			// TODO update farmTable added to the form with the new myFarms

			// var data = [][]string{[]string{"top left", "top right"},
			// 	[]string{"bottom left", "bottom right"}}
			// var data = [][]string{[]string{"Farm Email", "Farm name"}, myFarms}
			// farmTable := widget.NewTable(
			// 	func() (int, int) {
			// 		return len(data), len(data[0])
			// 	},
			// 	func() fyne.CanvasObject {
			// 		return widget.NewLabel("wide content")
			// 	},
			// 	func(i widget.TableCellID, o fyne.CanvasObject) {
			// 		o.(*widget.Label).SetText(data[i.Row][i.Col])
			// 	})

		},
	}

	if expclient != nil {
		myFarms := []string{}
		if farmsRet, _, err := listFarms(expclient, threebotId); err == nil {
			if len(farmsRet) > 0 {
				for _, f := range farmsRet {
					myFarms = append(myFarms, []string{f.Email, f.Name}...)
					// fmt.Println(myFarms)
					// fmt.Println("-------------------------------------------")
				}
			}
		} else {
			errorsFarmLabel.Text = fmt.Sprintf("Error %s", err)
			dialog.ShowError(fmt.Errorf(errorsFarmLabel.Text), myWindow)
		}

		// var data = [][]string{{"top left", "top right"},
		// 	{"bottom left", "bottom right"}}
		var data = [][]string{{"Farm Email", "Farm name"}, myFarms}
		farmTable := widget.NewTable(
			func() (int, int) {
				return len(data), len(data[0])
			},
			func() fyne.CanvasObject {
				item := widget.NewLabel("wide content")
				item.Resize(fyne.Size{
					Width:  500,
					Height: 30,
				})

				return item
			},
			func(i widget.TableCellID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(data[i.Row][i.Col])
				o.(*widget.Label).Resize(fyne.Size{
					Width:  500,
					Height: 30,
				})
			})

		formFarms.AppendItem(&widget.FormItem{Widget: widget.NewCard("Farms", "", farmTable)})

	}

	tabs := container.NewAppTabs(
		container.NewTabItem("Identity", formIdentity),
		container.NewTabItem("Register Farm", formFarm),
		// container.NewTabItem("Farms", formListFarms),
		container.NewTabItem("Farms", formFarms),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	myWindow.SetContent(tabs)
	myWindow.Resize(fyne.NewSize(600, 300))

	myWindow.ShowAndRun()
}

func validateIdentityData(name, email, words string) []string {
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
	if words != "" {
		ui := UserIdentity{}
		if err := ui.FromMnemonic(words); err != nil {
			errs = append(errs, "words are invalid")

		}
	}
	fmt.Println("validation errs: ", errs)
	return errs

}
func validateData(name, email, farm, tftAddress string) []string {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	tftAddress = strings.TrimSpace(tftAddress)
	farm = strings.TrimSpace(farm)
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
	if _, err := mail.ParseAddress(email); err != nil {
		errs = append(errs, err.Error())
	}
	if !isAlphaNumeric(farm) {
		errs = append(errs, "farm needs to be alphanumeric")
	}

	if len(tftAddress) != 56 {
		errs = append(errs, "invalid tft wallet address")
	}
	return errs

}
func registerFarm(expclient *Client, name, email, tftAddress string, tid int) (Farm, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	tftAddress = strings.TrimSpace(tftAddress)
	addresses := make([]WalletAddress, 1)
	address := WalletAddress{Address: tftAddress, Asset: "TFT"}
	addresses = append(addresses, address)
	farm := Farm{
		Name:            name,
		ThreebotID:      int64(tid),
		Email:           email,
		WalletAddresses: addresses,
	}

	farmID, err := expclient.Directory.FarmRegister(farm)
	if err != nil {
		fmt.Println("err:", err)
		return farm, err
	}
	farm.ID = farmID
	fmt.Println("registered farm: ", farm)
	return farm, nil
}

func formatFarm(farm Farm) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, ("-------------------------------------------"))
	fmt.Fprintf(b, "ID: %d\n", farm.ID)
	fmt.Fprintf(b, "Name: %s\n", farm.Name)
	fmt.Fprintf(b, "Email: %s\n", farm.Email)
	fmt.Fprintf(b, "Farmer TheebotID: %d\n", farm.ThreebotID)
	fmt.Fprintf(b, "Wallet addresses:\n")
	for _, a := range farm.WalletAddresses {
		fmt.Fprintf(b, "  %s:%s\n", a.Asset, a.Address)
	}
	fmt.Fprintf(b, "IP addresses:\n")
	for _, a := range farm.IPAddresses {
		fmt.Fprintf(b, "  IP-> %s   Gateway-> %s\n", a.Address, a.Gateway)
	}
	return b.String()
}

func listFarms(expclient *Client, tid int) (farmsRet []Farm, farmsFormatted string, err error) {
	farmsRet = make([]Farm, 0)
	pageNumber := 1

	for {
		pager := Page(pageNumber, 20)
		farms, err := expclient.Directory.FarmList(int64(tid), "", pager)
		farmsRet = append(farmsRet, farms...)
		if err != nil {
			break
		}
		if len(farms) == 0 {
			break
		}
		pageNumber++
	}
	if len(farmsRet) > 0 {
		for _, f := range farmsRet {
			farmsFormatted = farmsFormatted + formatFarm(f)

		}
	} else {

		fmt.Println("No farms found")
	}
	return farmsRet, farmsFormatted, nil
}

func generateID(url, name, email, seedPath, words string) (user User, ui *UserIdentity, err error) {
	fmt.Println("generating against ", words, seedPath)
	ui = &UserIdentity{}
	if words != "" {
		err := ui.FromMnemonic(words)
		if err != nil {
			return user, ui, err
		}

	} else {
		// check if have the seed path already
		if _, err = os.Stat(seedPath); !os.IsNotExist(err) {
			err = ui.Load(seedPath)
			if err != nil {
				return User{}, ui, err
			}
		} else {
			// no words and no seedpath, generate new
			k, err := GenerateKeyPair()
			if err != nil {
				return User{}, ui, err
			}

			ui = NewUserIdentity(k, 0)
		}

	}

	user = User{
		Name:        name,
		Email:       email,
		Pubkey:      hex.EncodeToString(ui.Key().PublicKey),
		Description: "",
	}

	httpClient, err := NewClient(url, ui)

	if err != nil {
		return user, ui, err
	}
	// if current UI name, email and pubkey match the one in the explorer, then we don't need to register
	eluser, elerr := httpClient.Phonebook.GetUserByNameOrEmail(name, email)
	if elerr == nil {
		// user exists already now we check against the publick key
		if eluser.Pubkey == hex.EncodeToString(ui.Key().PublicKey) {
			fmt.Println("user exists and matches explorer registered user pubkey")

			user.ID = eluser.ID
			ui.ThreebotID = int64(user.ID)
			return user, ui, nil
		} else {
			return user, ui, fmt.Errorf("user already exists and its public key doesn't match the one on explorer")
		}

	}

	id, err := httpClient.Phonebook.Create(user)
	if err != nil {
		fmt.Println(err)
		return user, ui, errors.Wrap(err, "failed to register user")
	}

	// Update UserData with created id
	ui.ThreebotID = int64(id)

	// Saving new seed struct

	os.Chmod(seedPath, 0755)
	if err := ui.Save(seedPath); err != nil {
		return user, ui, errors.Wrap(err, "failed to save seed")
	} else {
		fmt.Println("errr: ", err)
	}

	fmt.Println("Your ID is: ", ui.ThreebotID)
	fmt.Println("Seed saved in: ", seedPath, " Please make sure you have it backed up.")
	return user, ui, nil
}

func getSeedPath() (location string, err error) {
	// Get home directory for current user

	configdDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	// os.MkdirAll(expandedDir, 0755)

	path := filepath.Join(configdDir, "tffarmer.seed")
	return path, nil

}

// LoadSeed from path
func LoadSeedData(path string) (string, int, error) {
	version, seed, err := ReadFile(path)

	if version.EQ(SeedVersion11) {
		// it means we read json data instead of the secret
		type Seed110Struct struct {
			Mnemonics  string `json:"mnemonic"`
			ThreebotID int    `json:"threebotid"`
		}
		var seed110 Seed110Struct
		if err = json.Unmarshal(seed, &seed110); err != nil {
			return "", 0, err
		}
		return seed110.Mnemonics, seed110.ThreebotID, nil
	}
	return "", 0, fmt.Errorf("couldn't get mnemonics")
}
