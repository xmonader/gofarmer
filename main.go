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
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
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

	threebotIdInput := widget.NewEntry()
	threebotIdInput.Disable()
	threebotNameInput := widget.NewEntry()
	emailInput := widget.NewEntry()
	wordsInput := widget.NewMultiLineEntry()
	infoIdentityLabel := widget.NewLabel("")
	errorsIdentityLabel := widget.NewLabel("")

	farmNameInput := widget.NewEntry()
	tftAddressInput := widget.NewEntry()
	errorsFarmLabel := widget.NewLabel("")
	infoFarmLabel := widget.NewLabel("")

	farmOwnerIdEntry := widget.NewEntry()
	farmIdEntryUpdate := widget.NewEntry()
	farmIdEntryUpdate.Disable()
	farmNameInputUpdate := widget.NewEntry()
	tftAddressInputUpdate := widget.NewEntry()
	errorsFarmLabelUpdate := widget.NewLabel("")
	infoFarmLabelUpdate := widget.NewLabel("")

	nodeId := widget.NewEntry()
	nodeVersion := widget.NewEntry()
	nodeHostName := widget.NewEntry()
	nodeFarmerName := widget.NewEntry()
	nodeLocation := widget.NewEntry()
	nodeUptime := widget.NewEntry()
	nodeHRU := widget.NewEntry()
	nodeSRU := widget.NewEntry()
	nodeCRU := widget.NewEntry()
	nodeMRU := widget.NewEntry()

	farmsListData := make([]Farm, 0)
	farmsNames := make([]string, 0)
	farmsBinding := binding.BindStringList(&farmsNames)

	nodesListData := make([]Node, 0)
	nodesNames := make([]string, 0)
	nodesBinding := binding.BindStringList(&nodesNames)

	seedpath, err := getSeedPath()
	fmt.Println(seedpath)
	if err != nil {
		println(err)
		os.Exit(1)
	}
	if _, err = os.Stat(seedpath); !os.IsNotExist(err) {
		userid.Load(seedpath)
		threebotId = int(userid.ThreebotID)
		threebotIdInput.SetText(fmt.Sprintf("%d", threebotId))
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

	var farmToEditIdx int64 = 0

	formIdentity := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{Text: "3Bot ID", Widget: threebotIdInput, HintText: "3Bot ID"},
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
					farmsListData, farmsNames = ListAllFarmsAndNames(expclient, int64(threebotId))
					farmsBinding.Set(farmsNames)
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

	formFarmUpdate := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{Text: "Owner ID", Widget: farmOwnerIdEntry, HintText: "Change to transfer farm ownership"},
			{Text: "Farm ID", Widget: farmIdEntryUpdate},
			{Text: "Farm Name", Widget: farmNameInputUpdate},
			{Text: "TFT Address", Widget: tftAddressInputUpdate, HintText: "valid TFT address (56 characters)"},
			{Widget: infoFarmLabelUpdate},
			{Widget: errorsFarmLabelUpdate},
		},
		SubmitText: "Edit your farm",
		OnSubmit: func() { // optional, handle form submission
			log.Println(threebotNameInput.Text, emailInput.Text, farmNameInputUpdate.Text, wordsInput.Text, tftAddressInputUpdate.Text)
			errs := validateData(threebotNameInput.Text, emailInput.Text, farmNameInputUpdate.Text, tftAddressInputUpdate.Text)
			errorsFarmLabelUpdate.Text = strings.Join(errs, "\n")
			if len(errs) == 0 && threebotId > 0 {
				farmOwnerIDAsInt, err := strconv.Atoi(farmOwnerIdEntry.Text)
				if err != nil {
					errorsFarmLabelUpdate.Text = fmt.Sprintf("Error while updating farm %s", err)
					dialog.ShowError(fmt.Errorf(errorsFarmLabelUpdate.Text), myWindow)
				}
				if farm, err := updateFarm(expclient, farmsListData[farmToEditIdx].ID, int64(farmOwnerIDAsInt), farmNameInputUpdate.Text, emailInput.Text, tftAddressInputUpdate.Text, threebotId); err == nil {

					infoFarmLabelUpdate.Text = fmt.Sprintf("farm with ID %d is updated", farm.ID)
					dialog.ShowInformation("Farm updated!", infoFarmLabelUpdate.Text, myWindow)
					farmsListData, farmsNames = ListAllFarmsAndNames(expclient, int64(threebotId))
					farmsBinding.Set(farmsNames)
				} else {
					errorsFarmLabelUpdate.Text = fmt.Sprintf("Error while updating farm %s", err)
					dialog.ShowError(fmt.Errorf(errorsFarmLabelUpdate.Text), myWindow)
				}
				log.Println(errs)
			}
		},
	}

	if expclient != nil {
		farmsListData, farmsNames = ListAllFarmsAndNames(expclient, int64(threebotId))
		farmsBinding.Set(farmsNames)
	}

	farmsList := widget.NewListWithData(farmsBinding,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		})

	farmsList.OnSelected = func(id widget.ListItemID) {
		formFarmUpdate.Show()
		farmToEditIdx = int64(id)
		if id >= len(farmsListData) {

			return
		}
		farmNameInputUpdate.SetText(farmsListData[id].Name)
		for _, x := range farmsListData[id].WalletAddresses {
			if x.Address != "" && x.Asset != "" {
				tftAddressInputUpdate.SetText(x.Address)
				break
			}
		}
		farmOwnerIdEntry.SetText(fmt.Sprintf("%d", farmsListData[id].ThreebotID))
		farmIdEntryUpdate.SetText(fmt.Sprintf("%d", farmsListData[id].ID))
		nodesListData, nodesNames = ListAllNodesAndNames(expclient, farmsListData[id].ID)
		nodesBinding.Set(nodesNames)

	}
	scrolledFarmsList := container.NewVScroll(farmsList)
	scrolledFarmsList.SetMinSize(fyne.NewSize(100, 300))

	nodeDetailsLayout := container.NewVScroll(container.New(layout.NewFormLayout(),
		widget.NewLabel("Node ID"), nodeId,
		widget.NewLabel("Node Version"), nodeVersion,
		widget.NewLabel("Hostname"), nodeHostName,
		widget.NewLabel("Farm name"), nodeFarmerName,
		widget.NewLabel("Location"), nodeLocation,
		widget.NewLabel("Uptime"), nodeUptime,
		widget.NewLabel("CRU"), nodeCRU,
		widget.NewLabel("MRU"), nodeMRU,
		widget.NewLabel("HRU"), nodeHRU,
		widget.NewLabel("SRU"), nodeSRU,
	))
	nodeDetailsLayout.Hide()

	nodesList := widget.NewListWithData(nodesBinding,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		})
	nodesList.OnSelected = func(id widget.ListItemID) {
		nodeDetailsLayout.Show()
		nodeIdx := int64(id)

		if id > len(nodesNames) {
			return
		}
		nodeSelected := nodesListData[nodeIdx]
		nodeId.SetText(nodeSelected.NodeId)
		nodeVersion.SetText(nodeSelected.OsVersion)
		nodeHostName.SetText(nodeSelected.HostName)
		nodeLocation.Text = ""
		if nodeSelected.Location.Country != "" {
			nodeLocation.Text += nodeSelected.Location.Country
		}
		if nodeSelected.Location.City != "" {
			nodeLocation.Text += " - " + nodeSelected.Location.City
		}
		nodeLocation.Refresh()
		nodeCRU.SetText(fmt.Sprintf("%d", nodeSelected.TotalResources.Cru))
		nodeMRU.SetText(fmt.Sprintf("%f", nodeSelected.TotalResources.Mru))
		nodeSRU.SetText(fmt.Sprintf("%f", nodeSelected.TotalResources.Sru))
		nodeHRU.SetText(fmt.Sprintf("%f", nodeSelected.TotalResources.Hru))
		t := time.Unix(nodeSelected.Uptime, 0)

		nodeUptime.SetText(fmt.Sprintf("%s", humanize.Time(t)))

	}
	scrolledNodesList := container.NewVScroll(nodesList)
	nodeDetailsLayout.SetMinSize(fyne.NewSize(100, 400))

	scrolledNodesCont := container.NewVSplit(scrolledNodesList, nodeDetailsLayout)

	// contScrolledList := container.NewVBox(container.NewPadded(), scrolledFarmsList)
	scolledFarmsListCont := container.NewVSplit(scrolledFarmsList, formFarmUpdate)

	formFarmUpdate.Resize(fyne.NewSize(700, 400))
	formFarmUpdate.Hide()
	contFarmsList := container.NewHSplit(scolledFarmsListCont, scrolledNodesCont)

	themes := fyne.NewContainerWithLayout(layout.NewGridLayout(2),
		widget.NewButton("Dark", func() {
			fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
		}),
		widget.NewButton("Light", func() {
			fyne.CurrentApp().Settings().SetTheme(theme.LightTheme())
		}),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Identity", formIdentity),
		container.NewTabItem("Register Farm", formFarm),
		container.NewTabItem("Farms", contFarmsList),
		container.NewTabItem("Settings", themes),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	myWindow.SetContent(tabs)
	myWindow.Resize(fyne.NewSize(800, 600))

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
	addresses := make([]WalletAddress, 0)
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

func updateFarm(expclient *Client, farmId, ownerId int64, name, email, tftAddress string, tid int) (Farm, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	tftAddress = strings.TrimSpace(tftAddress)
	addresses := make([]WalletAddress, 0)
	address := WalletAddress{Address: tftAddress, Asset: "TFT"}
	addresses = append(addresses, address)
	farm := Farm{
		Name:            name,
		ID:              farmId,
		ThreebotID:      ownerId,
		Email:           email,
		WalletAddresses: addresses,
	}
	err := expclient.Directory.FarmUpdate(farm)
	if err != nil {
		fmt.Println("err:", err)
		return farm, err
	}
	return farm, nil
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
			fmt.Println("user exists an matches explorer registered user pubkey")

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
func ListAllFarmsAndNames(expclient *Client, tid int64) ([]Farm, []string) {
	farmsRet := make([]Farm, 0)
	farmsNames := make([]string, 0)
	pageNumber := 1

	for {
		pager := Page(pageNumber, 20)
		farms, err := expclient.Directory.FarmList(tid, "", pager)
		farmsRet = append(farmsRet, farms...)
		if err != nil {
			break
		}
		if len(farms) == 0 {
			break
		}
		pageNumber++
	}
	for _, f := range farmsRet {
		farmsNames = append(farmsNames, f.Name)
	}
	return farmsRet, farmsNames

}

func ListAllNodesAndNames(expclient *Client, farmId int64) ([]Node, []string) {
	nodesRet := make([]Node, 0)
	nodesNames := make([]string, 0)
	pageNumber := 1

	filter := NodeFilter{}
	filter = filter.WithFarm(farmId)
	for {
		pager := Page(pageNumber, 20)
		nodes, err := expclient.Directory.NodeList(filter, pager)
		nodesRet = append(nodesRet, nodes...)
		if err != nil {
			break
		}
		if len(nodes) == 0 {
			break
		}
		pageNumber++
	}
	for _, n := range nodesRet {
		nodesNames = append(nodesNames, n.NodeId)
	}
	return nodesRet, nodesNames

}
